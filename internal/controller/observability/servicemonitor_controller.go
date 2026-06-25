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

// ServiceMonitorReconciler reconciles ServiceMonitor resources for an Observability object
type ServiceMonitorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ServiceMonitorReconciler) Reconcile(
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

	smName := fmt.Sprintf("%s-servicemonitor", obs.Name)

	targetSM := &unstructured.Unstructured{}

	targetSM.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})

	targetSM.SetName(smName)
	targetSM.SetNamespace(obs.Namespace)

	if !obs.Spec.ServiceMonitor.Enabled {

		var sm unstructured.Unstructured

		sm.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "monitoring.coreos.com",
			Version: "v1",
			Kind:    "ServiceMonitor",
		})

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: obs.Namespace, Name: smName},
			&sm,
		); err == nil {

			if err := r.Delete(
				ctx,
				&sm,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetSM,
			func() error {
				path := "/metrics"

				if obs.Spec.ServiceMonitor.Path != "" {
					path = obs.Spec.ServiceMonitor.Path
				}

				interval := obs.Spec.ServiceMonitor.Interval

				if interval == "" {
					interval = "30s"
				}

				spec := map[string]interface{}{
					"selector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"app": obs.Spec.TargetRef.Resource,
						},
					},
					"endpoints": []interface{}{
						map[string]interface{}{
							"path":     path,
							"interval": interval,
						},
					},
				}

				targetSM.Object["spec"] = spec

				if err := ctrl.SetControllerReference(
					&obs,
					targetSM,
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

func (r *ServiceMonitorReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Observability{}).
		Named("observability-servicemonitor").
		Complete(r)
}
