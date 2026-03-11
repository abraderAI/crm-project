package board

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

func boardRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/spaces/{space}/boards", h.Create)
	r.Get("/spaces/{space}/boards", h.List)
	r.Get("/spaces/{space}/boards/{board}", h.Get)
	r.Patch("/spaces/{space}/boards/{board}", h.Update)
	r.Delete("/spaces/{space}/boards/{board}", h.Delete)
	r.Post("/spaces/{space}/boards/{board}/lock", h.Lock)
	r.Post("/spaces/{space}/boards/{board}/unlock", h.Unlock)
	return r
}

func TestHandler_Create(t *testing.T) {
	db, spaceID := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := boardRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler Board"}`
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Handler Board", resp["name"])
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		body := `{"name":""}`
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_List(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := boardRouter(h)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, spaceID, CreateInput{Name: "HList " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID+"/boards?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 3)
}

func TestHandler_Get(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := boardRouter(h)
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "HGet Board"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID+"/boards/"+b.Slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID+"/boards/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Update(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := boardRouter(h)
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "HUpd Board"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Updated Board"}`
		req := httptest.NewRequest(http.MethodPatch, "/spaces/"+spaceID+"/boards/"+b.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/spaces/"+spaceID+"/boards/"+b.ID, strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"name":"X"}`
		req := httptest.NewRequest(http.MethodPatch, "/spaces/"+spaceID+"/boards/nope", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := boardRouter(h)
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "HDel Board"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/spaces/"+spaceID+"/boards/"+b.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/spaces/"+spaceID+"/boards/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_LockUnlock(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := boardRouter(h)
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "LockBoard"})
	require.NoError(t, err)

	t.Run("lock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards/"+b.ID+"/lock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, true, resp["is_locked"])
	})

	t.Run("unlock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards/"+b.ID+"/unlock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, false, resp["is_locked"])
	})

	t.Run("lock not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards/nope/lock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unlock not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID+"/boards/nope/unlock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
