package thread

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

	"github.com/abraderAI/crm-project/api/internal/auth"
)

func threadRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/boards/{board}/threads", h.Create)
	r.Get("/boards/{board}/threads", h.List)
	r.Get("/boards/{board}/threads/{thread}", h.Get)
	r.Patch("/boards/{board}/threads/{thread}", h.Update)
	r.Delete("/boards/{board}/threads/{thread}", h.Delete)
	r.Post("/boards/{board}/threads/{thread}/pin", h.Pin)
	r.Post("/boards/{board}/threads/{thread}/unpin", h.Unpin)
	r.Post("/boards/{board}/threads/{thread}/lock", h.Lock)
	r.Post("/boards/{board}/threads/{thread}/unlock", h.Unlock)
	return r
}

func withUser(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

func TestHandler_Create(t *testing.T) {
	db, boardID := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := threadRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"title":"Handler Thread","body":"content"}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Handler Thread", resp["title"])
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		body := `{"title":""}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_CreateWithBoardCheck(t *testing.T) {
	db, boardID := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))

	r := chi.NewRouter()
	r.Post("/boards/{board}/threads", h.CreateWithBoardCheck(true))

	t.Run("board locked", func(t *testing.T) {
		body := `{"title":"Blocked","body":"content"}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("board locked invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_List(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "HList " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/threads?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 3)
}

func TestHandler_ListWithMetadataFilters(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	_, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Open", Metadata: `{"status":"open"}`})
	require.NoError(t, err)
	_, err = svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Closed", Metadata: `{"status":"closed"}`})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/threads?metadata[status]=open", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "HGet Thread"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/threads/"+th.Slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/threads/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Update(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "HUpd Thread"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		body := `{"title":"Updated"}`
		req := httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/threads/"+th.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/threads/"+th.ID, strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"title":"X"}`
		req := httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/threads/nope", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "HDel Thread"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/boards/"+boardID+"/threads/"+th.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/boards/"+boardID+"/threads/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_PinUnpin(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "PinThread"})
	require.NoError(t, err)

	t.Run("pin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/"+th.ID+"/pin", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, true, resp["is_pinned"])
	})

	t.Run("unpin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/"+th.ID+"/unpin", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, false, resp["is_pinned"])
	})

	t.Run("pin not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/nope/pin", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unpin not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/nope/unpin", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_LockUnlock(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := threadRouter(h)
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "LockThread"})
	require.NoError(t, err)

	t.Run("lock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/"+th.ID+"/lock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unlock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/"+th.ID+"/unlock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("lock not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/nope/lock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unlock not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/threads/nope/unlock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
