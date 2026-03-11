package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"key": "value"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "value", body["key"])
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	Created(w, map[string]string{"id": "123"})
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "123", body["id"])
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestListResponse_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	lr := ListResponse{
		Data:     []string{"a", "b"},
		PageInfo: &pagination.PageInfo{HasMore: true, NextCursor: "abc"},
	}
	JSON(w, http.StatusOK, lr)
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	data := body["data"].([]any)
	assert.Len(t, data, 2)
	pi := body["page_info"].(map[string]any)
	assert.Equal(t, true, pi["has_more"])
	assert.Equal(t, "abc", pi["next_cursor"])
}
