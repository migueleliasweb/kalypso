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

// TargetRef represents a generic reference to a target Kubernetes resource.
type TargetRef struct {
	// Name is the name of the resource (e.g. "my-app").
	// +kubebuilder:validation:Required
	Name string `json:"resource"`

	// Kind is the kind of the resource (e.g. "Deployment").
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// ApiVersion is the API version of the resource (e.g. "apps/v1").
	// +kubebuilder:validation:Required
	ApiVersion string `json:"apiVersion"`

	// Namespace is the namespace of the resource.
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
}

// EscapeHatch allows users to patch fields of resources managed by the capability.
type EscapeHatch struct {
	// Kind is the kind of resource to patch (e.g. "Pod", "Service", "Deployment", "HorizontalPodAutoscaler", etc.)
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// PatchType is the type of patch to apply. Supported types: JSONPatch, JSONMergePatch.
	// +kubebuilder:validation:Enum=JSONPatch;JSONMergePatch
	// +kubebuilder:validation:Required
	PatchType string `json:"patchType"`

	// Patch is the patch string payload to apply (JSON patch or Merge patch content).
	// +kubebuilder:validation:Required
	Patch string `json:"patch"`
}
