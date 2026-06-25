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

// NetworkingReconciler reconciles Networking CRs for a Workload object
type NetworkingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NetworkingReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

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

	return ctrl.Result{}, nil
}

func (r *NetworkingReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Workload{}).
		Owns(&calypsov1alpha1.Networking{}).
		Named("workload-networking").
		Complete(r)
}
