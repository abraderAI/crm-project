package org

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func orgRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/v1/orgs", h.Create)
	r.Get("/v1/orgs", h.List)
	r.Get("/v1/orgs/{org}", h.Get)
	r.Patch("/v1/orgs/{org}", h.Update)
	r.Delete("/v1/orgs/{org}", h.Delete)
	return r
}

func TestHandler_Create(t *testing.T) {
	db := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := orgRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler Org","description":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Handler Org", resp["name"])
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		body := `{"name":""}`
		req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_List(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := orgRouter(h)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, CreateInput{Name: "List Org " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 3)
}

func TestHandler_Get(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := orgRouter(h)

	ctx := context.Background()
	org, err := svc.Create(ctx, CreateInput{Name: "Get Org"})
	require.NoError(t, err)

	t.Run("by slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.Slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/orgs/nonexistent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Update(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := orgRouter(h)

	ctx := context.Background()
	org, err := svc.Create(ctx, CreateInput{Name: "Upd Org"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Updated Via Handler"}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/"+org.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/"+org.ID, strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"name":"X"}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/nonexistent", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := orgRouter(h)

	ctx := context.Background()
	org, err := svc.Create(ctx, CreateInput{Name: "Del Org"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+org.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/nonexistent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
