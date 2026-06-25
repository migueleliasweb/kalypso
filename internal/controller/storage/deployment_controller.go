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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// DeploymentReconciler patches target Deployments with volume mounts for a Storage object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(
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

	if storage.Spec.TargetRef.Kind == "Deployment" {

		var deploy appsv1.Deployment

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: storage.Namespace, Name: storage.Spec.TargetRef.Resource},
			&deploy,
		); err != nil {
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

		patchedDeployObj, err := patch.ApplyEscapeHatches(
			&deploy,
			storage.Spec.EscapeHatches,
			"Deployment",
		)

		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to apply escape hatch to Deployment: %w", err)
		}

		deploy = *(patchedDeployObj.(*appsv1.Deployment))

		if err := r.Patch(
			ctx,
			&deploy,
			client.MergeFrom(originalDeploy),
		); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch deployment with volumes: %w", err)
		}

	}

	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) findStoragesForDeployment(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {

	deploy, ok := obj.(*appsv1.Deployment)

	if !ok {
		return nil
	}

	var list calypsov1alpha1.StorageList

	if err := r.List(
		ctx,
		&list,
		client.InNamespace(deploy.Namespace),
	); err != nil {
		return nil
	}

	var requests []reconcile.Request

	for _, storage := range list.Items {
		if storage.Spec.TargetRef.Resource != "" &&
			storage.Spec.TargetRef.Kind == "Deployment" &&
			storage.Spec.TargetRef.Resource == deploy.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: storage.Namespace,
					Name:      storage.Name,
				},
			})
		}
	}

	return requests
}

func (r *DeploymentReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Storage{}).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.findStoragesForDeployment),
		).
		Named("storage-deployment").
		Complete(r)
}
