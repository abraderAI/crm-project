package moderation

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

func setupHandler(t *testing.T) (*Handler, *testHierarchy) {
	t.Helper()
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	handler := NewHandler(svc)
	h := seedHierarchy(t, db)
	return handler, h
}

func makeReq(t *testing.T, handler http.HandlerFunc, method, path, body string, params map[string]string, userCtx *auth.UserContext) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	if userCtx != nil {
		req = req.WithContext(auth.SetUserContext(req.Context(), userCtx))
	}
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func TestHandler_CreateFlag_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	body := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	w := makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusCreated, w.Code)

	var flag models.Flag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&flag))
	assert.Equal(t, models.FlagStatusOpen, flag.Status)
}

func TestHandler_CreateFlag_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	body := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	w := makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateFlag_MissingReason(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	body := `{"thread_id":"` + hier.thread.ID + `"}`
	w := makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFlag_BadBody(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.CreateFlag, "POST", "/flags", "not json", map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListFlags(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}

	// Create a flag first via the handler.
	body := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, uc)

	w := makeReq(t, h.ListFlags, "GET", "/flags", "", map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	data := result["data"].([]any)
	assert.GreaterOrEqual(t, len(data), 1)
}

func TestHandler_ResolveFlag_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}

	// Create a flag.
	createBody := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	cw := makeReq(t, h.CreateFlag, "POST", "/flags", createBody, map[string]string{"org": hier.org.ID}, uc)
	var created models.Flag
	require.NoError(t, json.NewDecoder(cw.Body).Decode(&created))

	// Resolve.
	w := makeReq(t, h.ResolveFlag, "POST", "/flags/"+created.ID+"/resolve", "", map[string]string{"org": hier.org.ID, "flag": created.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)
	var resolved models.Flag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resolved))
	assert.Equal(t, models.FlagStatusResolved, resolved.Status)
}

func TestHandler_ResolveFlag_NotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.ResolveFlag, "POST", "/flags/nonexistent/resolve", "", map[string]string{"org": hier.org.ID, "flag": "nonexistent"}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_DismissFlag_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}

	createBody := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	cw := makeReq(t, h.CreateFlag, "POST", "/flags", createBody, map[string]string{"org": hier.org.ID}, uc)
	var created models.Flag
	require.NoError(t, json.NewDecoder(cw.Body).Decode(&created))

	w := makeReq(t, h.DismissFlag, "POST", "/flags/"+created.ID+"/dismiss", "", map[string]string{"org": hier.org.ID, "flag": created.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)
	var dismissed models.Flag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&dismissed))
	assert.Equal(t, models.FlagStatusDismissed, dismissed.Status)
}

func TestHandler_MoveThread_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_board_id":"` + hier.board2.ID + `"}`
	w := makeReq(t, h.MoveThread, "POST", "/move", body, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_MoveThread_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	body := `{"target_board_id":"` + hier.board2.ID + `"}`
	w := makeReq(t, h.MoveThread, "POST", "/move", body, map[string]string{"thread": hier.thread.ID}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_MoveThread_BadBody(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.MoveThread, "POST", "/move", "not json", map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MergeThread_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_thread_id":"` + hier.thread2.ID + `"}`
	w := makeReq(t, h.MergeThread, "POST", "/merge", body, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_MergeThread_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	body := `{"target_thread_id":"` + hier.thread2.ID + `"}`
	w := makeReq(t, h.MergeThread, "POST", "/merge", body, map[string]string{"thread": hier.thread.ID}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_HideThread_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.HideThread, "POST", "/hide", "", map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HideThread_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	w := makeReq(t, h.HideThread, "POST", "/hide", "", map[string]string{"thread": hier.thread.ID}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UnhideThread_Success(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	// Hide first.
	makeReq(t, h.HideThread, "POST", "/hide", "", map[string]string{"thread": hier.thread.ID}, uc)
	// Unhide.
	w := makeReq(t, h.UnhideThread, "POST", "/unhide", "", map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UnhideThread_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	w := makeReq(t, h.UnhideThread, "POST", "/unhide", "", map[string]string{"thread": hier.thread.ID}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_HideThread_NotFound(t *testing.T) {
	h, _ := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.HideThread, "POST", "/hide", "", map[string]string{"thread": "nonexistent"}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UnhideThread_NotFound(t *testing.T) {
	h, _ := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.UnhideThread, "POST", "/unhide", "", map[string]string{"thread": "nonexistent"}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ResolveFlag_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	w := makeReq(t, h.ResolveFlag, "POST", "/flags/x/resolve", "", map[string]string{"org": hier.org.ID, "flag": "x"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DismissFlag_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	w := makeReq(t, h.DismissFlag, "POST", "/flags/x/dismiss", "", map[string]string{"org": hier.org.ID, "flag": "x"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DismissFlag_NotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.DismissFlag, "POST", "/flags/nonexistent/dismiss", "", map[string]string{"org": hier.org.ID, "flag": "nonexistent"}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_MoveThread_NotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_board_id":"` + hier.board2.ID + `"}`
	w := makeReq(t, h.MoveThread, "POST", "/move", body, map[string]string{"thread": "nonexistent"}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_MoveThread_SameBoard(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_board_id":"` + hier.board.ID + `"}`
	w := makeReq(t, h.MoveThread, "POST", "/move", body, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MoveThread_MissingTarget(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.MoveThread, "POST", "/move", `{}`, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MergeThread_NotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_thread_id":"nonexistent"}`
	w := makeReq(t, h.MergeThread, "POST", "/merge", body, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_MergeThread_SelfMerge(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_thread_id":"` + hier.thread.ID + `"}`
	w := makeReq(t, h.MergeThread, "POST", "/merge", body, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MergeThread_BadBody(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.MergeThread, "POST", "/merge", "not json", map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MergeThread_MissingTarget(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	w := makeReq(t, h.MergeThread, "POST", "/merge", `{}`, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MoveThread_TargetBoardNotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_board_id":"nonexistent"}`
	w := makeReq(t, h.MoveThread, "POST", "/move", body, map[string]string{"thread": hier.thread.ID}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CreateFlag_ThreadNotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	body := `{"thread_id":"nonexistent","reason":"spam"}`
	w := makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ResolveFlag_AlreadyResolved(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}

	createBody := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	cw := makeReq(t, h.CreateFlag, "POST", "/flags", createBody, map[string]string{"org": hier.org.ID}, uc)
	var created models.Flag
	require.NoError(t, json.NewDecoder(cw.Body).Decode(&created))

	// Resolve first time.
	makeReq(t, h.ResolveFlag, "POST", "/flags/"+created.ID+"/resolve", "", map[string]string{"org": hier.org.ID, "flag": created.ID}, uc)

	// Resolve second time — should fail.
	w := makeReq(t, h.ResolveFlag, "POST", "/flags/"+created.ID+"/resolve", "", map[string]string{"org": hier.org.ID, "flag": created.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DismissFlag_AlreadyDismissed(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}

	createBody := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
	cw := makeReq(t, h.CreateFlag, "POST", "/flags", createBody, map[string]string{"org": hier.org.ID}, uc)
	var created models.Flag
	require.NoError(t, json.NewDecoder(cw.Body).Decode(&created))

	// Dismiss first time.
	makeReq(t, h.DismissFlag, "POST", "/flags/"+created.ID+"/dismiss", "", map[string]string{"org": hier.org.ID, "flag": created.ID}, uc)

	// Dismiss second time — should fail.
	w := makeReq(t, h.DismissFlag, "POST", "/flags/"+created.ID+"/dismiss", "", map[string]string{"org": hier.org.ID, "flag": created.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MergeThread_SourceNotFound(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "mod1", AuthMethod: auth.AuthMethodJWT}
	body := `{"target_thread_id":"` + hier.thread2.ID + `"}`
	w := makeReq(t, h.MergeThread, "POST", "/merge", body, map[string]string{"thread": "nonexistent"}, uc)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CreateFlag_MissingThreadID(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}
	body := `{"reason":"spam"}`
	w := makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListFlags_Unauthenticated(t *testing.T) {
	h, hier := setupHandler(t)
	w := makeReq(t, h.ListFlags, "GET", "/flags", "", map[string]string{"org": hier.org.ID}, nil)
	// ListFlags does not require auth, should return 200.
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListFlags_WithPagination(t *testing.T) {
	h, hier := setupHandler(t)
	uc := &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT}

	// Create 3 flags.
	for i := 0; i < 3; i++ {
		body := `{"thread_id":"` + hier.thread.ID + `","reason":"spam"}`
		makeReq(t, h.CreateFlag, "POST", "/flags", body, map[string]string{"org": hier.org.ID}, uc)
	}

	w := makeReq(t, h.ListFlags, "GET", "/flags?limit=2", "", map[string]string{"org": hier.org.ID}, uc)
	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	data := result["data"].([]any)
	assert.GreaterOrEqual(t, len(data), 1)
}
