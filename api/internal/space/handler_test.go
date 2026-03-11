package space

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

func spaceRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/spaces", h.Create)
	r.Get("/v1/orgs/{org}/spaces", h.List)
	r.Get("/v1/orgs/{org}/spaces/{space}", h.Get)
	r.Patch("/v1/orgs/{org}/spaces/{space}", h.Update)
	r.Delete("/v1/orgs/{org}/spaces/{space}", h.Delete)
	return r
}

func TestHandler_Create(t *testing.T) {
	db, orgID := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := spaceRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler Space","type":"general"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+orgID+"/spaces", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Handler Space", resp["name"])
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+orgID+"/spaces", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		body := `{"name":""}`
		req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+orgID+"/spaces", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_List(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := spaceRouter(h)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, orgID, CreateInput{Name: "HList " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/spaces?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 3)
}

func TestHandler_Get(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := spaceRouter(h)
	ctx := context.Background()

	sp, err := svc.Create(ctx, orgID, CreateInput{Name: "HGet Space"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/spaces/"+sp.Slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/spaces/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Update(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := spaceRouter(h)
	ctx := context.Background()

	sp, err := svc.Create(ctx, orgID, CreateInput{Name: "HUpd Space"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Updated Space"}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/"+orgID+"/spaces/"+sp.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/"+orgID+"/spaces/"+sp.ID, strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"name":"X"}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/"+orgID+"/spaces/nope", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := spaceRouter(h)
	ctx := context.Background()

	sp, err := svc.Create(ctx, orgID, CreateInput{Name: "HDel Space"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+orgID+"/spaces/"+sp.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+orgID+"/spaces/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
