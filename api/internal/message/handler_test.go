package message

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/abraderAI/crm-project/api/internal/auth"
)

type stubThreadGetter struct{ threadID string }

func (s *stubThreadGetter) ResolveThreadID(_ context.Context, _, _, _, _ string) (string, error) {
	return s.threadID, nil
}

func chiCtx(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func authedCtx(ctx context.Context) context.Context {
	return auth.SetUserContext(ctx, &auth.UserContext{UserID: "test-user", AuthMethod: auth.AuthMethodJWT})
}

func TestHandler_Create(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"body":"Hello"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t"})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_NoAuth(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"body":"X"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t"})
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t"})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	_, _ = svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "M1"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t"})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	m, _ := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "Get Msg"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t", "message": m.ID})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t", "message": "nonexistent"})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Update(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	m, _ := svc.Create(context.Background(), th.ID, "test-user", CreateInput{Body: "Original"})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"body":"updated"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t", "message": m.ID})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_NoAuth(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t", "message": "x"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t", "message": "x"})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	h := NewHandler(svc, &stubThreadGetter{threadID: th.ID})

	m, _ := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "Del Msg"})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "t", "message": m.ID})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ResolveThread_EmptyParams(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "", "space": "", "board": "", "thread": ""})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWriteMessageError_AllPaths(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrBodyRequired, http.StatusBadRequest},
		{ErrInvalidMeta, http.StatusBadRequest},
		{ErrInvalidType, http.StatusBadRequest},
		{ErrThreadLocked, http.StatusConflict},
		{ErrNotAuthor, http.StatusForbidden},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		writeMessageError(w, tt.err)
		assert.Equal(t, tt.status, w.Code)
	}
}

func TestDecodeCursorID(t *testing.T) {
	assert.Equal(t, "", decodeCursorID(""))
	assert.Equal(t, "", decodeCursorID("invalid"))
}
