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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// StorageReconciler reconciles a Storage object
type StorageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=storages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=storages/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop.
func (r *StorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the Storage resource
	var storage calypsov1alpha1.Storage
	if err := r.Get(ctx, req.NamespacedName, &storage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if storage.Spec.TargetRef == nil {
		log.Info("Storage targetRef is nil, skipping reconciliation", "name", storage.Name)
		return ctrl.Result{}, nil
	}

	// 2. Reconcile PVCs
	for _, vol := range storage.Spec.Volumes {
		if err := r.reconcilePVC(ctx, &storage, vol); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Patch Target Deployment
	if storage.Spec.TargetRef.Kind == "Deployment" {
		var deploy appsv1.Deployment
		err := r.Get(ctx, client.ObjectKey{Namespace: storage.Namespace, Name: storage.Spec.TargetRef.Resource}, &deploy)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Target Deployment not found", "name", storage.Spec.TargetRef.Resource)
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		originalDeploy := deploy.DeepCopy()

		for _, vol := range storage.Spec.Volumes {
			pvcName := fmt.Sprintf("%s-%s", storage.Name, vol.Name)

			volumeExists := false
			for i, v := range deploy.Spec.Template.Spec.Volumes {
				if v.Name == vol.Name {
					deploy.Spec.Template.Spec.Volumes[i].VolumeSource = corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					}
					volumeExists = true
					break
				}
			}
			if !volumeExists {
				deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, corev1.Volume{
					Name: vol.Name,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				})
			}

			if len(deploy.Spec.Template.Spec.Containers) > 0 {
				mountExists := false
				for i, m := range deploy.Spec.Template.Spec.Containers[0].VolumeMounts {
					if m.Name == vol.Name {
						deploy.Spec.Template.Spec.Containers[0].VolumeMounts[i].MountPath = vol.MountPath
						mountExists = true
						break
					}
				}
				if !mountExists {
					deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(
						deploy.Spec.Template.Spec.Containers[0].VolumeMounts,
						corev1.VolumeMount{
							Name:      vol.Name,
							MountPath: vol.MountPath,
						},
					)
				}
			}
		}

		patchedDeployObj, err := patch.ApplyEscapeHatches(&deploy, storage.Spec.EscapeHatches, "Deployment")
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to apply escape hatch to Deployment: %w", err)
		}
		deploy = *(patchedDeployObj.(*appsv1.Deployment))

		if err := r.Patch(ctx, &deploy, client.MergeFrom(originalDeploy)); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch deployment with volumes: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *StorageReconciler) reconcilePVC(ctx context.Context, storage *calypsov1alpha1.Storage, vol calypsov1alpha1.VolumeSpec) error {
	pvcName := fmt.Sprintf("%s-%s", storage.Name, vol.Name)
	var pvc corev1.PersistentVolumeClaim
	pvcErr := r.Get(ctx, client.ObjectKey{Namespace: storage.Namespace, Name: pvcName}, &pvc)

	accessModes := vol.AccessModes
	if len(accessModes) == 0 {
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	quantity, err := resource.ParseQuantity(vol.Size)
	if err != nil {
		return fmt.Errorf("failed to parse volume size %q: %w", vol.Size, err)
	}

	targetPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: storage.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
			StorageClassName: storage.Spec.StorageClassName,
		},
	}

	if err := ctrl.SetControllerReference(storage, targetPVC, r.Scheme); err != nil {
		return err
	}

	patchedPVCObj, err := patch.ApplyEscapeHatches(targetPVC, storage.Spec.EscapeHatches, "PersistentVolumeClaim")
	if err != nil {
		return fmt.Errorf("failed to apply escape hatch to PVC: %w", err)
	}
	targetPVC = patchedPVCObj.(*corev1.PersistentVolumeClaim)

	if pvcErr != nil {
		if apierrors.IsNotFound(pvcErr) {
			return r.Create(ctx, targetPVC)
		}
		return pvcErr
	}

	targetPVC.ResourceVersion = pvc.ResourceVersion
	return r.Update(ctx, targetPVC)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Storage{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("storage").
		Complete(r)
}
