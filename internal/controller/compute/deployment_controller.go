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

package compute

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// DeploymentReconciler reconciles target Deployment scheduling for a Compute object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var compute calypsov1alpha1.Compute

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&compute,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if compute.Spec.TargetRef.Resource == "" {

		log.Info("Compute targetRef is nil, skipping reconciliation", "name", compute.Name)

		return ctrl.Result{}, nil
	}

	var deploy appsv1.Deployment

	var deployFound bool

	if compute.Spec.TargetRef.Kind == "Deployment" {

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: compute.Namespace, Name: compute.Spec.TargetRef.Resource},
			&deploy,
		); err != nil {
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

	hasScheduling := len(compute.Spec.Scheduling.NodeSelector) > 0 ||
		compute.Spec.Scheduling.Affinity.NodeAffinity != nil ||
		compute.Spec.Scheduling.Affinity.PodAffinity != nil ||
		compute.Spec.Scheduling.Affinity.PodAntiAffinity != nil ||
		len(compute.Spec.Scheduling.TopologySpreadConstraints) > 0

	if deployFound && hasScheduling {

		originalDeploy := deploy.DeepCopy()

		deploy.Spec.Template.Spec.NodeSelector = compute.Spec.Scheduling.NodeSelector
		deploy.Spec.Template.Spec.Affinity = &compute.Spec.Scheduling.Affinity
		deploy.Spec.Template.Spec.TopologySpreadConstraints = compute.Spec.Scheduling.TopologySpreadConstraints

		patchedDeployObj, err := patch.ApplyEscapeHatches(
			&deploy,
			compute.Spec.EscapeHatches,
			"Deployment",
		)

		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to apply escape hatch to deployment: %w", err)
		}

		deploy = *(patchedDeployObj.(*appsv1.Deployment))

		if err := r.Patch(
			ctx,
			&deploy,
			client.MergeFrom(originalDeploy),
		); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch deployment scheduling: %w", err)
		}

	}

	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) findComputesForDeployment(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {

	deploy, ok := obj.(*appsv1.Deployment)

	if !ok {
		return nil
	}

	var list calypsov1alpha1.ComputeList

	if err := r.List(
		ctx,
		&list,
		client.InNamespace(deploy.Namespace),
	); err != nil {
		return nil
	}

	var requests []reconcile.Request

	for _, compute := range list.Items {
		if compute.Spec.TargetRef.Resource != "" &&
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

func (r *DeploymentReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Compute{}).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.findComputesForDeployment),
		).
		Named("compute-deployment").
		Complete(r)
}
