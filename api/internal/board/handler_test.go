package board

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type stubSpaceGetter struct{ spaceID string }

func (s *stubSpaceGetter) ResolveSpaceID(_ context.Context, _, _ string) (string, error) {
	return s.spaceID, nil
}

func chiCtx(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestHandler_Create(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"H Board"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s"})
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s"})
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	_, _ = svc.Create(context.Background(), sp.ID, CreateInput{Name: "B1"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s"})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	b, _ := svc.Create(context.Background(), sp.ID, CreateInput{Name: "Get Board"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": b.ID})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "nonexistent"})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Update(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	b, _ := svc.Create(context.Background(), sp.ID, CreateInput{Name: "Upd Board"})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"description":"updated"}`))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": b.ID})
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": "x"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Lock(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	b, _ := svc.Create(context.Background(), sp.ID, CreateInput{Name: "Lock Board"})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": b.ID})
	w := httptest.NewRecorder()
	h.Lock(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Unlock(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	b, _ := svc.Create(context.Background(), sp.ID, CreateInput{Name: "Unlock Board"})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": b.ID})
	w := httptest.NewRecorder()
	h.Unlock(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Delete(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubSpaceGetter{spaceID: sp.ID})

	b, _ := svc.Create(context.Background(), sp.ID, CreateInput{Name: "Del Board"})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"org": "o", "space": "s", "board": b.ID})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ResolveSpace_EmptyParams(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": "", "space": ""})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWriteBoardError_AllPaths(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrNameRequired, http.StatusBadRequest},
		{ErrInvalidMeta, http.StatusBadRequest},
		{ErrBoardLocked, http.StatusConflict},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		writeBoardError(w, tt.err)
		assert.Equal(t, tt.status, w.Code)
	}
}

func TestDecodeCursorID(t *testing.T) {
	assert.Equal(t, "", decodeCursorID(""))
	assert.Equal(t, "", decodeCursorID("invalid"))
}
