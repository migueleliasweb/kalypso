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

package observability

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
)

// PodMonitorReconciler reconciles PodMonitor resources for an Observability object
type PodMonitorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PodMonitorReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var obs calypsov1alpha1.Observability

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&obs,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if obs.Spec.TargetRef.Resource == "" {

		log.Info("Observability targetRef is nil, skipping reconciliation", "name", obs.Name)

		return ctrl.Result{}, nil
	}

	pmName := fmt.Sprintf("%s-podmonitor", obs.Name)

	targetPM := &unstructured.Unstructured{}

	targetPM.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PodMonitor",
	})

	targetPM.SetName(pmName)
	targetPM.SetNamespace(obs.Namespace)

	if !obs.Spec.PodMonitor.Enabled {

		var pm unstructured.Unstructured

		pm.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "monitoring.coreos.com",
			Version: "v1",
			Kind:    "PodMonitor",
		})

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: obs.Namespace, Name: pmName},
			&pm,
		); err == nil {

			if err := r.Delete(
				ctx,
				&pm,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetPM,
			func() error {
				path := "/metrics"

				if obs.Spec.PodMonitor.Path != "" {
					path = obs.Spec.PodMonitor.Path
				}

				interval := obs.Spec.PodMonitor.Interval

				if interval == "" {
					interval = "30s"
				}

				spec := map[string]interface{}{
					"selector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"app": obs.Spec.TargetRef.Resource,
						},
					},
					"podMetricsEndpoints": []interface{}{
						map[string]interface{}{
							"path":     path,
							"interval": interval,
						},
					},
				}

				targetPM.Object["spec"] = spec

				if err := ctrl.SetControllerReference(
					&obs,
					targetPM,
					r.Scheme,
				); err != nil {
					return err
				}

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *PodMonitorReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Observability{}).
		Named("observability-podmonitor").
		Complete(r)
}
