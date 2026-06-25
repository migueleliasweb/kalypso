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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
)

// ObservabilityReconciler reconciles a Observability object
type ObservabilityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=observabilities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=observabilities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=observabilities/finalizers,verbs=update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop.
func (r *ObservabilityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Observability resource
	var obs calypsov1alpha1.Observability
	if err := r.Get(ctx, req.NamespacedName, &obs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if obs.Spec.TargetRef == nil {
		log.Info("Observability targetRef is nil, skipping reconciliation", "name", obs.Name)
		return ctrl.Result{}, nil
	}

	// Reconcile ServiceMonitor if enabled
	if obs.Spec.ServiceMonitor != nil && obs.Spec.ServiceMonitor.Enabled {
		if err := r.reconcileServiceMonitor(ctx, &obs); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile PodMonitor if enabled
	if obs.Spec.PodMonitor != nil && obs.Spec.PodMonitor.Enabled {
		if err := r.reconcilePodMonitor(ctx, &obs); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ObservabilityReconciler) reconcileServiceMonitor(ctx context.Context, obs *calypsov1alpha1.Observability) error {
	name := fmt.Sprintf("%s-servicemonitor", obs.Name)
	sm := &unstructured.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})

	err := r.Get(ctx, client.ObjectKey{Namespace: obs.Namespace, Name: name}, sm)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	targetSM := &unstructured.Unstructured{}
	targetSM.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})
	targetSM.SetName(name)
	targetSM.SetNamespace(obs.Namespace)

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

	if err := ctrl.SetControllerReference(obs, targetSM, r.Scheme); err != nil {
		return err
	}

	if apierrors.IsNotFound(err) {
		return r.Create(ctx, targetSM)
	}
	targetSM.SetResourceVersion(sm.GetResourceVersion())
	return r.Update(ctx, targetSM)
}

func (r *ObservabilityReconciler) reconcilePodMonitor(ctx context.Context, obs *calypsov1alpha1.Observability) error {
	name := fmt.Sprintf("%s-podmonitor", obs.Name)
	pm := &unstructured.Unstructured{}
	pm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PodMonitor",
	})

	err := r.Get(ctx, client.ObjectKey{Namespace: obs.Namespace, Name: name}, pm)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	targetPM := &unstructured.Unstructured{}
	targetPM.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PodMonitor",
	})
	targetPM.SetName(name)
	targetPM.SetNamespace(obs.Namespace)

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

	if err := ctrl.SetControllerReference(obs, targetPM, r.Scheme); err != nil {
		return err
	}

	if apierrors.IsNotFound(err) {
		return r.Create(ctx, targetPM)
	}
	targetPM.SetResourceVersion(pm.GetResourceVersion())
	return r.Update(ctx, targetPM)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Observability{}).
		Named("observability").
		Complete(r)
}
