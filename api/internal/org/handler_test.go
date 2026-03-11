package org

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

// stubMemberRepo implements MemberRepo for handler tests.
type stubMemberRepo struct{}

func (s *stubMemberRepo) CreateOrgMembership(_ interface{ Value(any) any }, _, _ string, _ models.Role) error {
	return nil
}

func authedCtx(ctx context.Context) context.Context {
	return auth.SetUserContext(ctx, &auth.UserContext{UserID: "test-user", AuthMethod: auth.AuthMethodJWT})
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
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, &stubMemberRepo{})

	body := `{"name":"Handler Org","metadata":"{\"tier\":\"pro\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()

	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "Handler Org", result["name"])
}

func TestHandler_Create_NoAuth(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader(`{"name":"X"}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("invalid"))
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Create_EmptyName(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader(`{"name":""}`))
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateInput{Name: "List Org 1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs", nil)
	req = req.WithContext(authedCtx(req.Context()))
	w := httptest.NewRecorder()

	h.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_List_NoAuth(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Get Handler Org"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+created.ID, nil)
	req = chiCtx(req, map[string]string{"org": created.ID})
	w := httptest.NewRecorder()

	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/nonexistent", nil)
	req = chiCtx(req, map[string]string{"org": "nonexistent"})
	w := httptest.NewRecorder()

	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Get_EmptyParam(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/", nil)
	req = chiCtx(req, map[string]string{"org": ""})
	w := httptest.NewRecorder()

	h.Get(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Update Handler Org"})
	require.NoError(t, err)

	body := `{"description":"updated via handler"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/"+created.ID, strings.NewReader(body))
	req = chiCtx(req, map[string]string{"org": created.ID})
	w := httptest.NewRecorder()

	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_EmptyParam(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/", strings.NewReader(`{}`))
	req = chiCtx(req, map[string]string{"org": ""})
	w := httptest.NewRecorder()

	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPatch, "/v1/orgs/x", strings.NewReader("invalid"))
	req = chiCtx(req, map[string]string{"org": "x"})
	w := httptest.NewRecorder()

	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Delete Handler Org"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+created.ID, nil)
	req = chiCtx(req, map[string]string{"org": created.ID})
	w := httptest.NewRecorder()

	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Delete_EmptyParam(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/", nil)
	req = chiCtx(req, map[string]string{"org": ""})
	w := httptest.NewRecorder()

	h.Delete(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/nonexistent", nil)
	req = chiCtx(req, map[string]string{"org": "nonexistent"})
	w := httptest.NewRecorder()

	h.Delete(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test writeServiceError paths.
func TestWriteServiceError_Paths(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"not found", ErrNotFound, http.StatusNotFound},
		{"name required", ErrNameRequired, http.StatusBadRequest},
		{"invalid meta", ErrInvalidMeta, http.StatusBadRequest},
		{"slug conflict", ErrSlugConflict, http.StatusConflict},
		{"generic", assert.AnError, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeServiceError(w, tt.err)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}
