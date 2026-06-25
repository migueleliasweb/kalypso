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

// SecurityReconciler reconciles a Security object
type SecurityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SecurityReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	// Fetch the Security resource
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

	// 2. Reconcile RBAC
	roleName := fmt.Sprintf("%s-role", sec.Name)
	rbName := fmt.Sprintf("%s-rb", sec.Name)

	targetRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: sec.Namespace,
		},
	}

	targetRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbName,
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

		var rb rbacv1.RoleBinding

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: sec.Namespace, Name: rbName},
			&rb,
		); err == nil {

			if err := r.Delete(
				ctx,
				&rb,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		// Reconcile Role
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

		// Reconcile RoleBinding
		_, err = controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetRB,
			func() error {
				targetRB.RoleRef = rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     roleName,
				}
				targetRB.Subjects = []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      sec.Spec.TargetRef.Resource,
						Namespace: sec.Namespace,
					},
				}

				if err := ctrl.SetControllerReference(
					&sec,
					targetRB,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedRBObj, err := patch.ApplyEscapeHatches(
					targetRB,
					sec.Spec.EscapeHatches,
					"RoleBinding",
				)

				if err != nil {
					return fmt.Errorf("failed to apply escape hatch to RoleBinding: %w", err)
				}

				*targetRB = *(patchedRBObj.(*rbacv1.RoleBinding))

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
func (r *SecurityReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Security{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Named("security").
		Complete(r)
}
