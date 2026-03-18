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
