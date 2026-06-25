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

package networking

import (
	"context"
	"fmt"

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

// ServiceReconciler reconciles Service resources for a Networking object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ServiceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var net calypsov1alpha1.Networking

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&net,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if net.Spec.TargetRef.Resource == "" {

		log.Info("Networking targetRef is nil, skipping reconciliation", "name", net.Name)

		return ctrl.Result{}, nil
	}

	svcName := net.Spec.TargetRef.Resource

	targetSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: net.Namespace,
		},
	}

	if len(net.Spec.Service.Ports) == 0 {

		var svc corev1.Service

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: net.Namespace, Name: svcName},
			&svc,
		); err == nil {

			if err := r.Delete(
				ctx,
				&svc,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetSvc,
			func() error {
				targetSvc.Spec.Selector = map[string]string{
					"app": net.Spec.TargetRef.Resource,
				}
				targetSvc.Spec.Ports = net.Spec.Service.Ports
				targetSvc.Spec.Type = net.Spec.Service.Type

				if err := ctrl.SetControllerReference(
					&net,
					targetSvc,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedSvcObj, err := patch.ApplyEscapeHatches(
					targetSvc,
					net.Spec.EscapeHatches,
					"Service",
				)

				if err != nil {
					return fmt.Errorf("failed to apply escape hatch to Service: %w", err)
				}

				*targetSvc = *(patchedSvcObj.(*corev1.Service))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Networking{}).
		Owns(&corev1.Service{}).
		Named("networking-service").
		Complete(r)
}
