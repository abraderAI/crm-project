// Package metadata provides JSON metadata deep-merge and parsing helpers.
package metadata

import (
	"encoding/json"
	"fmt"
)

// DeepMerge merges the patch JSON into the base JSON string.
// Objects are merged recursively; scalar values and arrays from patch overwrite base.
// Returns the merged JSON string.
func DeepMerge(base, patch string) (string, error) {
	if patch == "" || patch == "{}" {
		return base, nil
	}
	if base == "" || base == "{}" {
		return patch, nil
	}

	var baseMap, patchMap map[string]any
	if err := json.Unmarshal([]byte(base), &baseMap); err != nil {
		return "", fmt.Errorf("parsing base metadata: %w", err)
	}
	if err := json.Unmarshal([]byte(patch), &patchMap); err != nil {
		return "", fmt.Errorf("parsing patch metadata: %w", err)
	}

	merged := merge(baseMap, patchMap)
	result, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("serializing merged metadata: %w", err)
	}
	return string(result), nil
}

// merge recursively merges src into dst.
func merge(dst, src map[string]any) map[string]any {
	for key, srcVal := range src {
		if srcVal == nil {
			delete(dst, key)
			continue
		}
		dstVal, exists := dst[key]
		if !exists {
			dst[key] = srcVal
			continue
		}
		srcMap, srcOk := srcVal.(map[string]any)
		dstMap, dstOk := dstVal.(map[string]any)
		if srcOk && dstOk {
			dst[key] = merge(dstMap, srcMap)
		} else {
			dst[key] = srcVal
		}
	}
	return dst
}

// Validate checks that a string is valid JSON and is an object (not array/scalar).
func Validate(s string) error {
	if s == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return fmt.Errorf("metadata must be a valid JSON object: %w", err)
	}
	return nil
}
