package message

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
	"github.com/abraderAI/crm-project/api/internal/models"
)

func msgRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/threads/{thread}/messages", h.Create)
	r.Get("/threads/{thread}/messages", h.List)
	r.Get("/threads/{thread}/messages/{message}", h.Get)
	r.Patch("/threads/{thread}/messages/{message}", h.Update)
	r.Delete("/threads/{thread}/messages/{message}", h.Delete)
	return r
}

func withUser(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

func TestHandler_Create(t *testing.T) {
	db, threadID := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := msgRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"body":"Hello","type":"comment"}`
		req := httptest.NewRequest(http.MethodPost, "/threads/"+threadID+"/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Hello", resp["body"])
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/threads/"+threadID+"/messages", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		body := `{"body":""}`
		req := httptest.NewRequest(http.MethodPost, "/threads/"+threadID+"/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_List(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := msgRouter(h)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Msg " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/threads/"+threadID+"/messages?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 3)
}

func TestHandler_Get(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := msgRouter(h)
	ctx := context.Background()

	msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "HGet Msg", Type: models.MessageTypeComment})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/threads/"+threadID+"/messages/"+msg.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/threads/"+threadID+"/messages/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Update(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := msgRouter(h)
	ctx := context.Background()

	msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Original"})
	require.NoError(t, err)

	t.Run("author update", func(t *testing.T) {
		body := `{"body":"Updated"}`
		req := httptest.NewRequest(http.MethodPatch, "/threads/"+threadID+"/messages/"+msg.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("non-author forbidden", func(t *testing.T) {
		body := `{"body":"Hacked"}`
		req := httptest.NewRequest(http.MethodPatch, "/threads/"+threadID+"/messages/"+msg.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/threads/"+threadID+"/messages/"+msg.ID, strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"body":"X"}`
		req := httptest.NewRequest(http.MethodPatch, "/threads/"+threadID+"/messages/nope", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	r := msgRouter(h)
	ctx := context.Background()

	msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Del me"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/threads/"+threadID+"/messages/"+msg.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/threads/"+threadID+"/messages/nope", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
