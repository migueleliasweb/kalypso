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

	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// PDBReconciler reconciles PDB resources for a Compute object
type PDBReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PDBReconciler) Reconcile(
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

	pdbName := fmt.Sprintf("%s-pdb", compute.Name)

	targetPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pdbName,
			Namespace: compute.Namespace,
		},
	}

	hasPDB := compute.Spec.PodDisruptionBudget.MinAvailable.IntVal > 0 ||
		compute.Spec.PodDisruptionBudget.MinAvailable.StrVal != "" ||
		compute.Spec.PodDisruptionBudget.MaxUnavailable.IntVal > 0 ||
		compute.Spec.PodDisruptionBudget.MaxUnavailable.StrVal != ""

	if !hasPDB {

		var pdb policyv1.PodDisruptionBudget

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: compute.Namespace, Name: pdbName},
			&pdb,
		); err == nil {

			if err := r.Delete(
				ctx,
				&pdb,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		var deploy appsv1.Deployment

		deployFound := false

		if compute.Spec.TargetRef.Kind == "Deployment" {

			if err := r.Get(
				ctx,
				client.ObjectKey{Namespace: compute.Namespace, Name: compute.Spec.TargetRef.Resource},
				&deploy,
			); err == nil {
				deployFound = true
			}

		}

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetPDB,
			func() error {
				var selector *metav1.LabelSelector

				if deployFound && deploy.Spec.Selector != nil {
					selector = deploy.Spec.Selector
				} else {
					selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": compute.Spec.TargetRef.Resource,
						},
					}
				}

				targetPDB.Spec.Selector = selector

				if compute.Spec.PodDisruptionBudget.MinAvailable.IntVal > 0 || compute.Spec.PodDisruptionBudget.MinAvailable.StrVal != "" {
					minAvailableVal := compute.Spec.PodDisruptionBudget.MinAvailable

					targetPDB.Spec.MinAvailable = &minAvailableVal
					targetPDB.Spec.MaxUnavailable = nil

				} else if compute.Spec.PodDisruptionBudget.MaxUnavailable.IntVal > 0 || compute.Spec.PodDisruptionBudget.MaxUnavailable.StrVal != "" {
					maxUnavailableVal := compute.Spec.PodDisruptionBudget.MaxUnavailable

					targetPDB.Spec.MaxUnavailable = &maxUnavailableVal
					targetPDB.Spec.MinAvailable = nil
				}

				if err := ctrl.SetControllerReference(
					&compute,
					targetPDB,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedPDBObj, err := patch.ApplyEscapeHatches(
					targetPDB,
					compute.Spec.EscapeHatches,
					"PodDisruptionBudget",
				)

				if err != nil {
					return err
				}

				*targetPDB = *(patchedPDBObj.(*policyv1.PodDisruptionBudget))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *PDBReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Compute{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Named("compute-pdb").
		Complete(r)
}
