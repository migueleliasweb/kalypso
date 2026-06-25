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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VolumeSpec represents a PersistentVolumeClaim to be created and mounted.
type VolumeSpec struct {
	// Name of the volume (and the generated PVC).
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Size of the persistent volume claim (e.g. "10Gi").
	// +kubebuilder:validation:Required
	Size string `json:"size"`

	// MountPath is the path inside the container where the volume should be mounted.
	// +kubebuilder:validation:Required
	MountPath string `json:"mountPath"`

	// AccessModes for the PVC (defaults to ReadWriteOnce).
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
}

// StorageSpec defines the desired state of Storage
type StorageSpec struct {
	// TargetRef references the target resource this capability applies to.
	// +optional
	TargetRef *ResourceRef `json:"targetRef,omitempty"`

	// StorageClassName to use for volume creation.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Volumes list of persistent volume claims to manage and mount.
	// +optional
	Volumes []VolumeSpec `json:"volumes,omitempty"`

	// EscapeHatches allows applying raw patches to the target resource or managed PVCs.
	// +optional
	EscapeHatches []EscapeHatch `json:"escapeHatches,omitempty"`
}

// StorageStatus defines the observed state of Storage.
type StorageStatus struct {
	// conditions represent the current state of the Storage resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Storage is the Schema for the storages API
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec,omitempty"`
	Status StorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
