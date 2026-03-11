package membership

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

	"github.com/abraderAI/crm-project/api/internal/models"
)

func memberRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	// Org
	r.Post("/orgs/{org}/members", h.AddOrgMember)
	r.Get("/orgs/{org}/members", h.ListOrgMembers)
	r.Patch("/orgs/{org}/members/{userID}", h.UpdateOrgMember)
	r.Delete("/orgs/{org}/members/{userID}", h.RemoveOrgMember)
	// Space
	r.Post("/orgs/{org}/spaces/{space}/members", h.AddSpaceMember)
	r.Get("/orgs/{org}/spaces/{space}/members", h.ListSpaceMembers)
	r.Delete("/orgs/{org}/spaces/{space}/members/{userID}", h.RemoveSpaceMember)
	// Board
	r.Post("/boards/{board}/members", h.AddBoardMember)
	r.Get("/boards/{board}/members", h.ListBoardMembers)
	r.Delete("/boards/{board}/members/{userID}", h.RemoveBoardMember)
	return r
}

func TestHandler_AddOrgMember(t *testing.T) {
	env := setupDB(t)
	h := NewHandler(NewRepository(env.db))
	r := memberRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"user_id":"huser1","role":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("default role", func(t *testing.T) {
		body := `{"user_id":"huser2"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/members", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing user_id", func(t *testing.T) {
		body := `{"role":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role", func(t *testing.T) {
		body := `{"user_id":"huser3","role":"superuser"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate conflict", func(t *testing.T) {
		body := `{"user_id":"huser1","role":"viewer"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestHandler_ListOrgMembers(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddOrgMember(context.Background(), &models.OrgMembership{OrgID: env.orgID, UserID: "lu1", Role: models.RoleAdmin})
	_ = repo.AddOrgMember(context.Background(), &models.OrgMembership{OrgID: env.orgID, UserID: "lu2", Role: models.RoleViewer})

	req := httptest.NewRequest(http.MethodGet, "/orgs/"+env.orgID+"/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 2)
}

func TestHandler_UpdateOrgMember(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddOrgMember(context.Background(), &models.OrgMembership{OrgID: env.orgID, UserID: "upd1", Role: models.RoleViewer})

	t.Run("success", func(t *testing.T) {
		body := `{"role":"admin"}`
		req := httptest.NewRequest(http.MethodPatch, "/orgs/"+env.orgID+"/members/upd1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/orgs/"+env.orgID+"/members/upd1", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role", func(t *testing.T) {
		body := `{"role":"superuser"}`
		req := httptest.NewRequest(http.MethodPatch, "/orgs/"+env.orgID+"/members/upd1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"role":"admin"}`
		req := httptest.NewRequest(http.MethodPatch, "/orgs/"+env.orgID+"/members/nobody", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_RemoveOrgMember(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddOrgMember(context.Background(), &models.OrgMembership{OrgID: env.orgID, UserID: "rm1", Role: models.RoleAdmin})
	_ = repo.AddOrgMember(context.Background(), &models.OrgMembership{OrgID: env.orgID, UserID: "only_owner", Role: models.RoleOwner})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/orgs/"+env.orgID+"/members/rm1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("last owner protected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/orgs/"+env.orgID+"/members/only_owner", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/orgs/"+env.orgID+"/members/nobody", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- Space membership handlers ---

func TestHandler_AddSpaceMember(t *testing.T) {
	env := setupDB(t)
	h := NewHandler(NewRepository(env.db))
	r := memberRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"user_id":"su1","role":"contributor"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing user_id", func(t *testing.T) {
		body := `{"role":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role", func(t *testing.T) {
		body := `{"user_id":"su2","role":"superuser"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("default role", func(t *testing.T) {
		body := `{"user_id":"su3"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestHandler_ListSpaceMembers(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddSpaceMember(context.Background(), &models.SpaceMembership{SpaceID: env.spaceID, UserID: "sl1", Role: models.RoleAdmin})

	req := httptest.NewRequest(http.MethodGet, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RemoveSpaceMember(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddSpaceMember(context.Background(), &models.SpaceMembership{SpaceID: env.spaceID, UserID: "srm1", Role: models.RoleAdmin})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members/srm1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/orgs/"+env.orgID+"/spaces/"+env.spaceID+"/members/nobody", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- Board membership handlers ---

func TestHandler_AddBoardMember(t *testing.T) {
	env := setupDB(t)
	h := NewHandler(NewRepository(env.db))
	r := memberRouter(h)

	t.Run("success", func(t *testing.T) {
		body := `{"user_id":"bu1","role":"contributor"}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+env.boardID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/boards/"+env.boardID+"/members", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing user_id", func(t *testing.T) {
		body := `{"role":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+env.boardID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role", func(t *testing.T) {
		body := `{"user_id":"bu2","role":"superuser"}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+env.boardID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("default role", func(t *testing.T) {
		body := `{"user_id":"bu3"}`
		req := httptest.NewRequest(http.MethodPost, "/boards/"+env.boardID+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestHandler_ListBoardMembers(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddBoardMember(context.Background(), &models.BoardMembership{BoardID: env.boardID, UserID: "bl1", Role: models.RoleAdmin})

	req := httptest.NewRequest(http.MethodGet, "/boards/"+env.boardID+"/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RemoveBoardMember(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	h := NewHandler(repo)
	r := memberRouter(h)

	_ = repo.AddBoardMember(context.Background(), &models.BoardMembership{BoardID: env.boardID, UserID: "brm1", Role: models.RoleAdmin})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/boards/"+env.boardID+"/members/brm1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/boards/"+env.boardID+"/members/nobody", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
