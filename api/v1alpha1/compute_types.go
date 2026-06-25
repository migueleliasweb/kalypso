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
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AutoscalingSpec defines the HPA scaling parameters.
type AutoscalingSpec struct {
	// MinReplicas is the lower limit for the number of replicas.
	// +optional
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the upper limit for the number of replicas.
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// TargetCPUUtilizationPercentage is the target average CPU utilization percentage.
	// +optional
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage,omitempty"`

	// TargetMemoryUtilizationPercentage is the target average memory utilization percentage.
	// +optional
	TargetMemoryUtilizationPercentage int32 `json:"targetMemoryUtilizationPercentage,omitempty"`
}

// SchedulingSpec defines the scheduling criteria for pods.
type SchedulingSpec struct {
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity defines scheduling constraints (node affinity, pod affinity, pod anti-affinity).
	// +optional
	Affinity corev1.Affinity `json:"affinity,omitempty"`

	// TopologySpreadConstraints describes how a group of pods ought to spread across topology domains.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// PDBSpec defines PodDisruptionBudget parameters.
type PDBSpec struct {
	// MinAvailable is the minimum number of pods that must be available.
	// +optional
	MinAvailable intstr.IntOrString `json:"minAvailable,omitempty"`

	// MaxUnavailable is the maximum number of pods that can be unavailable.
	// +optional
	MaxUnavailable intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// ComputeSpec defines the desired state of Compute
type ComputeSpec struct {
	// TargetRef references the target resource this capability applies to.
	// +optional
	TargetRef TargetRef `json:"targetRef,omitempty"`

	// Autoscaling defines HPA autoscaling configuration.
	// +optional
	Autoscaling AutoscalingSpec `json:"autoscaling,omitempty"`

	// Scheduling defines node affinity, node selector, and topology spread rules.
	// +optional
	Scheduling SchedulingSpec `json:"scheduling,omitempty"`

	// PodDisruptionBudget defines the PDB configuration.
	// +optional
	PodDisruptionBudget PDBSpec `json:"podDisruptionBudget,omitempty"`

	// EscapeHatches allows applying raw patches to the target resource or managed HPAs/PDBs.
	// +optional
	EscapeHatches []EscapeHatch `json:"escapeHatches,omitempty"`
}

// ComputeStatus defines the observed state of Compute.
type ComputeStatus struct {
	// conditions represent the current state of the Compute resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Compute is the Schema for the computes API
type Compute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComputeSpec   `json:"spec,omitempty"`
	Status ComputeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComputeList contains a list of Compute
type ComputeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Compute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Compute{}, &ComputeList{})
}
