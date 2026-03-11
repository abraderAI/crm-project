package metadata

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		patch string
		want  map[string]any
	}{
		{"empty patch", `{"a":"1"}`, "{}", map[string]any{"a": "1"}},
		{"empty base", "{}", `{"a":"1"}`, map[string]any{"a": "1"}},
		{"simple merge", `{"a":"1"}`, `{"b":"2"}`, map[string]any{"a": "1", "b": "2"}},
		{"overwrite scalar", `{"a":"1"}`, `{"a":"2"}`, map[string]any{"a": "2"}},
		{"deep merge objects", `{"a":{"x":"1"}}`, `{"a":{"y":"2"}}`, map[string]any{"a": map[string]any{"x": "1", "y": "2"}}},
		{"null removes key", `{"a":"1","b":"2"}`, `{"a":null}`, map[string]any{"b": "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeepMerge(tt.base, tt.patch)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(result), &got))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeepMerge_Errors(t *testing.T) {
	_, err := DeepMerge("not json", `{"a":"1"}`)
	assert.Error(t, err)

	_, err = DeepMerge(`{"a":"1"}`, "not json")
	assert.Error(t, err)
}

func TestValidate(t *testing.T) {
	assert.NoError(t, Validate(""))
	assert.NoError(t, Validate(`{"key":"value"}`))
	assert.Error(t, Validate("not json"))
	assert.Error(t, Validate(`["array"]`))
}
