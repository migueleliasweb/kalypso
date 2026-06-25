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

// ServiceSpec configures the Kubernetes Service for the target workload.
type ServiceSpec struct {
	// Ports is a list of routing ports.
	// +optional
	Ports []corev1.ServicePort `json:"ports,omitempty"`

	// Type of the Service (e.g. ClusterIP, NodePort, LoadBalancer).
	// +optional
	// +kubebuilder:default="ClusterIP"
	Type corev1.ServiceType `json:"type,omitempty"`
}

// GatewayRoute defines ingress routing parameters.
type GatewayRoute struct {
	// Hostnames for the route.
	// +optional
	Hostnames []string `json:"hostnames,omitempty"`

	// Paths is a list of path prefixes or matches.
	// +optional
	Paths []string `json:"paths,omitempty"`
}

// TLSSpec defines ingress TLS configuration.
type TLSSpec struct {
	// SecretName containing the TLS certificate.
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// IngressSpec configures HTTP Ingress.
type IngressSpec struct {
	// PrivateRoutes configures Gateway API HTTPRoutes / internal routes.
	// +optional
	PrivateRoutes []GatewayRoute `json:"privateRoutes,omitempty"`

	// PublicRoutes configures Gateway API HTTPRoutes / public routes.
	// +optional
	PublicRoutes []GatewayRoute `json:"publicRoutes,omitempty"`

	// TLS configuration.
	// +optional
	TLS TLSSpec `json:"tls,omitempty"`
}

// NetworkingSpec defines the desired state of Networking
type NetworkingSpec struct {
	// TargetRef references the target resource this capability applies to.
	// +optional
	TargetRef TargetRef `json:"targetRef,omitempty"`

	// Service configures the Service for the target workload.
	// +optional
	Service ServiceSpec `json:"service,omitempty"`

	// Ingress configures external access routes.
	// +optional
	Ingress IngressSpec `json:"ingress,omitempty"`

	// EscapeHatches allows applying raw patches to the target resource or managed Services/Ingresses.
	// +optional
	EscapeHatches []EscapeHatch `json:"escapeHatches,omitempty"`
}

// NetworkingStatus defines the observed state of Networking.
type NetworkingStatus struct {
	// conditions represent the current state of the Networking resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Networking is the Schema for the networkings API
type Networking struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkingSpec   `json:"spec,omitempty"`
	Status NetworkingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkingList contains a list of Networking
type NetworkingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Networking `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Networking{}, &NetworkingList{})
}
