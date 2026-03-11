package thread

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

type stubBoardGetter struct{ boardID string }

func (s *stubBoardGetter) ResolveBoardID(_ context.Context, _, _, _ string) (string, error) {
	return s.boardID, nil
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
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"H Thread"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b"})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_NoAuth(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"X"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b"})
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b"})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	_, _ = svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "T1"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b"})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Get Thread"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "nonexistent"})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Update(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "test-user", CreateInput{Title: "Upd Thread"})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"body":"updated"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_NoAuth(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "x"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": "x"})
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Pin(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Pin Thread"})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	w := httptest.NewRecorder()
	h.Pin(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Unpin(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Unpin Thread"})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	w := httptest.NewRecorder()
	h.Unpin(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Lock(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Lock Thread"})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	w := httptest.NewRecorder()
	h.Lock(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Unlock(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Unlock Thread"})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	w := httptest.NewRecorder()
	h.Unlock(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Delete(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	h := NewHandler(svc, &stubBoardGetter{boardID: board.ID})

	th, _ := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Del Thread"})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "b", "thread": th.ID})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ResolveBoard_EmptyParams(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db), nil)
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "", "space": "", "board": ""})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWriteThreadError_AllPaths(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrTitleRequired, http.StatusBadRequest},
		{ErrInvalidMeta, http.StatusBadRequest},
		{ErrBoardLocked, http.StatusConflict},
		{ErrThreadLocked, http.StatusConflict},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		writeThreadError(w, tt.err)
		assert.Equal(t, tt.status, w.Code)
	}
}

func TestDecodeCursorID(t *testing.T) {
	assert.Equal(t, "", decodeCursorID(""))
	assert.Equal(t, "", decodeCursorID("invalid"))
}
