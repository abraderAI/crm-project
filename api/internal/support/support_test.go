package support

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
)

// --- Test helpers ---

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

// createHierarchy inserts a DEFT org → support space → board and returns the board.
func createHierarchy(t *testing.T, db *gorm.DB) (*models.Org, *models.Board) {
	t.Helper()
	org := &models.Org{Name: "DEFT", Slug: "deft", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	space := &models.Space{
		OrgID:    org.ID,
		Name:     "global-support",
		Slug:     "global-support",
		Type:     models.SpaceTypeSupport,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(space).Error)

	board := &models.Board{
		SpaceID:  space.ID,
		Name:     "Support",
		Slug:     "support",
		Metadata: "{}",
	}
	require.NoError(t, db.Create(board).Error)
	return org, board
}

// createTicket creates a support thread and returns it.
func createTicket(t *testing.T, db *gorm.DB, boardID, authorID string) *models.Thread {
	t.Helper()
	ticket := &models.Thread{
		BoardID:    boardID,
		Title:      "Test ticket",
		Slug:       "test-ticket",
		Metadata:   "{}",
		AuthorID:   authorID,
		ThreadType: models.ThreadTypeSupport,
		Visibility: models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(ticket).Error)
	return ticket
}

// withUser returns an http.Request with a UserContext for the given userID.
func withUser(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID, AuthMethod: auth.AuthMethodJWT})
	return r.WithContext(ctx)
}

// routeRequest routes an HTTP request through a chi router and returns the response.
func routeRequest(router http.Handler, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

// --- MessageType model tests ---

func TestMessageType_IsSupportType(t *testing.T) {
	tests := []struct {
		name        string
		msgType     models.MessageType
		wantSupport bool
	}{
		{"customer", models.MessageTypeCustomer, true},
		{"agent_reply", models.MessageTypeAgentReply, true},
		{"draft", models.MessageTypeDraft, true},
		{"context", models.MessageTypeContext, true},
		{"system_event", models.MessageTypeSystemEvent, true},
		{"comment", models.MessageTypeComment, false},
		{"note", models.MessageTypeNote, false},
		{"email", models.MessageTypeEmail, false},
		{"system", models.MessageTypeSystem, false},
		{"call_log", models.MessageTypeCallLog, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantSupport, tt.msgType.IsSupportType())
		})
	}
}

func TestMessageType_IsVisibleToCustomer(t *testing.T) {
	tests := []struct {
		msgType models.MessageType
		want    bool
	}{
		{models.MessageTypeCustomer, true},
		{models.MessageTypeAgentReply, true},
		{models.MessageTypeSystemEvent, true},
		{models.MessageTypeDraft, false},
		{models.MessageTypeContext, false},
		{models.MessageTypeComment, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.msgType), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.msgType.IsVisibleToCustomer())
		})
	}
}

func TestMessageType_IsValid(t *testing.T) {
	assert.True(t, models.MessageTypeCustomer.IsValid())
	assert.True(t, models.MessageTypeDraft.IsValid())
	assert.True(t, models.MessageTypeContext.IsValid())
	assert.False(t, models.MessageType("unknown").IsValid())
}

// --- Repository: ticket number tests ---

func TestRepository_NextTicketNumber_Sequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	n1, err := repo.NextTicketNumber(ctx, "org-1", "support")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n1)

	n2, err := repo.NextTicketNumber(ctx, "org-1", "support")
	require.NoError(t, err)
	assert.Equal(t, int64(2), n2)

	// Different org starts its own sequence.
	n3, err := repo.NextTicketNumber(ctx, "org-2", "support")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n3)
}

// --- Service: CreateEntry tests ---

func TestService_CreateEntry_CustomerEntry(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	ticket := createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	// Find by slug to get it working through the service.
	_ = ticket

	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>Customer message</p>",
	})
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, models.MessageTypeCustomer, msg.Type)
	assert.True(t, msg.IsPublished)
	assert.True(t, msg.IsImmutable)
	assert.NotNil(t, msg.PublishedAt)
}

func TestService_CreateEntry_NonDeftCannotCreateAgentReply(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)

	_, err := svc.CreateEntry(context.Background(), "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeAgentReply,
		Body: "<p>Reply</p>",
	})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestService_CreateEntry_DraftIsMutable(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u2")
	svc := NewService(NewRepository(db), nil)

	msg, err := svc.CreateEntry(context.Background(), "test-ticket", "u2", true, CreateEntryInput{
		Type: models.MessageTypeDraft,
		Body: "<p>Draft</p>",
	})
	require.NoError(t, err)
	assert.Equal(t, models.MessageTypeDraft, msg.Type)
	assert.False(t, msg.IsPublished)
	assert.False(t, msg.IsImmutable)
}

func TestService_CreateEntry_ContextForcedDeftOnly(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")
	svc := NewService(NewRepository(db), nil)

	msg, err := svc.CreateEntry(context.Background(), "test-ticket", "agent1", true, CreateEntryInput{
		Type: models.MessageTypeContext,
		Body: "<p>Internal note</p>",
	})
	require.NoError(t, err)
	assert.True(t, msg.IsDeftOnly, "context entries must be deft-only")
	assert.True(t, msg.IsImmutable)
}

func TestService_CreateEntry_EmptyBodyError(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)

	_, err := svc.CreateEntry(context.Background(), "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "",
	})
	assert.Error(t, err)
}

func TestService_CreateEntry_TicketNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db), nil)

	msg, err := svc.CreateEntry(context.Background(), "nonexistent", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "hi",
	})
	require.NoError(t, err)
	assert.Nil(t, msg)
}

// --- Service: UpdateEntryBody tests ---

func TestService_UpdateEntryBody_DraftUpdated(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	ticket := createTicket(t, db, board.ID, "agent1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	msg, err := svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{
		Type: models.MessageTypeDraft,
		Body: "<p>Initial</p>",
	})
	require.NoError(t, err)
	_ = ticket

	updated, err := svc.UpdateEntryBody(ctx, msg.ID, "<p>Updated</p>")
	require.NoError(t, err)
	assert.Equal(t, "<p>Updated</p>", updated.Body)
}

func TestService_UpdateEntryBody_ImmutableRejected(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>Customer</p>",
	})
	require.NoError(t, err)

	_, err = svc.UpdateEntryBody(ctx, msg.ID, "<p>Tamper</p>")
	assert.ErrorIs(t, err, ErrImmutable)
}

// --- Service: PublishDraft tests ---

func TestService_PublishDraft_Success(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	draft, err := svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{
		Type: models.MessageTypeDraft,
		Body: "<p>Draft reply</p>",
	})
	require.NoError(t, err)

	published, err := svc.PublishDraft(ctx, draft.ID, "agent1")
	require.NoError(t, err)
	assert.Equal(t, models.MessageTypeAgentReply, published.Type)
	assert.True(t, published.IsPublished)
	assert.True(t, published.IsImmutable)
	assert.NotNil(t, published.PublishedAt)
}

func TestService_PublishDraft_NotDraftError(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>Customer</p>",
	})
	require.NoError(t, err)

	_, err = svc.PublishDraft(ctx, msg.ID, "agent1")
	assert.ErrorIs(t, err, ErrNotDraft)
}

// --- Service: SetDeftVisibility tests ---

func TestService_SetDeftVisibility_DeftMemberCanToggle(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>Hi</p>",
	})
	require.NoError(t, err)
	assert.False(t, msg.IsDeftOnly)

	toggled, err := svc.SetDeftVisibility(ctx, msg.ID, true, true /* isDeft */)
	require.NoError(t, err)
	assert.True(t, toggled.IsDeftOnly)
}

func TestService_SetDeftVisibility_NonDeftForbidden(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>Hi</p>",
	})
	require.NoError(t, err)

	_, err = svc.SetDeftVisibility(ctx, msg.ID, true, false /* not deft */)
	assert.ErrorIs(t, err, ErrForbidden)
}

// --- Service: ListEntries visibility tests ---

func TestService_ListEntries_NonDeftFiltered(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	// Create one of each support entry type.
	_, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{Type: models.MessageTypeCustomer, Body: "<p>c</p>"})
	require.NoError(t, err)
	_, err = svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{Type: models.MessageTypeAgentReply, Body: "<p>r</p>"})
	require.NoError(t, err)
	_, err = svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{Type: models.MessageTypeDraft, Body: "<p>d</p>"})
	require.NoError(t, err)
	_, err = svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{Type: models.MessageTypeContext, Body: "<p>cx</p>"})
	require.NoError(t, err)
	_, err = svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{Type: models.MessageTypeSystemEvent, Body: "<p>e</p>"})
	require.NoError(t, err)

	// Non-DEFT sees only customer + agent_reply + system_event.
	visible, err := svc.ListEntries(ctx, "test-ticket", false)
	require.NoError(t, err)
	for _, m := range visible {
		assert.True(t, m.Type.IsVisibleToCustomer(), "non-DEFT should not see %s", m.Type)
		assert.False(t, m.IsDeftOnly)
	}
	assert.Len(t, visible, 3)

	// DEFT sees all 5.
	all, err := svc.ListEntries(ctx, "test-ticket", true)
	require.NoError(t, err)
	assert.Len(t, all, 5)
}

func TestService_ListEntries_DeftOnlyHiddenFromCustomer(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	// Create a customer entry and flag it DEFT-only.
	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>hi</p>",
	})
	require.NoError(t, err)
	_, err = svc.SetDeftVisibility(ctx, msg.ID, true, true)
	require.NoError(t, err)

	visible, err := svc.ListEntries(ctx, "test-ticket", false)
	require.NoError(t, err)
	assert.Empty(t, visible, "deft-only entry must not be visible to customer")
}

// --- Service: notification detail level ---

func TestService_SetNotificationDetailLevel_Valid(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()

	err := svc.SetNotificationDetailLevel(ctx, "test-ticket", "privacy")
	require.NoError(t, err)

	// Verify the metadata was updated.
	var t2 models.Thread
	require.NoError(t, db.First(&t2, "slug = ?", "test-ticket").Error)
	assert.Contains(t, t2.Metadata, "privacy")
}

func TestService_SetNotificationDetailLevel_Invalid(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db), nil)
	err := svc.SetNotificationDetailLevel(context.Background(), "any", "verbose")
	assert.Error(t, err)
}

// --- Handler HTTP tests ---

func buildRouter(t *testing.T, db *gorm.DB) (http.Handler, *Service) {
	t.Helper()
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Route("/v1/support/tickets/{slug}", func(sr chi.Router) {
		sr.Get("/entries", h.ListEntries)
		sr.Post("/entries", h.CreateEntry)
		sr.Patch("/entries/{id}", h.UpdateEntry)
		sr.Post("/entries/{id}/publish", h.PublishEntry)
		sr.Patch("/entries/{id}/deft-visibility", h.SetDeftVisibility)
		sr.Patch("/notifications", h.SetNotificationPref)
	})
	return r, svc
}

func TestHandler_ListEntries_Unauthenticated(t *testing.T) {
	db := setupTestDB(t)
	router, _ := buildRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/support/tickets/test-ticket/entries", nil)
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ListEntries_TicketNotFound(t *testing.T) {
	db := setupTestDB(t)
	// Create DEFT org so IsDeftMember check works.
	org := &models.Org{Name: "DEFT", Slug: "deft", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	router, _ := buildRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/support/tickets/nonexistent/entries", nil)
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CreateEntry_CustomerSuccess(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, _ := buildRouter(t, db)

	body, _ := json.Marshal(map[string]any{
		"type": "customer",
		"body": "<p>Help me</p>",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/support/tickets/test-ticket/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.Message
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, models.MessageTypeCustomer, result.Type)
	assert.True(t, result.IsPublished)
}

func TestHandler_CreateEntry_ForbiddenTypeForCustomer(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, _ := buildRouter(t, db)

	body, _ := json.Marshal(map[string]any{
		"type": "draft",
		"body": "<p>Draft</p>",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/support/tickets/test-ticket/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_UpdateEntry_ImmutableReturns403(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, svc := buildRouter(t, db)
	ctx := context.Background()

	// Create an immutable customer entry.
	msg, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>Original</p>",
	})
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{"body": "<p>Tamper</p>"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/support/tickets/test-ticket/entries/"+msg.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_PublishEntry_NonDeftForbidden(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")

	// Create draft via service directly.
	svc := NewService(NewRepository(db), nil)
	draft, err := svc.CreateEntry(context.Background(), "test-ticket", "agent1", true, CreateEntryInput{
		Type: models.MessageTypeDraft,
		Body: "<p>Draft</p>",
	})
	require.NoError(t, err)

	router, _ := buildRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/v1/support/tickets/test-ticket/entries/"+draft.ID+"/publish", nil)
	req = withUser(req, "external-user")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_SetNotificationPref_InvalidLevel(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, _ := buildRouter(t, db)

	body, _ := json.Marshal(map[string]any{"notification_detail_level": "all"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/support/tickets/test-ticket/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetNotificationPref_ValidPrivacy(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, _ := buildRouter(t, db)

	body, _ := json.Marshal(map[string]any{"notification_detail_level": "privacy"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/support/tickets/test-ticket/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
