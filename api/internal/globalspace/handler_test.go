package globalspace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// setupDB creates an in-memory SQLite test database with migrations applied
// and a seeded global-support space and board. It returns the db and board ID.
func setupDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))

	// Seed the _system org and global-support space/board.
	org := &models.Org{Name: "System", Slug: "_system", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	sp := &models.Space{
		OrgID:    org.ID,
		Name:     "Support",
		Slug:     "global-support",
		Type:     models.SpaceTypeSupport,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(sp).Error)
	bd := &models.Board{SpaceID: sp.ID, Name: "Support Board", Slug: "support-board", Metadata: "{}"}
	require.NoError(t, db.Create(bd).Error)
	return db, bd.ID
}

// globalSpaceRouter wires up a test chi router for the global-spaces endpoints.
func globalSpaceRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/global-spaces/{space}/threads", h.ListThreads)
	r.Post("/global-spaces/{space}/threads", h.CreateThread)
	r.Get("/global-spaces/{space}/threads/{slug}", h.GetThread)
	r.Patch("/global-spaces/{space}/threads/{slug}", h.UpdateThread)
	return r
}

// withUser attaches a UserContext to the request.
func withUser(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

// TestHandler_ListThreads covers the GET endpoint.
func TestHandler_ListThreads(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := NewService(NewRepository(db))

	// Seed a couple of threads.
	orgID := "org-abc"
	_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Ticket One", OrgID: &orgID})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "user2", CreateInput{Title: "Ticket Two"})
	require.NoError(t, err)

	t.Run("list all", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads", nil)
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 2)
	})

	t.Run("filter mine=true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads?mine=true", nil)
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 1)
	})

	t.Run("filter org_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads?org_id=org-abc", nil)
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 1)
	})

	t.Run("empty result for unknown space", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/no-such-space/threads", nil)
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Empty(t, data)
	})
}

// TestHandler_CreateThread covers the POST endpoint.
func TestHandler_CreateThread(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := globalSpaceRouter(h)

	t.Run("success without org_id", func(t *testing.T) {
		body := `{"title":"New Ticket","body":"Details here"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-tier6")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "New Ticket", resp["title"])
		assert.Equal(t, "user-tier6", resp["author_id"])
	})

	t.Run("success with org_id", func(t *testing.T) {
		body := `{"title":"Org Ticket","org_id":"org-123"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-tier3")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Org Ticket", resp["title"])
		assert.Equal(t, "org-123", resp["org_id"])
	})

	t.Run("missing title returns validation error", func(t *testing.T) {
		body := `{"body":"no title here"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := `{"title":"Anon Ticket"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No withUser call — unauthenticated.
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid request body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads", strings.NewReader("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unknown space returns 500", func(t *testing.T) {
		body := `{"title":"Phantom Ticket"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/no-such-space/threads", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestService_ListThreads covers service-level filtering.
func TestService_ListThreads(t *testing.T) {
	db, _ := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	orgA := "org-a"
	orgB := "org-b"

	_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "User1 Ticket", OrgID: &orgA})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "user2", CreateInput{Title: "User2 OrgB Ticket", OrgID: &orgB})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "User1 No Org Ticket"})
	require.NoError(t, err)

	t.Run("all threads", func(t *testing.T) {
		threads, pi, err := svc.ListThreads(ctx, ListInput{SpaceSlug: "global-support", Params: pagination.Params{Limit: 50}})
		require.NoError(t, err)
		assert.Len(t, threads, 3)
		assert.False(t, pi.HasMore)
	})

	t.Run("mine only", func(t *testing.T) {
		threads, _, err := svc.ListThreads(ctx, ListInput{
			SpaceSlug: "global-support",
			Params:    pagination.Params{Limit: 50},
			UserID:    "user1",
			Mine:      true,
		})
		require.NoError(t, err)
		assert.Len(t, threads, 2)
	})

	t.Run("org scoped", func(t *testing.T) {
		threads, _, err := svc.ListThreads(ctx, ListInput{
			SpaceSlug: "global-support",
			Params:    pagination.Params{Limit: 50},
			OrgID:     orgB,
		})
		require.NoError(t, err)
		assert.Len(t, threads, 1)
		assert.Equal(t, "user2", threads[0].AuthorID)
	})

	t.Run("unknown space returns empty", func(t *testing.T) {
		threads, pi, err := svc.ListThreads(ctx, ListInput{SpaceSlug: "no-such-space", Params: pagination.Params{Limit: 50}})
		require.NoError(t, err)
		assert.Empty(t, threads)
		assert.False(t, pi.HasMore)
	})
}

// TestService_CreateThread covers service-level creation.
func TestService_CreateThread(t *testing.T) {
	db, _ := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	t.Run("success sets thread type to support", func(t *testing.T) {
		th, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Test Ticket"})
		require.NoError(t, err)
		assert.Equal(t, models.ThreadTypeSupport, th.ThreadType)
		assert.NotEmpty(t, th.ID)
		assert.Equal(t, "test-ticket", th.Slug)
	})

	t.Run("empty title returns error", func(t *testing.T) {
		_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{})
		assert.EqualError(t, err, "title is required")
	})

	t.Run("unknown space returns error", func(t *testing.T) {
		_, err := svc.CreateThread(ctx, "no-such-space", "user1", CreateInput{Title: "Orphan"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("duplicate slug gets suffix", func(t *testing.T) {
		t1, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Dup Ticket"})
		require.NoError(t, err)
		t2, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Dup Ticket"})
		require.NoError(t, err)
		assert.NotEqual(t, t1.Slug, t2.Slug)
		assert.Equal(t, "dup-ticket-2", t2.Slug)
	})
}

// TestHandler_GetThread covers the GET /{slug} endpoint.
func TestHandler_GetThread(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := NewService(NewRepository(db))

	// Seed a user shadow so enrichment populates author fields.
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "user-creator",
		Email:       "creator@example.com",
		DisplayName: "Alice Creator",
	}).Error)

	orgID := "org-get-test"
	th, err := svc.CreateThread(ctx, "global-support", "user-creator", CreateInput{Title: "Fetch Me", OrgID: &orgID})
	require.NoError(t, err)

	t.Run("found — returns enriched thread", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/"+th.Slug, nil)
		req = withUser(req, "any-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Fetch Me", resp["title"])
		assert.Equal(t, "creator@example.com", resp["author_email"])
		assert.Equal(t, "Alice Creator", resp["author_name"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/no-such-slug", nil)
		req = withUser(req, "any-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unknown space returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/no-such-space/threads/anything", nil)
		req = withUser(req, "any-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestHandler_UpdateThread covers the PATCH /{slug} endpoint.
func TestHandler_UpdateThread(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := NewService(NewRepository(db))

	th, err := svc.CreateThread(ctx, "global-support", "user-editor", CreateInput{Title: "Editable Ticket", Body: "original body"})
	require.NoError(t, err)

	t.Run("update body", func(t *testing.T) {
		body := `{"body":"updated body"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/"+th.Slug, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-editor")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "updated body", resp["body"])
	})

	t.Run("update status", func(t *testing.T) {
		body := `{"status":"resolved"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/"+th.Slug, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "agent-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "resolved", resp["status"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		body := `{"body":"whatever"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/no-such-slug", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-editor")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := `{"status":"closed"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/"+th.Slug, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/"+th.Slug, strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-editor")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestService_Enrichment verifies that author_email/author_name/org_name are populated.
func TestService_Enrichment(t *testing.T) {
	db, _ := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	// Seed a user shadow and a real org.
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "user-enrich",
		Email:       "enrich@deft.co",
		DisplayName: "Enrich User",
	}).Error)
	org := &models.Org{Name: "Acme Corp", Slug: "acme", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	th, err := svc.CreateThread(ctx, "global-support", "user-enrich", CreateInput{
		Title: "Enriched Ticket",
		OrgID: &org.ID,
	})
	require.NoError(t, err)

	t.Run("list returns author and org info", func(t *testing.T) {
		threads, _, err := svc.ListThreads(ctx, ListInput{SpaceSlug: "global-support", Params: pagination.Params{Limit: 50}})
		require.NoError(t, err)
		// Find the enriched ticket.
		var found *ThreadWithAuthor
		for i := range threads {
			if threads[i].ID == th.ID {
				found = &threads[i]
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "enrich@deft.co", found.AuthorEmail)
		assert.Equal(t, "Enrich User", found.AuthorName)
		assert.Equal(t, "Acme Corp", found.OrgName)
	})

	t.Run("get returns author and org info", func(t *testing.T) {
		rich, err := svc.GetThread(ctx, "global-support", th.Slug)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "enrich@deft.co", rich.AuthorEmail)
		assert.Equal(t, "Acme Corp", rich.OrgName)
	})

	t.Run("get returns nil for unknown slug", func(t *testing.T) {
		rich, err := svc.GetThread(ctx, "global-support", "does-not-exist")
		require.NoError(t, err)
		assert.Nil(t, rich)
	})
}

// TestService_UpdateThread covers service-level update behavior.
func TestService_UpdateThread(t *testing.T) {
	db, _ := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "user-u", CreateInput{Title: "Update Me", Body: "old"})
	require.NoError(t, err)

	t.Run("update body", func(t *testing.T) {
		newBody := "new body"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "user-u", UpdateInput{Body: &newBody})
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "new body", rich.Body)
	})

	t.Run("update status via metadata merge", func(t *testing.T) {
		status := "pending"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "agent", UpdateInput{Status: &status})
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "pending", rich.Status)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		body := "x"
		rich, err := svc.UpdateThread(ctx, "global-support", "no-such-slug", "user-u", UpdateInput{Body: &body})
		require.NoError(t, err)
		assert.Nil(t, rich)
	})

	t.Run("unknown space returns nil", func(t *testing.T) {
		body := "x"
		rich, err := svc.UpdateThread(ctx, "no-such-space", th.Slug, "user-u", UpdateInput{Body: &body})
		require.NoError(t, err)
		assert.Nil(t, rich)
	})
}

// TestThreadTypeForSpace covers the space-to-thread-type mapping.
func TestThreadTypeForSpace(t *testing.T) {
	tests := []struct {
		slug string
		want models.ThreadType
	}{
		{"global-support", models.ThreadTypeSupport},
		{"global-leads", models.ThreadTypeLead},
		{"global-forum", models.ThreadTypeForum},
		{"global-docs", models.ThreadTypeForum},
		{"unknown-space", models.ThreadTypeForum},
	}
	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			assert.Equal(t, tt.want, threadTypeForSpace(tt.slug))
		})
	}
}
