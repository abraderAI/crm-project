package globalspace

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

type stubTicketNumberer struct {
	called bool
	orgID  string
}

func (s *stubTicketNumberer) AssignTicketNumber(_ context.Context, _ *models.Thread, orgID string) error {
	s.called = true
	s.orgID = orgID
	return nil
}

func createTempMultipartFile(t *testing.T, name, content string) multipart.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

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

// newSvc creates a Service with no event bus or upload service for tests.
func newSvc(db *gorm.DB) *Service {
	return NewService(NewRepository(db), nil, nil)
}

// newUploadSvc creates an upload.Service backed by a temp directory for tests.
func newUploadSvc(t *testing.T, db *gorm.DB) *upload.Service {
	t.Helper()
	storage, err := upload.NewLocalStorage(t.TempDir())
	require.NoError(t, err)
	return upload.NewService(db, storage, 10<<20) // 10 MB limit
}

// globalSpaceRouter wires up a test chi router for the global-spaces endpoints.
func globalSpaceRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/global-spaces/{space}/threads", h.ListThreads)
	r.Post("/global-spaces/{space}/threads", h.CreateThread)
	r.Get("/global-spaces/{space}/threads/{slug}", h.GetThread)
	r.Patch("/global-spaces/{space}/threads/{slug}", h.UpdateThread)
	r.Get("/global-spaces/{space}/threads/{slug}/attachments", h.ListAttachments)
	r.Post("/global-spaces/{space}/threads/{slug}/attachments", h.UploadAttachment)
	return r
}

// withUser attaches a UserContext to the request.
func withUser(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

// seedDeftMember creates the DEFT org (if needed) and adds the user as a member.
func seedDeftMember(t *testing.T, db *gorm.DB, userID string) {
	t.Helper()
	var org models.Org
	if err := db.Where("slug = ?", "deft").First(&org).Error; err != nil {
		org = models.Org{Name: "DEFT", Slug: "deft", Metadata: "{}"}
		require.NoError(t, db.Create(&org).Error)
	}
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: userID, Role: models.RoleContributor,
	}).Error)
}

// seedOrgMember creates an org and adds the user as a member, returning the org ID.
func seedOrgMember(t *testing.T, db *gorm.DB, orgName, orgSlug, userID string) string {
	t.Helper()
	org := &models.Org{Name: orgName, Slug: orgSlug, Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: userID, Role: models.RoleContributor,
	}).Error)
	return org.ID
}

// TestHandler_ListThreads covers the GET endpoint.
func TestHandler_ListThreads(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	// Make user1 a DEFT member so they can see all tickets.
	seedDeftMember(t, db, "user1")

	// Seed a couple of threads.
	orgID := "org-abc"
	_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Ticket One", OrgID: &orgID})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "user2", CreateInput{Title: "Ticket Two"})
	require.NoError(t, err)

	t.Run("list all as deft member", func(t *testing.T) {
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
	h := NewHandler(newSvc(db))
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
	svc := newSvc(db)
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
	svc := newSvc(db)
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

	t.Run("support ticket with body creates initial customer message entry", func(t *testing.T) {
		th, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{
			Title: "Initial Body Ticket",
			Body:  "<p>First customer message</p>",
		})
		require.NoError(t, err)

		var msgs []models.Message
		require.NoError(t, db.Where("thread_id = ?", th.ID).Order("created_at ASC").Find(&msgs).Error)
		require.Len(t, msgs, 1)
		assert.Equal(t, models.MessageTypeCustomer, msgs[0].Type)
		assert.Equal(t, "<p>First customer message</p>", msgs[0].Body)
		assert.True(t, msgs[0].IsPublished)
		assert.True(t, msgs[0].IsImmutable)
	})
}

// TestHandler_GetThread covers the GET /{slug} endpoint.
func TestHandler_GetThread(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

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
		req = withUser(req, "user-creator")
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
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

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
		req = withUser(req, "user-editor")
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
	svc := newSvc(db)
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
		rich, err := svc.GetThread(ctx, "global-support", th.Slug, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "enrich@deft.co", rich.AuthorEmail)
		assert.Equal(t, "Acme Corp", rich.OrgName)
	})

	t.Run("get returns nil for unknown slug", func(t *testing.T) {
		rich, err := svc.GetThread(ctx, "global-support", "does-not-exist", nil)
		require.NoError(t, err)
		assert.Nil(t, rich)
	})
}

// TestService_UpdateThread covers service-level update behavior.
func TestService_UpdateThread(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "user-u", CreateInput{Title: "Update Me", Body: "old"})
	require.NoError(t, err)

	t.Run("update body", func(t *testing.T) {
		newBody := "new body"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "user-u", UpdateInput{Body: &newBody}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "new body", rich.Body)
	})

	t.Run("update status via metadata merge", func(t *testing.T) {
		status := "pending"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "agent", UpdateInput{Status: &status}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "pending", rich.Status)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		body := "x"
		rich, err := svc.UpdateThread(ctx, "global-support", "no-such-slug", "user-u", UpdateInput{Body: &body}, nil)
		require.NoError(t, err)
		assert.Nil(t, rich)
	})

	t.Run("unknown space returns nil", func(t *testing.T) {
		body := "x"
		rich, err := svc.UpdateThread(ctx, "no-such-space", th.Slug, "user-u", UpdateInput{Body: &body}, nil)
		require.NoError(t, err)
		assert.Nil(t, rich)
	})
}

// TestHandler_ListAttachments covers the GET /{slug}/attachments endpoint.
func TestHandler_ListAttachments(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	th, err := svc.CreateThread(ctx, "global-support", "user-attach", CreateInput{Title: "Attachable Ticket"})
	require.NoError(t, err)

	t.Run("empty attachments returns 200 with empty array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/"+th.Slug+"/attachments", nil)
		req = withUser(req, "user-attach")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp []any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Empty(t, resp)
	})

	t.Run("not found thread returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/no-such-slug/attachments", nil)
		req = withUser(req, "any-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unknown space returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/no-such-space/threads/anything/attachments", nil)
		req = withUser(req, "any-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestHandler_UploadAttachment covers POST /{slug}/attachments.
func TestHandler_UploadAttachment(t *testing.T) {
	db, _ := setupDB(t)
	upSvc := newUploadSvc(t, db)
	svc := NewService(NewRepository(db), nil, upSvc)
	h := NewHandler(svc)
	r := globalSpaceRouter(h)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "user-up", CreateInput{Title: "Upload Ticket"})
	require.NoError(t, err)

	// buildMultipart creates a multipart/form-data body with a single text file.
	buildMultipart := func(t *testing.T, filename, content string) (*bytes.Buffer, string) {
		t.Helper()
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("file", filename)
		require.NoError(t, err)
		_, err = fw.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, mw.Close())
		return &buf, mw.FormDataContentType()
	}

	t.Run("success returns 201 with upload record", func(t *testing.T) {
		body, ct := buildMultipart(t, "notes.txt", "some content")
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/attachments", body)
		req.Header.Set("Content-Type", ct)
		req = withUser(req, "user-up")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "notes.txt", resp["filename"])
		assert.Equal(t, "thread", resp["entity_type"])
	})

	t.Run("unknown thread returns 404", func(t *testing.T) {
		body, ct := buildMultipart(t, "x.txt", "x")
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/no-such/attachments", body)
		req.Header.Set("Content-Type", ct)
		req = withUser(req, "user-up")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("missing file field returns 400", func(t *testing.T) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		require.NoError(t, mw.Close())
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/attachments", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req = withUser(req, "user-up")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body, ct := buildMultipart(t, "y.txt", "y")
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/attachments", body)
		req.Header.Set("Content-Type", ct)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestService_UpdateThread_PublishesEvent verifies the event bus integration.
func TestService_UpdateThread_PublishesEvent(t *testing.T) {
	db, _ := setupDB(t)
	bus := eventbus.New()
	done := make(chan eventbus.Event, 1)
	events, _ := bus.Subscribe("thread.updated", 4)
	go func() {
		for e := range events {
			done <- e
		}
	}()

	svc := NewService(NewRepository(db), bus, nil)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "opener", CreateInput{Title: "Notify Ticket"})
	require.NoError(t, err)

	newBody := "updated content"
	_, err = svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{Body: &newBody}, nil)
	require.NoError(t, err)

	select {
	case evt := <-done:
		assert.Equal(t, "thread.updated", evt.Type)
		assert.Equal(t, "editor", evt.UserID)
		payload, ok := evt.Payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "global-support", payload["source"])
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected event was not published")
	}
	bus.Close()
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

func TestSetTicketNumberer(t *testing.T) {
	// SetTicketNumberer stores the numberer for use in CreateThread.
	// Verify it can be set and cleared without panicking.
	SetTicketNumberer(nil)
	// A nil numberer is safe — CreateThread skips numbering when nil.
	SetTicketNumberer(nil)
}

func TestService_CreateThread_LockedBoard(t *testing.T) {
	db, boardID := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	require.NoError(t, db.Model(&models.Board{}).Where("id = ?", boardID).Update("is_locked", true).Error)

	_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Cannot Create"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "board is locked")
}

func TestService_CreateThread_AssignsTicketNumberWhenNumbererConfigured(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	mockNumberer := &stubTicketNumberer{}
	SetTicketNumberer(mockNumberer)
	t.Cleanup(func() { SetTicketNumberer(nil) })

	_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{Title: "Needs Number"})
	require.NoError(t, err)
	assert.True(t, mockNumberer.called)
	assert.Equal(t, "_system", mockNumberer.orgID)
}

func TestService_UploadAttachment_NoUploadService(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	f := createTempMultipartFile(t, "missing-upload-service.txt", "x")
	_, err := svc.UploadAttachment(context.Background(), "global-support", "any", "u1", "a.txt", 1, f, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload service not available")
}

func TestService_UploadAttachment_FailsWhenSystemOrgMissing(t *testing.T) {
	db, _ := setupDB(t)
	upSvc := newUploadSvc(t, db)
	svc := NewService(NewRepository(db), nil, upSvc)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "user-up", CreateInput{Title: "Upload Ticket"})
	require.NoError(t, err)

	require.NoError(t, db.Where("slug = ?", "_system").Delete(&models.Org{}).Error)

	f := createTempMultipartFile(t, "system-org-missing.txt", "test")
	_, err = svc.UploadAttachment(ctx, "global-support", th.Slug, "user-up", "notes.txt", int64(len("test")), f, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving org for upload")
}

func TestService_UploadAttachment_NotFoundPaths(t *testing.T) {
	db, _ := setupDB(t)
	upSvc := newUploadSvc(t, db)
	svc := NewService(NewRepository(db), nil, upSvc)
	ctx := context.Background()
	file := createTempMultipartFile(t, "not-found-upload.txt", "content")

	uploaded, err := svc.UploadAttachment(ctx, "no-such-space", "x", "u1", "a.txt", 7, file, nil)
	require.NoError(t, err)
	assert.Nil(t, uploaded)

	file2 := createTempMultipartFile(t, "not-found-thread-upload.txt", "content")
	uploaded, err = svc.UploadAttachment(ctx, "global-support", "no-such-thread", "u1", "a.txt", 7, file2, nil)
	require.NoError(t, err)
	assert.Nil(t, uploaded)
}

func TestService_CreateThread_NumbererScopeBehavior(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	var systemOrg models.Org
	require.NoError(t, db.Where("slug = ?", "_system").First(&systemOrg).Error)
	forumSpace := &models.Space{
		OrgID:    systemOrg.ID,
		Name:     "Forum",
		Slug:     "global-forum",
		Type:     models.SpaceTypeCommunity,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(forumSpace).Error)
	forumBoard := &models.Board{
		SpaceID:  forumSpace.ID,
		Name:     "Forum Board",
		Slug:     "forum-board",
		Metadata: "{}",
	}
	require.NoError(t, db.Create(forumBoard).Error)

	mockNumberer := &stubTicketNumberer{}
	SetTicketNumberer(mockNumberer)
	t.Cleanup(func() { SetTicketNumberer(nil) })

	customerOrg := "org-abc"
	_, err := svc.CreateThread(ctx, "global-support", "user1", CreateInput{
		Title: "Org Scoped Support Ticket",
		OrgID: &customerOrg,
	})
	require.NoError(t, err)
	assert.True(t, mockNumberer.called)
	assert.Equal(t, customerOrg, mockNumberer.orgID)

	mockNumberer.called = false
	_, err = svc.CreateThread(ctx, "global-forum", "user1", CreateInput{
		Title: "Forum Topic",
	})
	require.NoError(t, err)
	assert.False(t, mockNumberer.called)
}

func TestRepository_ListThreads_InvalidCursor(t *testing.T) {
	db, _ := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	var board models.Board
	require.NoError(t, db.Where("slug = ?", "support-board").First(&board).Error)

	_, _, err := repo.ListThreads(ctx, board.ID, ListParams{
		Params: pagination.Params{Limit: 25, Cursor: "invalid-cursor"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor")
}

func TestRepository_EmptyLookupFastPaths(t *testing.T) {
	db, _ := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	shadows, err := repo.GetUserShadowsByIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, shadows)

	orgNames, err := repo.GetOrgNamesByIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, orgNames)
}

// --- Visibility scope enforcement tests ---

// TestVisibility_ScopeOwner verifies that a user without DEFT/org membership
// can only see tickets they authored or that are assigned to them.
func TestVisibility_ScopeOwner(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	// Create tickets by different authors.
	_, err := svc.CreateThread(ctx, "global-support", "solo-user", CreateInput{Title: "My Ticket"})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "other-user", CreateInput{Title: "Other Ticket"})
	require.NoError(t, err)

	// Create a ticket assigned to solo-user (via metadata for assigned_to generated column).
	assigned, err := svc.CreateThread(ctx, "global-support", "agent-user", CreateInput{Title: "Assigned Ticket"})
	require.NoError(t, err)
	// Set assigned_to via metadata update.
	require.NoError(t, db.Model(&models.Thread{}).Where("id = ?", assigned.ID).
		Update("metadata", `{"assigned_to":"solo-user"}`).Error)

	t.Run("list shows only own and assigned tickets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads", nil)
		req = withUser(req, "solo-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		// solo-user authored "My Ticket" and is assigned "Assigned Ticket".
		assert.Len(t, data, 2)
	})

	t.Run("get own ticket succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/my-ticket", nil)
		req = withUser(req, "solo-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get other's ticket returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/other-ticket", nil)
		req = withUser(req, "solo-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update other's ticket returns 404", func(t *testing.T) {
		body := `{"body":"tamper"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/other-ticket", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "solo-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unauthenticated list returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestVisibility_ScopeOrg verifies that an org member can see all tickets
// from fellow org members, but not tickets from other orgs.
func TestVisibility_ScopeOrg(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	// Create customer org and add two members.
	orgID := seedOrgMember(t, db, "Acme Corp", "acme", "org-user-a")
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: orgID, UserID: "org-user-b", Role: models.RoleContributor,
	}).Error)

	// Create a different org.
	orgID2 := seedOrgMember(t, db, "Rival Inc", "rival", "rival-user")

	// Create tickets scoped to each org.
	_, err := svc.CreateThread(ctx, "global-support", "org-user-a", CreateInput{Title: "Acme Ticket A", OrgID: &orgID})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "org-user-b", CreateInput{Title: "Acme Ticket B", OrgID: &orgID})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "rival-user", CreateInput{Title: "Rival Ticket", OrgID: &orgID2})
	require.NoError(t, err)
	// Ticket without org — should NOT be visible to org member.
	_, err = svc.CreateThread(ctx, "global-support", "no-org-user", CreateInput{Title: "No Org Ticket"})
	require.NoError(t, err)

	t.Run("org member sees own org tickets only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads", nil)
		req = withUser(req, "org-user-a")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 2, "org member should see both acme tickets")
	})

	t.Run("org member can get fellow member's ticket", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/acme-ticket-b", nil)
		req = withUser(req, "org-user-a")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("org member cannot get rival's ticket", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/rival-ticket", nil)
		req = withUser(req, "org-user-a")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("org member cannot get unscoped ticket", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/no-org-ticket", nil)
		req = withUser(req, "org-user-a")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestVisibility_ScopeAll verifies that DEFT members and platform admins see all tickets.
func TestVisibility_ScopeAll(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	// Seed DEFT member and platform admin.
	seedDeftMember(t, db, "deft-agent")
	require.NoError(t, db.Create(&models.PlatformAdmin{UserID: "plat-admin", IsActive: true}).Error)

	// Create tickets from various sources.
	orgA := seedOrgMember(t, db, "OrgA", "org-a", "org-a-user")
	_, err := svc.CreateThread(ctx, "global-support", "org-a-user", CreateInput{Title: "OrgA Ticket", OrgID: &orgA})
	require.NoError(t, err)
	_, err = svc.CreateThread(ctx, "global-support", "random-user", CreateInput{Title: "Random Ticket"})
	require.NoError(t, err)

	for _, userID := range []string{"deft-agent", "plat-admin"} {
		t.Run("user "+userID+" sees all tickets", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads", nil)
			req = withUser(req, userID)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			var resp map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			data := resp["data"].([]any)
			assert.Len(t, data, 2, "DEFT/admin should see all tickets")
		})

		t.Run("user "+userID+" can get any ticket", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/random-ticket", nil)
			req = withUser(req, userID)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestVisibility_ResolveVisibility verifies the service-level tier resolution.
func TestVisibility_ResolveVisibility(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	// Platform admin.
	require.NoError(t, db.Create(&models.PlatformAdmin{UserID: "admin-vis", IsActive: true}).Error)
	cv, err := svc.ResolveVisibility(ctx, "admin-vis")
	require.NoError(t, err)
	assert.Equal(t, ScopeAll, cv.Scope)

	// DEFT org member.
	seedDeftMember(t, db, "deft-vis")
	cv, err = svc.ResolveVisibility(ctx, "deft-vis")
	require.NoError(t, err)
	assert.Equal(t, ScopeAll, cv.Scope)

	// Customer org member.
	orgID := seedOrgMember(t, db, "CustOrg", "cust-org", "cust-vis")
	cv, err = svc.ResolveVisibility(ctx, "cust-vis")
	require.NoError(t, err)
	assert.Equal(t, ScopeOrg, cv.Scope)
	assert.Contains(t, cv.OrgIDs, orgID)

	// Solo user — no memberships.
	cv, err = svc.ResolveVisibility(ctx, "solo-vis")
	require.NoError(t, err)
	assert.Equal(t, ScopeOwner, cv.Scope)
	assert.Equal(t, "solo-vis", cv.UserID)
}

// TestVisibility_MutationEndpointsGated verifies that UpdateThread and attachment
// endpoints respect visibility scoping.
func TestVisibility_MutationEndpointsGated(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	orgID := seedOrgMember(t, db, "GatedOrg", "gated-org", "gated-user")
	_, err := svc.CreateThread(ctx, "global-support", "gated-user", CreateInput{Title: "Gated Ticket", OrgID: &orgID})
	require.NoError(t, err)

	// outsider-user has no org membership — ScopeOwner, not the author.
	t.Run("outsider cannot update", func(t *testing.T) {
		body := `{"body":"hacked"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/gated-ticket", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "outsider-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("outsider cannot list attachments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/gated-ticket/attachments", nil)
		req = withUser(req, "outsider-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Same-org member CAN access.
	t.Run("same org member can update", func(t *testing.T) {
		body := `{"body":"updated by colleague"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/gated-ticket", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "gated-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- Ticket assignment tests ---

// TestService_UpdateThread_AssignDeftMember verifies assigning a DEFT member.
func TestService_UpdateThread_AssignDeftMember(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	seedDeftMember(t, db, "agent-1")

	th, err := svc.CreateThread(ctx, "global-support", "customer", CreateInput{Title: "Assign Me"})
	require.NoError(t, err)

	t.Run("assign valid DEFT member", func(t *testing.T) {
		assignee := "agent-1"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &assignee}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "agent-1", rich.AssignedTo)
	})

	t.Run("assign non-DEFT member fails", func(t *testing.T) {
		assignee := "random-outsider"
		_, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &assignee}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "assignee must be a DEFT org member")
	})

	t.Run("unassign with empty string", func(t *testing.T) {
		empty := ""
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &empty}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "", rich.AssignedTo)
	})
}

// TestHandler_UpdateThread_AssignedTo covers the handler-level assignment validation.
func TestHandler_UpdateThread_AssignedTo(t *testing.T) {
	db, _ := setupDB(t)
	h := NewHandler(newSvc(db))
	r := globalSpaceRouter(h)
	ctx := context.Background()
	svc := newSvc(db)

	seedDeftMember(t, db, "deft-assignee")

	th, err := svc.CreateThread(ctx, "global-support", "user-assign", CreateInput{Title: "Handler Assign"})
	require.NoError(t, err)

	t.Run("assign valid member via handler", func(t *testing.T) {
		body := `{"assigned_to":"deft-assignee"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/"+th.Slug, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-assign")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "deft-assignee", resp["assigned_to"])
	})

	t.Run("assign non-DEFT member returns 400", func(t *testing.T) {
		body := `{"assigned_to":"not-deft"}`
		req := httptest.NewRequest(http.MethodPatch, "/global-spaces/global-support/threads/"+th.Slug, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "user-assign")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestService_UpdateThread_AssignAutoTransitionsStatus verifies open→assigned and assigned→open.
func TestService_UpdateThread_AssignAutoTransitionsStatus(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	seedDeftMember(t, db, "agent-status")

	th, err := svc.CreateThread(ctx, "global-support", "customer", CreateInput{Title: "Status Transition"})
	require.NoError(t, err)

	t.Run("assign transitions open to assigned", func(t *testing.T) {
		assignee := "agent-status"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &assignee}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "assigned", rich.Status)
		assert.Equal(t, "agent-status", rich.AssignedTo)
	})

	t.Run("unassign transitions assigned back to open", func(t *testing.T) {
		empty := ""
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &empty}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "open", rich.Status)
		assert.Equal(t, "", rich.AssignedTo)
	})

	t.Run("assign does not override pending status", func(t *testing.T) {
		// First set status to pending.
		pending := "pending"
		_, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{Status: &pending}, nil)
		require.NoError(t, err)

		// Now assign — should NOT override pending to assigned.
		assignee := "agent-status"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &assignee}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "pending", rich.Status, "assigning should not override non-open status")
		assert.Equal(t, "agent-status", rich.AssignedTo)
	})

	t.Run("unassign does not revert resolved status", func(t *testing.T) {
		// Set status to resolved.
		resolved := "resolved"
		_, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{Status: &resolved}, nil)
		require.NoError(t, err)

		// Now unassign — should NOT revert resolved to open.
		empty := ""
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &empty}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "resolved", rich.Status, "unassigning should not revert non-assigned status")
	})

	t.Run("explicit status in same request takes precedence", func(t *testing.T) {
		// Reset to open first.
		open := "open"
		_, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{Status: &open}, nil)
		require.NoError(t, err)

		// Assign AND set status to pending in the same request.
		assignee := "agent-status"
		pending := "pending"
		rich, err := svc.UpdateThread(ctx, "global-support", th.Slug, "editor", UpdateInput{AssignedTo: &assignee, Status: &pending}, nil)
		require.NoError(t, err)
		require.NotNil(t, rich)
		assert.Equal(t, "pending", rich.Status, "explicit status should override auto-transition")
	})
}

// TestService_UpdateThread_AssignPublishesEvent verifies assigned_to is in the event payload.
func TestService_UpdateThread_AssignPublishesEvent(t *testing.T) {
	db, _ := setupDB(t)
	bus := eventbus.New()
	done := make(chan eventbus.Event, 1)
	events, _ := bus.Subscribe("thread.updated", 4)
	go func() {
		for e := range events {
			done <- e
		}
	}()

	svc := NewService(NewRepository(db), bus, nil)
	ctx := context.Background()

	seedDeftMember(t, db, "assign-target")

	th, err := svc.CreateThread(ctx, "global-support", "opener", CreateInput{Title: "Event Assign Ticket"})
	require.NoError(t, err)

	assignee := "assign-target"
	_, err = svc.UpdateThread(ctx, "global-support", th.Slug, "assigner", UpdateInput{AssignedTo: &assignee}, nil)
	require.NoError(t, err)

	select {
	case evt := <-done:
		assert.Equal(t, "thread.updated", evt.Type)
		payload, ok := evt.Payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "assign-target", payload["assigned_to"])
		assert.Equal(t, "global-support", payload["source"])
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected assignment event was not published")
	}
	bus.Close()
}
