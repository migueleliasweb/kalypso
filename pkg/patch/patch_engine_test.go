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

package patch

import (
	"testing"

	"github.com/migueleliasweb/kalypso/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyEscapeHatches(t *testing.T) {
	// 1. Arrange a mock Deployment
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	// 2. Define dynamic patches
	hatches := []v1alpha1.EscapeHatch{
		{
			Kind:      "Deployment",
			PatchType: "JSONMergePatch",
			Patch:     `{"metadata":{"labels":{"new-label":"value"}}}`,
		},
		{
			Kind:      "Deployment",
			PatchType: "JSONPatch",
			Patch:     `[{"op": "replace", "path": "/metadata/name", "value": "patched-name"}]`,
		},
		{
			Kind:      "Service", // should be ignored as kind does not match
			PatchType: "JSONMergePatch",
			Patch:     `{"spec":{"type":"LoadBalancer"}}`,
		},
	}

	// 3. Act
	result, err := ApplyEscapeHatches(deploy, hatches, "Deployment")
	if err != nil {
		t.Fatalf("unexpected error applying escape hatches: %v", err)
	}

	patchedDeploy, ok := result.(*appsv1.Deployment)
	if !ok {
		t.Fatalf("expected result to be a *appsv1.Deployment, got %T", result)
	}

	// 4. Assert
	if patchedDeploy.Name != "patched-name" {
		t.Errorf("expected name to be 'patched-name', got %q", patchedDeploy.Name)
	}

	if patchedDeploy.Labels["new-label"] != "value" {
		t.Errorf("expected label 'new-label' to be 'value', got %q", patchedDeploy.Labels["new-label"])
	}

	if patchedDeploy.Labels["app"] != "test" {
		t.Errorf("expected original label 'app' to remain 'test', got %q", patchedDeploy.Labels["app"])
	}
}
