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

package workload

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	computeName := fmt.Sprintf("%s-compute", workload.Name)

	targetCompute := &calypsov1alpha1.Compute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      computeName,
			Namespace: targetNamespace,
		},
	}

	if workload.Spec.Compute.TargetRef.Resource == "" {

		var compute calypsov1alpha1.Compute

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: targetNamespace, Name: computeName},
			&compute,
		); err == nil {

			if err := r.Delete(
				ctx,
				&compute,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetCompute,
			func() error {
				targetCompute.Spec = workload.Spec.Compute
				targetCompute.Spec.TargetRef = workload.Spec.TargetRef

				if err := ctrl.SetControllerReference(
					&workload,
					targetCompute,
					r.Scheme,
				); err != nil {
					return fmt.Errorf("failed to set controller reference on compute: %w", err)
				}

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	// 3. Reconcile Storage Capability
	storageName := fmt.Sprintf("%s-storage", workload.Name)

	targetStorage := &calypsov1alpha1.Storage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageName,
			Namespace: targetNamespace,
		},
	}

	if workload.Spec.Storage.TargetRef.Resource == "" {

		var storage calypsov1alpha1.Storage

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: targetNamespace, Name: storageName},
			&storage,
		); err == nil {

			if err := r.Delete(
				ctx,
				&storage,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetStorage,
			func() error {
				targetStorage.Spec = workload.Spec.Storage
				targetStorage.Spec.TargetRef = workload.Spec.TargetRef

				if err := ctrl.SetControllerReference(
					&workload,
					targetStorage,
					r.Scheme,
				); err != nil {
					return fmt.Errorf("failed to set controller reference on storage: %w", err)
				}

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	// 4. Reconcile Networking Capability
	networkingName := fmt.Sprintf("%s-networking", workload.Name)

	targetNetworking := &calypsov1alpha1.Networking{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkingName,
			Namespace: targetNamespace,
		},
	}

	if workload.Spec.Networking.TargetRef.Resource == "" {

		var networking calypsov1alpha1.Networking

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: targetNamespace, Name: networkingName},
			&networking,
		); err == nil {

			if err := r.Delete(
				ctx,
				&networking,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetNetworking,
			func() error {
				targetNetworking.Spec = workload.Spec.Networking
				targetNetworking.Spec.TargetRef = workload.Spec.TargetRef

				if err := ctrl.SetControllerReference(
					&workload,
					targetNetworking,
					r.Scheme,
				); err != nil {
					return fmt.Errorf("failed to set controller reference on networking: %w", err)
				}

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	// 5. Reconcile Observability Capability
	observabilityName := fmt.Sprintf("%s-observability", workload.Name)

	targetObservability := &calypsov1alpha1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityName,
			Namespace: targetNamespace,
		},
	}

	if workload.Spec.Observability.TargetRef.Resource == "" {

		var observability calypsov1alpha1.Observability

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: targetNamespace, Name: observabilityName},
			&observability,
		); err == nil {

			if err := r.Delete(
				ctx,
				&observability,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetObservability,
			func() error {
				targetObservability.Spec = workload.Spec.Observability
				targetObservability.Spec.TargetRef = workload.Spec.TargetRef

				if err := ctrl.SetControllerReference(
					&workload,
					targetObservability,
					r.Scheme,
				); err != nil {
					return fmt.Errorf("failed to set controller reference on observability: %w", err)
				}

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	// 6. Reconcile Security Capability
	securityName := fmt.Sprintf("%s-security", workload.Name)

	targetSecurity := &calypsov1alpha1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name:      securityName,
			Namespace: targetNamespace,
		},
	}

	if workload.Spec.Security.TargetRef.Resource == "" {

		var security calypsov1alpha1.Security

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: targetNamespace, Name: securityName},
			&security,
		); err == nil {

			if err := r.Delete(
				ctx,
				&security,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetSecurity,
			func() error {
				targetSecurity.Spec = workload.Spec.Security
				targetSecurity.Spec.TargetRef = workload.Spec.TargetRef

				if err := ctrl.SetControllerReference(
					&workload,
					targetSecurity,
					r.Scheme,
				); err != nil {
					return fmt.Errorf("failed to set controller reference on security: %w", err)
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
