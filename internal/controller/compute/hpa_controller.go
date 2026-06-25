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

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// HPAReconciler reconciles HPA resources for a Compute object
type HPAReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *HPAReconciler) Reconcile(
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

	hpaName := fmt.Sprintf("%s-hpa", compute.Name)

	targetHPA := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpaName,
			Namespace: compute.Namespace,
		},
	}

	if compute.Spec.Autoscaling.MaxReplicas == 0 {

		var hpa autoscalingv2.HorizontalPodAutoscaler

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: compute.Namespace, Name: hpaName},
			&hpa,
		); err == nil {

			if err := r.Delete(
				ctx,
				&hpa,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetHPA,
			func() error {
				targetHPA.Spec.ScaleTargetRef = autoscalingv2.CrossVersionObjectReference{
					APIVersion: compute.Spec.TargetRef.ApiVersion,
					Kind:       compute.Spec.TargetRef.Kind,
					Name:       compute.Spec.TargetRef.Resource,
				}

				targetHPA.Spec.MaxReplicas = compute.Spec.Autoscaling.MaxReplicas

				if compute.Spec.Autoscaling.MinReplicas > 0 {
					minReps := compute.Spec.Autoscaling.MinReplicas
					targetHPA.Spec.MinReplicas = &minReps
				} else {
					targetHPA.Spec.MinReplicas = nil
				}

				var metrics []autoscalingv2.MetricSpec

				if compute.Spec.Autoscaling.TargetCPUUtilizationPercentage > 0 {
					cpuVal := compute.Spec.Autoscaling.TargetCPUUtilizationPercentage

					metrics = append(metrics, autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &cpuVal,
							},
						},
					})

				}

				if compute.Spec.Autoscaling.TargetMemoryUtilizationPercentage > 0 {
					memVal := compute.Spec.Autoscaling.TargetMemoryUtilizationPercentage

					metrics = append(metrics, autoscalingv2.MetricSpec{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceMemory,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &memVal,
							},
						},
					})

				}

				targetHPA.Spec.Metrics = metrics

				if err := ctrl.SetControllerReference(
					&compute,
					targetHPA,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedHPAObj, err := patch.ApplyEscapeHatches(
					targetHPA,
					compute.Spec.EscapeHatches,
					"HorizontalPodAutoscaler",
				)

				if err != nil {
					return err
				}

				*targetHPA = *(patchedHPAObj.(*autoscalingv2.HorizontalPodAutoscaler))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *HPAReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Compute{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Named("compute-hpa").
		Complete(r)
}
