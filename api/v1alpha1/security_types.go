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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RBACSpec configures Role and RoleBinding for the workload.
type RBACSpec struct {
	// CreateRole determines if a Role should be created.
	CreateRole bool `json:"createRole"`

	// Rules is the list of PolicyRules to assign to the Role.
	// +optional
	Rules []rbacv1.PolicyRule `json:"rules,omitempty"`
}

// SecuritySpec defines the desired state of Security
type SecuritySpec struct {
	// TargetRef references the target resource this capability applies to.
	// +optional
	TargetRef TargetRef `json:"targetRef,omitempty"`

	// RBAC configures Role and RoleBinding for the workload.
	// +optional
	RBAC RBACSpec `json:"rbac,omitempty"`

	// EscapeHatches allows applying raw patches to security resources.
	// +optional
	EscapeHatches []EscapeHatch `json:"escapeHatches,omitempty"`
}

// SecurityStatus defines the observed state of Security.
type SecurityStatus struct {
	// conditions represent the current state of the Security resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Security is the Schema for the securities API
type Security struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecuritySpec   `json:"spec,omitempty"`
	Status SecurityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityList contains a list of Security
type SecurityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Security `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Security{}, &SecurityList{})
}
