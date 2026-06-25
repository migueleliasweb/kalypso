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

package security

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// RoleReconciler reconciles Role resources for a Security object
type RoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *RoleReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var sec calypsov1alpha1.Security

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&sec,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if sec.Spec.TargetRef.Resource == "" {

		log.Info("Security targetRef is nil, skipping reconciliation", "name", sec.Name)

		return ctrl.Result{}, nil
	}

	roleName := fmt.Sprintf("%s-role", sec.Name)

	targetRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: sec.Namespace,
		},
	}

	if !sec.Spec.RBAC.CreateRole {

		var role rbacv1.Role

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: sec.Namespace, Name: roleName},
			&role,
		); err == nil {

			if err := r.Delete(
				ctx,
				&role,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetRole,
			func() error {
				targetRole.Rules = sec.Spec.RBAC.Rules

				if err := ctrl.SetControllerReference(
					&sec,
					targetRole,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedRoleObj, err := patch.ApplyEscapeHatches(
					targetRole,
					sec.Spec.EscapeHatches,
					"Role",
				)

				if err != nil {
					return fmt.Errorf("failed to apply escape hatch to Role: %w", err)
				}

				*targetRole = *(patchedRoleObj.(*rbacv1.Role))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *RoleReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Security{}).
		Owns(&rbacv1.Role{}).
		Named("security-role").
		Complete(r)
}
