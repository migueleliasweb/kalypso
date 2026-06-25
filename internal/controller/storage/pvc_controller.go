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

package storage

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// PVCReconciler reconciles PVC resources for a Storage object
type PVCReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PVCReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var storage calypsov1alpha1.Storage

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&storage,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if storage.Spec.TargetRef.Resource == "" {

		log.Info("Storage targetRef is nil, skipping reconciliation", "name", storage.Name)

		return ctrl.Result{}, nil
	}

	for _, vol := range storage.Spec.Volumes {

		pvcName := fmt.Sprintf("%s-%s", storage.Name, vol.Name)

		targetPVC := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: storage.Namespace,
			},
		}

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetPVC,
			func() error {
				accessModes := vol.AccessModes

				if len(accessModes) == 0 {
					accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
				}

				quantity, err := resource.ParseQuantity(vol.Size)

				if err != nil {
					return fmt.Errorf("failed to parse volume size %q: %w", vol.Size, err)
				}

				var storageClassPtr *string

				if storage.Spec.StorageClassName != "" {
					scName := storage.Spec.StorageClassName
					storageClassPtr = &scName
				}

				targetPVC.Spec.AccessModes = accessModes
				targetPVC.Spec.Resources = corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: quantity,
					},
				}
				targetPVC.Spec.StorageClassName = storageClassPtr

				if err := ctrl.SetControllerReference(
					&storage,
					targetPVC,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedPVCObj, err := patch.ApplyEscapeHatches(
					targetPVC,
					storage.Spec.EscapeHatches,
					"PersistentVolumeClaim",
				)

				if err != nil {
					return fmt.Errorf("failed to apply escape hatch to PVC: %w", err)
				}

				*targetPVC = *(patchedPVCObj.(*corev1.PersistentVolumeClaim))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *PVCReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Storage{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("storage-pvc").
		Complete(r)
}
