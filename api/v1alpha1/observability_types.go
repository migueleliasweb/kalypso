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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceMonitorSpec enables Prometheus monitoring.
type ServiceMonitorSpec struct {
	// Enabled defines if a ServiceMonitor should be created.
	Enabled bool `json:"enabled"`

	// Interval at which metrics should be scraped.
	// +optional
	Interval string `json:"interval,omitempty"`

	// Path is the metrics endpoint path (defaults to /metrics).
	// +optional
	// +kubebuilder:default="/metrics"
	Path string `json:"path,omitempty"`
}

// PodMonitorSpec enables Prometheus monitoring at the pod level.
type PodMonitorSpec struct {
	// Enabled defines if a PodMonitor should be created.
	Enabled bool `json:"enabled"`

	// Interval at which metrics should be scraped.
	// +optional
	Interval string `json:"interval,omitempty"`

	// Path is the metrics endpoint path (defaults to /metrics).
	// +optional
	// +kubebuilder:default="/metrics"
	Path string `json:"path,omitempty"`
}

// ObservabilitySpec defines the desired state of Observability
type ObservabilitySpec struct {
	// TargetRef references the target resource this capability applies to.
	// +optional
	TargetRef *ResourceRef `json:"targetRef,omitempty"`

	// ServiceMonitor enables Prometheus monitoring via ServiceMonitor resources.
	// +optional
	ServiceMonitor *ServiceMonitorSpec `json:"serviceMonitor,omitempty"`

	// PodMonitor enables Prometheus monitoring via PodMonitor resources.
	// +optional
	PodMonitor *PodMonitorSpec `json:"podMonitor,omitempty"`

	// EscapeHatches allows applying raw patches to observability resources.
	// +optional
	EscapeHatches []EscapeHatch `json:"escapeHatches,omitempty"`
}

// ObservabilityStatus defines the observed state of Observability.
type ObservabilityStatus struct {
	// conditions represent the current state of the Observability resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Observability is the Schema for the observabilities API
type Observability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObservabilitySpec   `json:"spec,omitempty"`
	Status ObservabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObservabilityList contains a list of Observability
type ObservabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Observability `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Observability{}, &ObservabilityList{})
}
