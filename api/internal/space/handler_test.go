package space

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
)

type stubOrgGetter struct{ orgID string }

func (s *stubOrgGetter) ResolveOrgID(_ context.Context, _ string) (string, error) {
	return s.orgID, nil
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
	org := createOrg(t, db, "handler-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"H Space","type":"crm"}`))
	req = chiCtx(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "inv-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid"))
	req = chiCtx(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "list-h-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	_, _ = svc.Create(context.Background(), org.ID, CreateInput{Name: "S1"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "get-h-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	sp, _ := svc.Create(context.Background(), org.ID, CreateInput{Name: "Get Space"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": org.ID, "space": sp.ID})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "nf-h-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": org.ID, "space": "nonexistent"})
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Update(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "upd-h-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	sp, _ := svc.Create(context.Background(), org.ID, CreateInput{Name: "Upd Space"})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"description":"updated"}`))
	req = chiCtx(req, map[string]string{"org": org.ID, "space": sp.ID})
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "upd-inv-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": org.ID, "space": "x"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "del-h-org")
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubOrgGetter{orgID: org.ID})

	sp, _ := svc.Create(context.Background(), org.ID, CreateInput{Name: "Del Space"})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"org": org.ID, "space": sp.ID})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ResolveOrg_Empty(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": ""})
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWriteSpaceError_AllPaths(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrNameRequired, http.StatusBadRequest},
		{ErrInvalidMeta, http.StatusBadRequest},
		{ErrInvalidType, http.StatusBadRequest},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		writeSpaceError(w, tt.err)
		assert.Equal(t, tt.status, w.Code)
	}
}

func TestDecodeCursorID(t *testing.T) {
	assert.Equal(t, "", decodeCursorID(""))
	assert.Equal(t, "", decodeCursorID("invalid"))
}

// Ensure createOrg returns valid model.
func TestCreateOrg_Helper(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "helper-org")
	assert.NotEmpty(t, org.ID)

	var found models.Org
	require.NoError(t, db.First(&found, "id = ?", org.ID).Error)
}
