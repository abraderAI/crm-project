// Package metadata provides JSON metadata merge and filter utilities.
package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// DeepMerge merges the patch JSON string into the base JSON string.
// Nested objects are merged recursively; other values are overwritten.
// Returns the merged JSON string.
func DeepMerge(base, patch string) (string, error) {
	if base == "" || base == "{}" {
		if patch == "" {
			return "{}", nil
		}
		return patch, nil
	}
	if patch == "" || patch == "{}" {
		return base, nil
	}

	var baseMap, patchMap map[string]interface{}
	if err := json.Unmarshal([]byte(base), &baseMap); err != nil {
		return "", fmt.Errorf("parsing base metadata: %w", err)
	}
	if err := json.Unmarshal([]byte(patch), &patchMap); err != nil {
		return "", fmt.Errorf("parsing patch metadata: %w", err)
	}

	merged := deepMergeMaps(baseMap, patchMap)
	result, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("serializing merged metadata: %w", err)
	}
	return string(result), nil
}

// deepMergeMaps recursively merges src into dst.
func deepMergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(dst)+len(src))
	for k, v := range dst {
		result[k] = v
	}
	for k, v := range src {
		if v == nil {
			delete(result, k)
			continue
		}
		if srcMap, ok := v.(map[string]interface{}); ok {
			if dstMap, ok := result[k].(map[string]interface{}); ok {
				result[k] = deepMergeMaps(dstMap, srcMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}

// Validate checks if the given string is valid JSON.
func Validate(s string) error {
	if s == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return fmt.Errorf("invalid JSON metadata: %w", err)
	}
	return nil
}

// Filter represents a metadata filter parsed from query parameters.
type Filter struct {
	Key      string
	Operator string // "eq", "gt", "lt", "gte", "lte"
	Value    string
}

// metadataKeyPattern matches metadata[key] or metadata[key][op].
var metadataKeyPattern = regexp.MustCompile(`^metadata\[([^\]]+)\](?:\[([^\]]+)\])?$`)

// ParseFilters extracts metadata filters from query parameters.
// Supports: metadata[key]=value, metadata[key][gt]=value, etc.
func ParseFilters(r *http.Request) []Filter {
	var filters []Filter
	for key, values := range r.URL.Query() {
		matches := metadataKeyPattern.FindStringSubmatch(key)
		if matches == nil || len(values) == 0 {
			continue
		}
		metaKey := matches[1]
		op := "eq"
		if matches[2] != "" {
			op = matches[2]
		}
		if !isValidOperator(op) {
			continue
		}
		filters = append(filters, Filter{
			Key:      metaKey,
			Operator: op,
			Value:    values[0],
		})
	}
	return filters
}

// ToSQLConditions converts filters to SQL WHERE conditions using json_extract.
// Returns conditions and args for GORM Where calls.
func ToSQLConditions(filters []Filter) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	for _, f := range filters {
		jsonPath := fmt.Sprintf("$.%s", sanitizeJSONPath(f.Key))
		expr := fmt.Sprintf("json_extract(metadata, '%s')", jsonPath)
		var cond string
		switch f.Operator {
		case "eq":
			cond = expr + " = ?"
		case "gt":
			cond = expr + " > ?"
		case "lt":
			cond = expr + " < ?"
		case "gte":
			cond = expr + " >= ?"
		case "lte":
			cond = expr + " <= ?"
		default:
			continue
		}
		conditions = append(conditions, cond)
		args = append(args, f.Value)
	}
	return conditions, args
}

func isValidOperator(op string) bool {
	switch op {
	case "eq", "gt", "lt", "gte", "lte":
		return true
	}
	return false
}

// sanitizeJSONPath removes unsafe characters from a JSON path key.
func sanitizeJSONPath(key string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '.' {
			return r
		}
		return -1
	}, key)
	return safe
}
