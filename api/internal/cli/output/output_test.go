package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderListTable(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)

	entities := []map[string]any{
		{"id": "1", "name": "Org One", "status": "active"},
		{"id": "2", "name": "Org Two", "status": "inactive"},
	}

	err := f.RenderList(entities)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Org One")
	assert.Contains(t, out, "Org Two")
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "NAME")
}

func TestRenderListJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatJSON, false)

	entities := []map[string]any{
		{"id": "1", "name": "Test"},
	}

	err := f.RenderList(entities)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "Test", result[0]["name"])
}

func TestRenderListEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)

	err := f.RenderList(nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No results found")
}

func TestRenderDetailTable(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)

	entity := map[string]any{
		"id":         "org-1",
		"name":       "My Organization",
		"created_at": "2024-01-01",
	}

	err := f.RenderDetail(entity)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "My Organization")
	assert.Contains(t, out, "org-1")
}

func TestRenderDetailJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatJSON, false)

	entity := map[string]any{"id": "1", "name": "Test"}
	err := f.RenderDetail(entity)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "Test", result["name"])
}

func TestRenderDetailEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)
	err := f.RenderDetail(nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No data")
}

func TestRenderText(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)
	f.RenderText("Hello World")
	assert.Contains(t, buf.String(), "Hello World")
}

func TestRenderError(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)
	f.RenderError("something failed")
	assert.Contains(t, buf.String(), "Error: something failed")
}

func TestRenderSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)
	f.RenderSuccess("operation completed")
	assert.Contains(t, buf.String(), "operation completed")
}

func TestRenderTextWithColor(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, true)
	f.RenderText("colored text")
	assert.Contains(t, buf.String(), "colored text")
}

func TestRenderErrorWithColor(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, true)
	f.RenderError("colored error")
	assert.Contains(t, buf.String(), "colored error")
}

func TestRenderSuccessWithColor(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, true)
	f.RenderSuccess("colored success")
	assert.Contains(t, buf.String(), "colored success")
}

func TestNewRespectsNOCOLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	buf := &bytes.Buffer{}
	f := New(buf, FormatTable)
	assert.False(t, f.color)
}

func TestNewDefaultColor(t *testing.T) {
	buf := &bytes.Buffer{}
	f := New(buf, FormatTable)
	assert.True(t, f.color)
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"long string", string(make([]byte, 100)), string(make([]byte, 77)) + "..."},
		{"int float", float64(42), "42"},
		{"decimal float", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"map", map[string]any{"k": "v"}, `{"k":"v"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCollectKeys(t *testing.T) {
	entities := []map[string]any{
		{"id": "1", "name": "A", "custom": "x"},
		{"id": "2", "title": "B", "other": "y"},
	}
	keys := collectKeys(entities)
	// "id" and "name" should appear before custom fields.
	assert.Equal(t, "id", keys[0])
	assert.Contains(t, keys, "name")
	assert.Contains(t, keys, "title")
	assert.Contains(t, keys, "custom")
	assert.Contains(t, keys, "other")
}

func TestSortedKeys(t *testing.T) {
	m := map[string]any{"z": 1, "a": 2, "m": 3}
	keys := sortedKeys(m)
	assert.Equal(t, []string{"a", "m", "z"}, keys)
}

func TestRenderListWithMixedFields(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, false)

	entities := []map[string]any{
		{"id": "1", "name": "A"},
		{"id": "2", "extra": "B"},
	}

	err := f.RenderList(entities)
	require.NoError(t, err)
	// Should not panic and should render something.
	assert.True(t, buf.Len() > 0)
}

func TestRenderDetailWithColor(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewWithColor(buf, FormatTable, true)

	entity := map[string]any{"id": "1", "name": "Test"}
	err := f.RenderDetail(entity)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Test")
}

func TestFormatValueMap(t *testing.T) {
	m := map[string]any{"key": "value"}
	result := formatValue(m)
	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

func TestFormatValueLongMap(t *testing.T) {
	m := map[string]any{}
	for i := 0; i < 50; i++ {
		m[string(rune('a'+i%26))+string(rune('0'+i/26))] = "x"
	}
	result := formatValue(m)
	assert.True(t, len(result) <= 80)
}
