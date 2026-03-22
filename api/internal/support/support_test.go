package support

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// --- Repository: ListEntries own-draft visibility ---

func TestRepository_ListEntries_OwnDraftVisible(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	ticket := createTicket(t, db, board.ID, "customer1")
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	ctx := context.Background()

	// Create a draft authored by customer1.
	draft, err := svc.CreateEntry(ctx, "test-ticket", "customer1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer, // use customer to create as non-DEFT
		Body: "<p>initial</p>",
	})
	require.NoError(t, err)
	// Manually flip to draft type for this test (bypass service permission).
	draft.Type = models.MessageTypeDraft
	draft.IsPublished = false
	draft.IsImmutable = false
	require.NoError(t, db.Save(draft).Error)
	_ = ticket

	// Customer1 sees own draft.
	entries, err := repo.ListEntries(ctx, ticket.ID, false, "customer1")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, models.MessageTypeDraft, entries[0].Type)

	// Different customer does NOT see the draft.
	visible, err := repo.ListEntries(ctx, ticket.ID, false, "customer2")
	require.NoError(t, err)
	assert.Empty(t, visible)
}

func TestService_IsDeftMember_PlatformAdmin(t *testing.T) {
	db := setupTestDB(t)
	// Create a platform admin.
	admin := &models.PlatformAdmin{UserID: "admin1", IsActive: true}
	require.NoError(t, db.Create(admin).Error)

	svc := NewService(NewRepository(db), nil)
	isDeft, err := svc.IsDeftMember(context.Background(), "admin1")
	require.NoError(t, err)
	assert.True(t, isDeft)
}

func TestService_IsDeftMember_NotMember(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db), nil)
	isDeft, err := svc.IsDeftMember(context.Background(), "random-user")
	require.NoError(t, err)
	assert.False(t, isDeft)
}

// --- Unclaimed ticket + claim tests ---

func TestService_ListUnclaimedTickets(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	ctx := context.Background()

	// Create user shadow for the claimant.
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "claimer",
		Email:       "claimer@example.com",
		DisplayName: "Claimer",
		LastSeenAt:  time.Now(),
		SyncedAt:    time.Now(),
	}).Error)

	// Create an orphaned ticket (contact_email set, author is deft member).
	orphan := &models.Thread{
		BoardID:      board.ID,
		Title:        "Orphaned Ticket",
		Slug:         "orphaned-ticket",
		Metadata:     "{}",
		AuthorID:     "deft-member",
		ContactEmail: "claimer@example.com",
		ThreadType:   models.ThreadTypeSupport,
		Visibility:   models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(orphan).Error)

	// Non-orphaned ticket (same email, but author IS the claimer).
	ownerTicket := &models.Thread{
		BoardID:      board.ID,
		Title:        "Own Ticket",
		Slug:         "own-ticket",
		Metadata:     "{}",
		AuthorID:     "claimer",
		ContactEmail: "claimer@example.com",
		ThreadType:   models.ThreadTypeSupport,
		Visibility:   models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(ownerTicket).Error)

	t.Run("returns only orphaned tickets", func(t *testing.T) {
		tickets, err := svc.ListUnclaimedTickets(ctx, "claimer")
		require.NoError(t, err)
		require.Len(t, tickets, 1)
		assert.Equal(t, "Orphaned Ticket", tickets[0].Title)
	})

	t.Run("user without shadow returns nil", func(t *testing.T) {
		tickets, err := svc.ListUnclaimedTickets(ctx, "no-shadow")
		require.NoError(t, err)
		assert.Nil(t, tickets)
	})
}

func TestService_ClaimTickets(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "claimer2",
		Email:       "claimer2@example.com",
		DisplayName: "Claimer Two",
		LastSeenAt:  time.Now(),
		SyncedAt:    time.Now(),
	}).Error)

	orphan := &models.Thread{
		BoardID:      board.ID,
		Title:        "Claimable",
		Slug:         "claimable",
		Metadata:     "{}",
		AuthorID:     "someone-else",
		ContactEmail: "claimer2@example.com",
		ThreadType:   models.ThreadTypeSupport,
		Visibility:   models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(orphan).Error)

	mismatch := &models.Thread{
		BoardID:      board.ID,
		Title:        "Wrong Email",
		Slug:         "wrong-email",
		Metadata:     "{}",
		AuthorID:     "someone-else",
		ContactEmail: "other@example.com",
		ThreadType:   models.ThreadTypeSupport,
		Visibility:   models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(mismatch).Error)

	t.Run("claims matching ticket and nullifies contact_email", func(t *testing.T) {
		claimed, err := svc.ClaimTickets(ctx, "claimer2", []string{orphan.ID})
		require.NoError(t, err)
		assert.Equal(t, 1, claimed)

		// Verify author updated and contact_email cleared.
		var updated models.Thread
		require.NoError(t, db.First(&updated, "id = ?", orphan.ID).Error)
		assert.Equal(t, "claimer2", updated.AuthorID)
		assert.Equal(t, "", updated.ContactEmail, "contact_email should be nullified")

		// Verify system_event message posted.
		var msgs []models.Message
		require.NoError(t, db.Where("thread_id = ? AND type = ?", orphan.ID, models.MessageTypeSystemEvent).Find(&msgs).Error)
		require.Len(t, msgs, 1)
		assert.Contains(t, msgs[0].Body, "Claimer Two")
	})

	t.Run("skips mismatched email ticket", func(t *testing.T) {
		claimed, err := svc.ClaimTickets(ctx, "claimer2", []string{mismatch.ID})
		require.NoError(t, err)
		assert.Equal(t, 0, claimed)
	})

	t.Run("user without shadow returns error", func(t *testing.T) {
		_, err := svc.ClaimTickets(ctx, "no-shadow", []string{orphan.ID})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmailMismatch)
	})
}

// TestHandler_UnclaimedTickets covers the GET /v1/support/unclaimed-tickets endpoint.
func TestHandler_UnclaimedTickets(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	h := NewHandler(svc)

	router := chi.NewRouter()
	router.Get("/support/unclaimed-tickets", h.ListUnclaimedTickets)
	router.Post("/support/claim-tickets", h.ClaimTickets)

	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "api-claimer",
		Email:       "api-claimer@example.com",
		DisplayName: "API Claimer",
		LastSeenAt:  time.Now(),
		SyncedAt:    time.Now(),
	}).Error)

	orphan := &models.Thread{
		BoardID:      board.ID,
		Title:        "API Orphan",
		Slug:         "api-orphan",
		Metadata:     "{}",
		AuthorID:     "deft-agent",
		ContactEmail: "api-claimer@example.com",
		ThreadType:   models.ThreadTypeSupport,
		Visibility:   models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(orphan).Error)

	t.Run("GET unclaimed returns orphaned tickets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/support/unclaimed-tickets", nil)
		req = withUser(req, "api-claimer")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 1)
	})

	t.Run("POST claim transfers ownership", func(t *testing.T) {
		body := `{"ticket_ids":["` + orphan.ID + `"]}`
		req := httptest.NewRequest(http.MethodPost, "/support/claim-tickets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "api-claimer")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["claimed"])
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/support/unclaimed-tickets", nil)
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST claim with empty body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/support/claim-tickets", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "api-claimer")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST claim with empty ticket_ids returns 400", func(t *testing.T) {
		body := `{"ticket_ids":[]}`
		req := httptest.NewRequest(http.MethodPost, "/support/claim-tickets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "api-claimer")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST claim unauthenticated returns 401", func(t *testing.T) {
		body := `{"ticket_ids":["abc"]}`
		req := httptest.NewRequest(http.MethodPost, "/support/claim-tickets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRepository_FindUnclaimedTickets_EmptyEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	tickets, err := repo.FindUnclaimedTickets(context.Background(), "", "user")
	require.NoError(t, err)
	assert.Nil(t, tickets)
}

// --- DEFT members list ---

func TestRepository_ListDeftMembers(t *testing.T) {
	db := setupTestDB(t)
	org, _ := createHierarchy(t, db)
	ctx := context.Background()

	// Add two org members.
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: "deft-user-a", Role: models.RoleContributor,
	}).Error)
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: "deft-user-b", Role: models.RoleAdmin,
	}).Error)

	// Add user shadows for display info.
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "deft-user-a", Email: "a@deft.co", DisplayName: "Alice Agent",
	}).Error)
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "deft-user-b", Email: "b@deft.co", DisplayName: "Bob Agent",
	}).Error)

	repo := NewRepository(db)
	members, err := repo.ListDeftMembers(ctx)
	require.NoError(t, err)
	require.Len(t, members, 2)

	// Ordered by display_name ASC.
	assert.Equal(t, "Alice Agent", members[0].DisplayName)
	assert.Equal(t, "a@deft.co", members[0].Email)
	assert.Equal(t, "deft-user-a", members[0].UserID)
	assert.Equal(t, "Bob Agent", members[1].DisplayName)
}

func TestRepository_ListDeftMembers_EmptyWithoutDeftOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	members, err := repo.ListDeftMembers(ctx)
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestHandler_ListDeftMembers(t *testing.T) {
	db := setupTestDB(t)
	org, _ := createHierarchy(t, db)

	// Add a DEFT member.
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: "handler-deft", Role: models.RoleContributor,
	}).Error)
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "handler-deft", Email: "handler@deft.co", DisplayName: "Handler Agent",
	}).Error)

	repo := NewRepository(db)
	svc := NewService(repo, nil)
	h := NewHandler(svc)

	router := chi.NewRouter()
	router.Get("/support/deft-members", h.ListDeftMembers)

	t.Run("DEFT member gets list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/support/deft-members", nil)
		req = withUser(req, "handler-deft")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 1)
		first := data[0].(map[string]any)
		assert.Equal(t, "handler-deft", first["user_id"])
		assert.Equal(t, "Handler Agent", first["display_name"])
	})

	t.Run("non-DEFT member gets 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/support/deft-members", nil)
		req = withUser(req, "random-customer")
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/support/deft-members", nil)
		w := routeRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestService_IsDeftMember_DeftOrgMember(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "DEFT", Slug: "deft", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "deft-user", Role: models.RoleContributor}
	require.NoError(t, db.Create(membership).Error)

	svc := NewService(NewRepository(db), nil)
	isDeft, err := svc.IsDeftMember(context.Background(), "deft-user")
	require.NoError(t, err)
	assert.True(t, isDeft)
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

	// Non-DEFT sees only customer + agent_reply + system_event (not their own draft since ownerID = unknown).
	visible, err := svc.ListEntries(ctx, "test-ticket", false, "other-user")
	require.NoError(t, err)
	for _, m := range visible {
		assert.True(t, m.Type.IsVisibleToCustomer(), "non-DEFT should not see %s", m.Type)
		assert.False(t, m.IsDeftOnly)
	}
	assert.Len(t, visible, 3)

	// DEFT sees all 5.
	all, err := svc.ListEntries(ctx, "test-ticket", true, "")
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

	visible, err := svc.ListEntries(ctx, "test-ticket", false, "other-user")
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

func TestHandler_ListEntries_WithTicketAndEntries(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, svc := buildRouter(t, db)
	ctx := context.Background()

	// Create an entry so the list has something.
	_, err := svc.CreateEntry(ctx, "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>hello</p>",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/support/tickets/test-ticket/entries", nil)
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	data, ok := body["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 1)
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

func TestHandler_SetDeftVisibility_NonDeftForbidden(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	svc := NewService(NewRepository(db), nil)
	msg, err := svc.CreateEntry(context.Background(), "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer,
		Body: "<p>hi</p>",
	})
	require.NoError(t, err)

	router, _ := buildRouter(t, db)
	body, _ := json.Marshal(map[string]any{"is_deft_only": true})
	req := httptest.NewRequest(http.MethodPatch,
		"/v1/support/tickets/test-ticket/entries/"+msg.ID+"/deft-visibility",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "external-user")
	w := routeRequest(router, req)
	// non-DEFT user is forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_UpdateEntry_NotFound(t *testing.T) {
	db := setupTestDB(t)
	router, _ := buildRouter(t, db)
	body, _ := json.Marshal(map[string]any{"body": "<p>Updated</p>"})
	req := httptest.NewRequest(http.MethodPatch,
		"/v1/support/tickets/test-ticket/entries/nonexistent",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_PublishEntry_DraftSuccess(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")

	// Create DEFT org member so IsDeftMember returns true.
	var org models.Org
	require.NoError(t, db.Where("slug = ?", "deft").First(&org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "deft-agent", Role: models.RoleContributor}
	require.NoError(t, db.Create(membership).Error)

	svc := NewService(NewRepository(db), nil)
	draft, err := svc.CreateEntry(context.Background(), "test-ticket", "deft-agent", true, CreateEntryInput{
		Type: models.MessageTypeDraft,
		Body: "<p>Draft reply</p>",
	})
	require.NoError(t, err)

	router, _ := buildRouter(t, db)
	req := httptest.NewRequest(http.MethodPost,
		"/v1/support/tickets/test-ticket/entries/"+draft.ID+"/publish",
		nil)
	req = withUser(req, "deft-agent")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result models.Message
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, models.MessageTypeAgentReply, result.Type)
	assert.True(t, result.IsPublished)
}

// createTicketWithSlug creates a support thread with a custom slug.
func createTicketWithSlug(t *testing.T, db *gorm.DB, boardID, authorID, slug string) *models.Thread {
	t.Helper()
	ticket := &models.Thread{
		BoardID:    boardID,
		Title:      "Ticket " + slug,
		Slug:       slug,
		Metadata:   "{}",
		AuthorID:   authorID,
		ThreadType: models.ThreadTypeSupport,
		Visibility: models.ThreadVisibilityOrgOnly,
	}
	require.NoError(t, db.Create(ticket).Error)
	return ticket
}

func TestRepository_AssignTicketNumber(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	ticket := createTicketWithSlug(t, db, board.ID, "u1", "ticket-a")
	repo := NewRepository(db)

	err := repo.AssignTicketNumber(context.Background(), ticket, "org-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), ticket.TicketNumber)

	// Second assignment for same org gets next number.
	ticket2 := createTicketWithSlug(t, db, board.ID, "u2", "ticket-b")
	err = repo.AssignTicketNumber(context.Background(), ticket2, "org-1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), ticket2.TicketNumber)
}

func TestHandler_UpdateEntry_Success(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")
	router, svc := buildRouter(t, db)
	ctx := context.Background()

	// Create a draft (mutable).
	draft, err := svc.CreateEntry(ctx, "test-ticket", "agent1", true, CreateEntryInput{
		Type: models.MessageTypeDraft,
		Body: "<p>Original</p>",
	})
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{"body": "<p>Updated draft</p>"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/support/tickets/test-ticket/entries/"+draft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "agent1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result models.Message
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, "<p>Updated draft</p>", result.Body)
}

func TestHandler_PublishEntry_NotFound(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "agent1")
	// Add deft org membership so IsDeftMember returns true.
	var org models.Org
	require.NoError(t, db.Where("slug = ?", "deft").First(&org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "deft-a2", Role: models.RoleContributor}
	require.NoError(t, db.Create(membership).Error)

	router, _ := buildRouter(t, db)
	req := httptest.NewRequest(http.MethodPost, "/v1/support/tickets/test-ticket/entries/nonexistent/publish", nil)
	req = withUser(req, "deft-a2")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_PublishEntry_NotDraft(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	var org models.Org
	require.NoError(t, db.Where("slug = ?", "deft").First(&org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "deft-a3", Role: models.RoleContributor}
	require.NoError(t, db.Create(membership).Error)

	svc := NewService(NewRepository(db), nil)
	msg, err := svc.CreateEntry(context.Background(), "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer, Body: "<p>x</p>",
	})
	require.NoError(t, err)

	router, _ := buildRouter(t, db)
	req := httptest.NewRequest(http.MethodPost, "/v1/support/tickets/test-ticket/entries/"+msg.ID+"/publish", nil)
	req = withUser(req, "deft-a3")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetDeftVisibility_DeftMemberSuccess(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	// Add deft org membership.
	var org models.Org
	require.NoError(t, db.Where("slug = ?", "deft").First(&org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "deft-vis", Role: models.RoleContributor}
	require.NoError(t, db.Create(membership).Error)

	svc := NewService(NewRepository(db), nil)
	msg, err := svc.CreateEntry(context.Background(), "test-ticket", "u1", false, CreateEntryInput{
		Type: models.MessageTypeCustomer, Body: "<p>hi</p>",
	})
	require.NoError(t, err)

	router, _ := buildRouter(t, db)
	body, _ := json.Marshal(map[string]any{"is_deft_only": true})
	req := httptest.NewRequest(http.MethodPatch,
		"/v1/support/tickets/test-ticket/entries/"+msg.ID+"/deft-visibility",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "deft-vis")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result models.Message
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.True(t, result.IsDeftOnly)
}

func TestHandler_SetDeftVisibility_NotFound(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	var org models.Org
	require.NoError(t, db.Where("slug = ?", "deft").First(&org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "deft-vis2", Role: models.RoleContributor}
	require.NoError(t, db.Create(membership).Error)

	router, _ := buildRouter(t, db)
	body, _ := json.Marshal(map[string]any{"is_deft_only": false})
	req := httptest.NewRequest(http.MethodPatch,
		"/v1/support/tickets/test-ticket/entries/nonexistent/deft-visibility",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "deft-vis2")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CreateEntry_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, _ := buildRouter(t, db)

	body, _ := json.Marshal(map[string]any{"type": "note", "body": "<p>hi</p>"})
	req := httptest.NewRequest(http.MethodPost, "/v1/support/tickets/test-ticket/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetNotificationPref_Full(t *testing.T) {
	db := setupTestDB(t)
	_, board := createHierarchy(t, db)
	_ = createTicket(t, db, board.ID, "u1")
	router, _ := buildRouter(t, db)

	body, _ := json.Marshal(map[string]any{"notification_detail_level": "full"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/support/tickets/test-ticket/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "u1")
	w := routeRequest(router, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
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
