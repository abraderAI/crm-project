package membership

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chiCtx(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- Org Membership Handlers ---

func TestHandler_AddOrgMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"user_id":"u1","role":"viewer"}`))
	req = chiCtx(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.AddOrgMember(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddOrgMember_InvalidBody(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.AddOrgMember(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListOrgMembers(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	_ = svc.AddOrgMember(context.Background(), org.ID, MemberInput{UserID: "u1", Role: "viewer"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.ListOrgMembers(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateOrgMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	ctx := context.Background()

	_ = svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "u1", Role: "viewer"})
	members, _ := svc.ListOrgMembers(ctx, org.ID)
	require.Len(t, members, 1)

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"admin"}`))
	req = chiCtx(req, map[string]string{"org": org.ID, "id": members[0].ID})
	w := httptest.NewRecorder()
	h.UpdateOrgMember(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateOrgMember_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"org": "o", "id": "x"})
	w := httptest.NewRecorder()
	h.UpdateOrgMember(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemoveOrgMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	ctx := context.Background()

	_ = svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "u1", Role: "viewer"})
	members, _ := svc.ListOrgMembers(ctx, org.ID)
	require.Len(t, members, 1)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"org": org.ID, "id": members[0].ID})
	w := httptest.NewRecorder()
	h.RemoveOrgMember(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- Space Membership Handlers ---

func TestHandler_AddSpaceMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"user_id":"u1","role":"viewer"}`))
	req = chiCtx(req, map[string]string{"space": sp.ID})
	w := httptest.NewRecorder()
	h.AddSpaceMember(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddSpaceMember_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"space": "x"})
	w := httptest.NewRecorder()
	h.AddSpaceMember(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListSpaceMembers(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	_ = svc.AddSpaceMember(context.Background(), sp.ID, MemberInput{UserID: "u1", Role: "viewer"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"space": sp.ID})
	w := httptest.NewRecorder()
	h.ListSpaceMembers(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateSpaceMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	ctx := context.Background()

	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "u1", Role: "viewer"})
	members, _ := svc.ListSpaceMembers(ctx, sp.ID)
	require.Len(t, members, 1)

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"admin"}`))
	req = chiCtx(req, map[string]string{"space": sp.ID, "id": members[0].ID})
	w := httptest.NewRecorder()
	h.UpdateSpaceMember(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateSpaceMember_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"space": "x", "id": "x"})
	w := httptest.NewRecorder()
	h.UpdateSpaceMember(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemoveSpaceMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	ctx := context.Background()

	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "u1", Role: "viewer"})
	members, _ := svc.ListSpaceMembers(ctx, sp.ID)
	require.Len(t, members, 1)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"space": sp.ID, "id": members[0].ID})
	w := httptest.NewRecorder()
	h.RemoveSpaceMember(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- Board Membership Handlers ---

func TestHandler_AddBoardMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"user_id":"u1","role":"viewer"}`))
	req = chiCtx(req, map[string]string{"board": b.ID})
	w := httptest.NewRecorder()
	h.AddBoardMember(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddBoardMember_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"board": "x"})
	w := httptest.NewRecorder()
	h.AddBoardMember(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListBoardMembers(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	_ = svc.AddBoardMember(context.Background(), b.ID, MemberInput{UserID: "u1", Role: "viewer"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = chiCtx(req, map[string]string{"board": b.ID})
	w := httptest.NewRecorder()
	h.ListBoardMembers(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateBoardMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	ctx := context.Background()

	_ = svc.AddBoardMember(ctx, b.ID, MemberInput{UserID: "u1", Role: "viewer"})
	members, _ := svc.ListBoardMembers(ctx, b.ID)
	require.Len(t, members, 1)

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"admin"}`))
	req = chiCtx(req, map[string]string{"board": b.ID, "id": members[0].ID})
	w := httptest.NewRecorder()
	h.UpdateBoardMember(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateBoardMember_InvalidBody(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("bad"))
	req = chiCtx(req, map[string]string{"board": "x", "id": "x"})
	w := httptest.NewRecorder()
	h.UpdateBoardMember(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemoveBoardMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	h := NewHandler(svc)
	ctx := context.Background()

	_ = svc.AddBoardMember(ctx, b.ID, MemberInput{UserID: "u1", Role: "viewer"})
	members, _ := svc.ListBoardMembers(ctx, b.ID)
	require.Len(t, members, 1)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = chiCtx(req, map[string]string{"board": b.ID, "id": members[0].ID})
	w := httptest.NewRecorder()
	h.RemoveBoardMember(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- Error mapping ---

func TestWriteMemberError_AllPaths(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrInvalidRole, http.StatusBadRequest},
		{ErrUserRequired, http.StatusBadRequest},
		{ErrLastOwner, http.StatusConflict},
		{ErrAlreadyExists, http.StatusConflict},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		writeMemberError(w, tt.err)
		assert.Equal(t, tt.status, w.Code)
	}
}
