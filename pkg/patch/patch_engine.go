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
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/migueleliasweb/kalypso/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ApplyEscapeHatches applies a list of EscapeHatch patches to a Kubernetes runtime.Object.
// It filters the hatches by the object's resource Kind and applies the matching patches sequentially.
func ApplyEscapeHatches(obj runtime.Object, hatches []v1alpha1.EscapeHatch, targetKind string) (runtime.Object, error) {
	patchedObj := obj.DeepCopyObject()

	for _, hatch := range hatches {
		if hatch.Kind != targetKind {
			continue
		}

		// Convert object to JSON
		originalJSON, err := json.Marshal(patchedObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object of kind %s to JSON: %w", targetKind, err)
		}

		var patchedJSON []byte
		switch hatch.PatchType {
		case "JSONPatch":
			patch, err := jsonpatch.DecodePatch([]byte(hatch.Patch))
			if err != nil {
				return nil, fmt.Errorf("failed to decode JSONPatch for kind %s: %w", targetKind, err)
			}
			patchedJSON, err = patch.Apply(originalJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to apply JSONPatch for kind %s: %w", targetKind, err)
			}

		case "JSONMergePatch":
			patchedJSON, err = jsonpatch.MergePatch(originalJSON, []byte(hatch.Patch))
			if err != nil {
				return nil, fmt.Errorf("failed to apply JSONMergePatch for kind %s: %w", targetKind, err)
			}

		default:
			return nil, fmt.Errorf("unsupported patch type %q for kind %s", hatch.PatchType, targetKind)
		}

		// Unmarshal back to object
		if err := json.Unmarshal(patchedJSON, patchedObj); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patched JSON back to object of kind %s: %w", targetKind, err)
		}
	}

	return patchedObj, nil
}
