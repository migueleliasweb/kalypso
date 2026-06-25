/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// ComputeReconciler reconciles a Compute object
type ComputeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop.
func (r *ComputeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the Compute resource
	var compute calypsov1alpha1.Compute
	if err := r.Get(ctx, req.NamespacedName, &compute); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if compute.Spec.TargetRef == nil {
		log.Info("Compute targetRef is nil, skipping reconciliation", "name", compute.Name)
		return ctrl.Result{}, nil
	}

	// 2. Fetch the target Deployment (currently the only supported kind)
	var deploy appsv1.Deployment
	var deployFound bool
	if compute.Spec.TargetRef.Kind == "Deployment" {
		err := r.Get(ctx, client.ObjectKey{Namespace: compute.Namespace, Name: compute.Spec.TargetRef.Resource}, &deploy)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			log.Info("Target Deployment not found", "name", compute.Spec.TargetRef.Resource)
		} else {
			deployFound = true
		}
	} else {
		log.Info("Unsupported target resource Kind", "kind", compute.Spec.TargetRef.Kind)
	}

	// 3. Patch target Deployment scheduling if found
	if deployFound && compute.Spec.Scheduling != nil {
		originalDeploy := deploy.DeepCopy()
		deploy.Spec.Template.Spec.NodeSelector = compute.Spec.Scheduling.NodeSelector
		deploy.Spec.Template.Spec.Affinity = compute.Spec.Scheduling.Affinity
		deploy.Spec.Template.Spec.TopologySpreadConstraints = compute.Spec.Scheduling.TopologySpreadConstraints

		// Apply escape hatch to Deployment before updating
		patchedDeployObj, err := patch.ApplyEscapeHatches(&deploy, compute.Spec.EscapeHatches, "Deployment")
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to apply escape hatch to deployment: %w", err)
		}
		deploy = *(patchedDeployObj.(*appsv1.Deployment))

		// Apply patch using Client.Patch
		if err := r.Patch(ctx, &deploy, client.MergeFrom(originalDeploy)); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch deployment scheduling: %w", err)
		}
	}

	// 4. Reconcile HPA
	if err := r.reconcileHPA(ctx, &compute); err != nil {
		return ctrl.Result{}, err
	}

	// 5. Reconcile PDB
	if err := r.reconcilePDB(ctx, &compute, &deploy, deployFound); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ComputeReconciler) reconcileHPA(ctx context.Context, compute *calypsov1alpha1.Compute) error {
	hpaName := fmt.Sprintf("%s-hpa", compute.Name)
	var hpa autoscalingv2.HorizontalPodAutoscaler
	hpaErr := r.Get(ctx, client.ObjectKey{Namespace: compute.Namespace, Name: hpaName}, &hpa)

	if compute.Spec.Autoscaling == nil {
		if hpaErr == nil {
			return r.Delete(ctx, &hpa)
		}
		return nil
	}

	targetHPA := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpaName,
			Namespace: compute.Namespace,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: compute.Spec.TargetRef.ApiVersion,
				Kind:       compute.Spec.TargetRef.Kind,
				Name:       compute.Spec.TargetRef.Resource,
			},
			MaxReplicas: compute.Spec.Autoscaling.MaxReplicas,
		},
	}
	if compute.Spec.Autoscaling.MinReplicas != nil {
		targetHPA.Spec.MinReplicas = compute.Spec.Autoscaling.MinReplicas
	}

	var metrics []autoscalingv2.MetricSpec
	if compute.Spec.Autoscaling.TargetCPUUtilizationPercentage != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: compute.Spec.Autoscaling.TargetCPUUtilizationPercentage,
				},
			},
		})
	}
	if compute.Spec.Autoscaling.TargetMemoryUtilizationPercentage != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: compute.Spec.Autoscaling.TargetMemoryUtilizationPercentage,
				},
			},
		})
	}
	targetHPA.Spec.Metrics = metrics

	if err := ctrl.SetControllerReference(compute, targetHPA, r.Scheme); err != nil {
		return err
	}

	patchedHPAObj, err := patch.ApplyEscapeHatches(targetHPA, compute.Spec.EscapeHatches, "HorizontalPodAutoscaler")
	if err != nil {
		return fmt.Errorf("failed to apply escape hatch to HPA: %w", err)
	}
	targetHPA = patchedHPAObj.(*autoscalingv2.HorizontalPodAutoscaler)

	if hpaErr != nil {
		if apierrors.IsNotFound(hpaErr) {
			return r.Create(ctx, targetHPA)
		}
		return hpaErr
	}

	targetHPA.ResourceVersion = hpa.ResourceVersion
	return r.Update(ctx, targetHPA)
}

func (r *ComputeReconciler) reconcilePDB(ctx context.Context, compute *calypsov1alpha1.Compute, deploy *appsv1.Deployment, deployFound bool) error {
	pdbName := fmt.Sprintf("%s-pdb", compute.Name)
	var pdb policyv1.PodDisruptionBudget
	pdbErr := r.Get(ctx, client.ObjectKey{Namespace: compute.Namespace, Name: pdbName}, &pdb)

	if compute.Spec.PodDisruptionBudget == nil {
		if pdbErr == nil {
			return r.Delete(ctx, &pdb)
		}
		return nil
	}

	var selector *metav1.LabelSelector
	if deployFound && deploy.Spec.Selector != nil {
		selector = deploy.Spec.Selector
	} else {
		selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": compute.Spec.TargetRef.Resource,
			},
		}
	}

	targetPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pdbName,
			Namespace: compute.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: selector,
		},
	}
	if compute.Spec.PodDisruptionBudget.MinAvailable != nil {
		targetPDB.Spec.MinAvailable = compute.Spec.PodDisruptionBudget.MinAvailable
	}
	if compute.Spec.PodDisruptionBudget.MaxUnavailable != nil {
		targetPDB.Spec.MaxUnavailable = compute.Spec.PodDisruptionBudget.MaxUnavailable
	}

	if err := ctrl.SetControllerReference(compute, targetPDB, r.Scheme); err != nil {
		return err
	}

	patchedPDBObj, err := patch.ApplyEscapeHatches(targetPDB, compute.Spec.EscapeHatches, "PodDisruptionBudget")
	if err != nil {
		return fmt.Errorf("failed to apply escape hatch to PDB: %w", err)
	}
	targetPDB = patchedPDBObj.(*policyv1.PodDisruptionBudget)

	if pdbErr != nil {
		if apierrors.IsNotFound(pdbErr) {
			return r.Create(ctx, targetPDB)
		}
		return pdbErr
	}

	targetPDB.ResourceVersion = pdb.ResourceVersion
	return r.Update(ctx, targetPDB)
}

func (r *ComputeReconciler) findComputesForDeployment(ctx context.Context, obj client.Object) []reconcile.Request {
	deploy, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}

	var list calypsov1alpha1.ComputeList
	if err := r.List(ctx, &list, client.InNamespace(deploy.Namespace)); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, compute := range list.Items {
		if compute.Spec.TargetRef != nil &&
			compute.Spec.TargetRef.Kind == "Deployment" &&
			compute.Spec.TargetRef.Resource == deploy.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: compute.Namespace,
					Name:      compute.Name,
				},
			})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *ComputeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Compute{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.findComputesForDeployment),
		).
		Named("compute").
		Complete(r)
}
