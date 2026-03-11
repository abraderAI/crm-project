package vote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
)

func setupHandler(t *testing.T) (*Handler, *models.Thread) {
	t.Helper()
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	handler := NewHandler(svc)
	thread := seedThread(t, db)
	return handler, thread
}

func makeRequest(t *testing.T, handler http.HandlerFunc, method, path, threadID string, userCtx *auth.UserContext) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", threadID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	if userCtx != nil {
		req = req.WithContext(auth.SetUserContext(req.Context(), userCtx))
	}
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func TestHandler_Toggle_Success(t *testing.T) {
	h, thread := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	w := makeRequest(t, h.Toggle, "POST", "/vote", thread.ID, uc)

	assert.Equal(t, http.StatusOK, w.Code)
	var result VoteResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.True(t, result.Voted)
	assert.Equal(t, 1, result.VoteScore)
}

func TestHandler_Toggle_Unauthenticated(t *testing.T) {
	h, thread := setupHandler(t)
	w := makeRequest(t, h.Toggle, "POST", "/vote", thread.ID, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Toggle_ThreadNotFound(t *testing.T) {
	h, _ := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	w := makeRequest(t, h.Toggle, "POST", "/vote", "nonexistent", uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Toggle_DoubleVote(t *testing.T) {
	h, thread := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}

	// Vote on.
	w := makeRequest(t, h.Toggle, "POST", "/vote", thread.ID, uc)
	assert.Equal(t, http.StatusOK, w.Code)

	// Vote off.
	w = makeRequest(t, h.Toggle, "POST", "/vote", thread.ID, uc)
	assert.Equal(t, http.StatusOK, w.Code)
	var result VoteResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.False(t, result.Voted)
}

func TestHandler_GetWeightTable(t *testing.T) {
	h, _ := setupHandler(t)
	req := httptest.NewRequest("GET", "/vote/weights", nil)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	req = req.WithContext(auth.SetUserContext(req.Context(), uc))
	w := httptest.NewRecorder()
	h.GetWeightTable(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Contains(t, body, "role_weights")
	assert.Contains(t, body, "tier_bonuses")
	assert.Contains(t, body, "default_weight")
}
