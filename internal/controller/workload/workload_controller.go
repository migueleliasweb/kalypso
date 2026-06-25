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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
)

// WorkloadReconciler reconciles a Workload object
type WorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}


func (r *WorkloadReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	// 1. Fetch the Workload
	var workload calypsov1alpha1.Workload

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&workload,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	targetNamespace := workload.Spec.TargetRef.Namespace

	if targetNamespace == "" {

		log.Info("Target namespace is empty, skipping reconciliation", "workload", workload.Name)

		return ctrl.Result{}, nil
	}

	// 2. Reconcile Compute Capability
	if err := r.reconcileCompute(
		ctx,
		&workload,
		targetNamespace,
	); err != nil {
		log.Error(err, "Failed to reconcile Compute capability")

		return ctrl.Result{}, err
	}

	// 3. Reconcile Storage Capability
	if err := r.reconcileStorage(
		ctx,
		&workload,
		targetNamespace,
	); err != nil {
		log.Error(err, "Failed to reconcile Storage capability")

		return ctrl.Result{}, err
	}

	// 4. Reconcile Networking Capability
	if err := r.reconcileNetworking(
		ctx,
		&workload,
		targetNamespace,
	); err != nil {
		log.Error(err, "Failed to reconcile Networking capability")

		return ctrl.Result{}, err
	}

	// 5. Reconcile Observability Capability
	if err := r.reconcileObservability(
		ctx,
		&workload,
		targetNamespace,
	); err != nil {
		log.Error(err, "Failed to reconcile Observability capability")

		return ctrl.Result{}, err
	}

	// 6. Reconcile Security Capability
	if err := r.reconcileSecurity(
		ctx,
		&workload,
		targetNamespace,
	); err != nil {
		log.Error(err, "Failed to reconcile Security capability")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) reconcileCompute(
	ctx context.Context,
	workload *calypsov1alpha1.Workload,
	ns string,
) error {

	name := fmt.Sprintf("%s-compute", workload.Name)

	var compute calypsov1alpha1.Compute

	exists := true

	if err := r.Get(
		ctx,
		client.ObjectKey{Namespace: ns, Name: name},
		&compute,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		exists = false
	}

	if !exists {

		if workload.Spec.Compute.TargetRef.Resource == "" {
			return nil
		}

		// Create it
		newCompute := &calypsov1alpha1.Compute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: workload.Spec.Compute,
		}

		newCompute.Spec.TargetRef = workload.Spec.TargetRef

		if err := ctrl.SetControllerReference(
			workload,
			newCompute,
			r.Scheme,
		); err != nil {
			return fmt.Errorf("failed to set controller reference on compute: %w", err)
		}

		return r.Create(
			ctx,
			newCompute,
		)
	}

	if workload.Spec.Compute.TargetRef.Resource == "" {
		return r.Delete(
			ctx,
			&compute,
		)
	}

	compute.Spec = workload.Spec.Compute
	compute.Spec.TargetRef = workload.Spec.TargetRef

	return r.Update(
		ctx,
		&compute,
	)
}

func (r *WorkloadReconciler) reconcileStorage(
	ctx context.Context,
	workload *calypsov1alpha1.Workload,
	ns string,
) error {

	name := fmt.Sprintf("%s-storage", workload.Name)

	var storage calypsov1alpha1.Storage

	exists := true

	if err := r.Get(
		ctx,
		client.ObjectKey{Namespace: ns, Name: name},
		&storage,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		exists = false
	}

	if !exists {

		if workload.Spec.Storage.TargetRef.Resource == "" {
			return nil
		}

		newStorage := &calypsov1alpha1.Storage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: workload.Spec.Storage,
		}

		newStorage.Spec.TargetRef = workload.Spec.TargetRef

		if err := ctrl.SetControllerReference(
			workload,
			newStorage,
			r.Scheme,
		); err != nil {
			return fmt.Errorf("failed to set controller reference on storage: %w", err)
		}

		return r.Create(
			ctx,
			newStorage,
		)
	}

	if workload.Spec.Storage.TargetRef.Resource == "" {
		return r.Delete(
			ctx,
			&storage,
		)
	}

	storage.Spec = workload.Spec.Storage
	storage.Spec.TargetRef = workload.Spec.TargetRef

	return r.Update(
		ctx,
		&storage,
	)
}

func (r *WorkloadReconciler) reconcileNetworking(
	ctx context.Context,
	workload *calypsov1alpha1.Workload,
	ns string,
) error {

	name := fmt.Sprintf("%s-networking", workload.Name)

	var networking calypsov1alpha1.Networking

	exists := true

	if err := r.Get(
		ctx,
		client.ObjectKey{Namespace: ns, Name: name},
		&networking,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		exists = false
	}

	if !exists {

		if workload.Spec.Networking.TargetRef.Resource == "" {
			return nil
		}

		newNetworking := &calypsov1alpha1.Networking{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: workload.Spec.Networking,
		}

		newNetworking.Spec.TargetRef = workload.Spec.TargetRef

		if err := ctrl.SetControllerReference(
			workload,
			newNetworking,
			r.Scheme,
		); err != nil {
			return fmt.Errorf("failed to set controller reference on networking: %w", err)
		}

		return r.Create(
			ctx,
			newNetworking,
		)
	}

	if workload.Spec.Networking.TargetRef.Resource == "" {
		return r.Delete(
			ctx,
			&networking,
		)
	}

	networking.Spec = workload.Spec.Networking
	networking.Spec.TargetRef = workload.Spec.TargetRef

	return r.Update(
		ctx,
		&networking,
	)
}

func (r *WorkloadReconciler) reconcileObservability(
	ctx context.Context,
	workload *calypsov1alpha1.Workload,
	ns string,
) error {

	name := fmt.Sprintf("%s-observability", workload.Name)

	var observability calypsov1alpha1.Observability

	exists := true

	if err := r.Get(
		ctx,
		client.ObjectKey{Namespace: ns, Name: name},
		&observability,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		exists = false
	}

	if !exists {

		if workload.Spec.Observability.TargetRef.Resource == "" {
			return nil
		}

		newObservability := &calypsov1alpha1.Observability{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: workload.Spec.Observability,
		}

		newObservability.Spec.TargetRef = workload.Spec.TargetRef

		if err := ctrl.SetControllerReference(
			workload,
			newObservability,
			r.Scheme,
		); err != nil {
			return fmt.Errorf("failed to set controller reference on observability: %w", err)
		}

		return r.Create(
			ctx,
			newObservability,
		)
	}

	if workload.Spec.Observability.TargetRef.Resource == "" {
		return r.Delete(
			ctx,
			&observability,
		)
	}

	observability.Spec = workload.Spec.Observability
	observability.Spec.TargetRef = workload.Spec.TargetRef

	return r.Update(
		ctx,
		&observability,
	)
}

func (r *WorkloadReconciler) reconcileSecurity(
	ctx context.Context,
	workload *calypsov1alpha1.Workload,
	ns string,
) error {

	name := fmt.Sprintf("%s-security", workload.Name)

	var security calypsov1alpha1.Security

	exists := true

	if err := r.Get(
		ctx,
		client.ObjectKey{Namespace: ns, Name: name},
		&security,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		exists = false
	}

	if !exists {

		if workload.Spec.Security.TargetRef.Resource == "" {
			return nil
		}

		newSecurity := &calypsov1alpha1.Security{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: workload.Spec.Security,
		}

		newSecurity.Spec.TargetRef = workload.Spec.TargetRef

		if err := ctrl.SetControllerReference(
			workload,
			newSecurity,
			r.Scheme,
		); err != nil {
			return fmt.Errorf("failed to set controller reference on security: %w", err)
		}

		return r.Create(
			ctx,
			newSecurity,
		)
	}

	if workload.Spec.Security.TargetRef.Resource == "" {
		return r.Delete(
			ctx,
			&security,
		)
	}

	security.Spec = workload.Spec.Security
	security.Spec.TargetRef = workload.Spec.TargetRef

	return r.Update(
		ctx,
		&security,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Workload{}).
		Owns(&calypsov1alpha1.Compute{}).
		Owns(&calypsov1alpha1.Storage{}).
		Owns(&calypsov1alpha1.Networking{}).
		Owns(&calypsov1alpha1.Observability{}).
		Owns(&calypsov1alpha1.Security{}).
		Named("workload").
		Complete(r)
}
