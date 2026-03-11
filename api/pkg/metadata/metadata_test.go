package metadata

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DeepMerge ---

func TestDeepMerge_EmptyBase(t *testing.T) {
	result, err := DeepMerge("", `{"key":"val"}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"key":"val"}`, result)
}

func TestDeepMerge_EmptyPatch(t *testing.T) {
	result, err := DeepMerge(`{"key":"val"}`, "")
	require.NoError(t, err)
	assert.JSONEq(t, `{"key":"val"}`, result)
}

func TestDeepMerge_BothEmpty(t *testing.T) {
	result, err := DeepMerge("", "")
	require.NoError(t, err)
	assert.Equal(t, "{}", result)
}

func TestDeepMerge_BothEmptyObjects(t *testing.T) {
	result, err := DeepMerge("{}", "{}")
	require.NoError(t, err)
	assert.JSONEq(t, "{}", result)
}

func TestDeepMerge_Simple(t *testing.T) {
	result, err := DeepMerge(`{"a":"1"}`, `{"b":"2"}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"1","b":"2"}`, result)
}

func TestDeepMerge_Overwrite(t *testing.T) {
	result, err := DeepMerge(`{"a":"old"}`, `{"a":"new"}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"new"}`, result)
}

func TestDeepMerge_NestedMerge(t *testing.T) {
	base := `{"nested":{"x":"1","y":"2"}}`
	patch := `{"nested":{"y":"3","z":"4"}}`
	result, err := DeepMerge(base, patch)
	require.NoError(t, err)
	assert.JSONEq(t, `{"nested":{"x":"1","y":"3","z":"4"}}`, result)
}

func TestDeepMerge_NullRemoves(t *testing.T) {
	result, err := DeepMerge(`{"a":"1","b":"2"}`, `{"a":null}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"b":"2"}`, result)
}

func TestDeepMerge_InvalidBase(t *testing.T) {
	_, err := DeepMerge("invalid", `{"a":"1"}`)
	assert.Error(t, err)
}

func TestDeepMerge_InvalidPatch(t *testing.T) {
	_, err := DeepMerge(`{"a":"1"}`, "invalid")
	assert.Error(t, err)
}

func TestDeepMerge_DeeplyNested(t *testing.T) {
	base := `{"l1":{"l2":{"l3":"val"}}}`
	patch := `{"l1":{"l2":{"l3":"new","l3b":"extra"}}}`
	result, err := DeepMerge(base, patch)
	require.NoError(t, err)
	assert.JSONEq(t, `{"l1":{"l2":{"l3":"new","l3b":"extra"}}}`, result)
}

func TestDeepMerge_OverwriteObjectWithScalar(t *testing.T) {
	base := `{"a":{"nested":"val"}}`
	patch := `{"a":"scalar"}`
	result, err := DeepMerge(base, patch)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"scalar"}`, result)
}

// --- Validate ---

func TestValidate_Valid(t *testing.T) {
	assert.NoError(t, Validate(`{"key":"value"}`))
}

func TestValidate_Empty(t *testing.T) {
	assert.NoError(t, Validate(""))
}

func TestValidate_Invalid(t *testing.T) {
	assert.Error(t, Validate("not json"))
}

func TestValidate_EmptyObject(t *testing.T) {
	assert.NoError(t, Validate("{}"))
}

func TestValidate_ComplexJSON(t *testing.T) {
	assert.NoError(t, Validate(`{"nested":{"array":[1,2,3],"bool":true}}`))
}

// --- ParseFilters ---

func TestParseFilters_NoMetadata(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=10"}}
	filters := ParseFilters(r)
	assert.Empty(t, filters)
}

func TestParseFilters_SimpleEq(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "metadata[status]=open"}}
	filters := ParseFilters(r)
	require.Len(t, filters, 1)
	assert.Equal(t, "status", filters[0].Key)
	assert.Equal(t, "eq", filters[0].Operator)
	assert.Equal(t, "open", filters[0].Value)
}

func TestParseFilters_WithOperator(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "metadata[priority][gt]=3"}}
	filters := ParseFilters(r)
	require.Len(t, filters, 1)
	assert.Equal(t, "priority", filters[0].Key)
	assert.Equal(t, "gt", filters[0].Operator)
	assert.Equal(t, "3", filters[0].Value)
}

func TestParseFilters_MultipleFilters(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "metadata[status]=open&metadata[priority][gte]=2"}}
	filters := ParseFilters(r)
	assert.Len(t, filters, 2)
}

func TestParseFilters_InvalidOperator(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "metadata[status][invalid]=open"}}
	filters := ParseFilters(r)
	assert.Empty(t, filters)
}

func TestParseFilters_AllOperators(t *testing.T) {
	ops := []string{"eq", "gt", "lt", "gte", "lte"}
	for _, op := range ops {
		r := &http.Request{URL: &url.URL{RawQuery: "metadata[key][" + op + "]=val"}}
		filters := ParseFilters(r)
		require.Len(t, filters, 1, "operator %s should be valid", op)
		assert.Equal(t, op, filters[0].Operator)
	}
}

// --- ToSQLConditions ---

func TestToSQLConditions_Empty(t *testing.T) {
	conds, args := ToSQLConditions(nil)
	assert.Empty(t, conds)
	assert.Empty(t, args)
}

func TestToSQLConditions_EqOperator(t *testing.T) {
	filters := []Filter{{Key: "status", Operator: "eq", Value: "open"}}
	conds, args := ToSQLConditions(filters)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "json_extract(metadata, '$.status') = ?")
	assert.Equal(t, "open", args[0])
}

func TestToSQLConditions_AllOperators(t *testing.T) {
	tests := []struct {
		op       string
		expected string
	}{
		{"eq", "= ?"},
		{"gt", "> ?"},
		{"lt", "< ?"},
		{"gte", ">= ?"},
		{"lte", "<= ?"},
	}
	for _, tt := range tests {
		filters := []Filter{{Key: "k", Operator: tt.op, Value: "v"}}
		conds, args := ToSQLConditions(filters)
		require.Len(t, conds, 1)
		assert.Contains(t, conds[0], tt.expected)
		assert.Equal(t, "v", args[0])
	}
}

func TestToSQLConditions_InvalidOperator(t *testing.T) {
	filters := []Filter{{Key: "k", Operator: "invalid", Value: "v"}}
	conds, args := ToSQLConditions(filters)
	assert.Empty(t, conds)
	assert.Empty(t, args)
}

func TestToSQLConditions_MultipleFilters(t *testing.T) {
	filters := []Filter{
		{Key: "status", Operator: "eq", Value: "open"},
		{Key: "priority", Operator: "gt", Value: "3"},
	}
	conds, args := ToSQLConditions(filters)
	assert.Len(t, conds, 2)
	assert.Len(t, args, 2)
}

// --- sanitizeJSONPath ---

func TestSanitizeJSONPath_SafeKey(t *testing.T) {
	assert.Equal(t, "my_key.nested", sanitizeJSONPath("my_key.nested"))
}

func TestSanitizeJSONPath_UnsafeChars(t *testing.T) {
	result := sanitizeJSONPath("key'; DROP TABLE--")
	// Letters are kept, special chars removed.
	assert.Equal(t, "keyDROPTABLE", result)
}

// --- isValidOperator ---

func TestIsValidOperator(t *testing.T) {
	assert.True(t, isValidOperator("eq"))
	assert.True(t, isValidOperator("gt"))
	assert.True(t, isValidOperator("lt"))
	assert.True(t, isValidOperator("gte"))
	assert.True(t, isValidOperator("lte"))
	assert.False(t, isValidOperator("invalid"))
	assert.False(t, isValidOperator(""))
}

// --- Fuzz Tests ---

func FuzzDeepMerge(f *testing.F) {
	seeds := []struct{ base, patch string }{
		{`{}`, `{}`},
		{`{"a":"1"}`, `{"b":"2"}`},
		{`{"a":"1"}`, `{"a":null}`},
		{`{"n":{"x":"1"}}`, `{"n":{"y":"2"}}`},
	}
	for _, s := range seeds {
		f.Add(s.base, s.patch)
	}
	f.Fuzz(func(t *testing.T, base, patch string) {
		// Just ensure no panics.
		_, _ = DeepMerge(base, patch)
	})
}

func FuzzValidate(f *testing.F) {
	f.Add("")
	f.Add("{}")
	f.Add(`{"key":"value"}`)
	f.Add("invalid json")
	f.Add(`{"nested":{"a":1}}`)
	f.Fuzz(func(t *testing.T, s string) {
		_ = Validate(s)
	})
}

func FuzzParseFilters(f *testing.F) {
	f.Add("metadata[status]=open")
	f.Add("metadata[priority][gt]=3")
	f.Add("metadata[key][invalid]=x")
	f.Add("limit=10&offset=5")
	f.Add("")
	f.Fuzz(func(t *testing.T, query string) {
		r := &http.Request{URL: &url.URL{RawQuery: query}}
		_ = ParseFilters(r)
	})
}

func FuzzSanitizeJSONPath(f *testing.F) {
	f.Add("simple_key")
	f.Add("nested.key")
	f.Add("'; DROP TABLE users--")
	f.Add("")
	f.Add("key with spaces!")
	f.Fuzz(func(t *testing.T, key string) {
		result := sanitizeJSONPath(key)
		// Result should only contain safe characters.
		for _, r := range result {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.') {
				t.Errorf("sanitizeJSONPath returned unsafe character: %q in %q", r, result)
			}
		}
	})
}
