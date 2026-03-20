package channel

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
)

// --- Test DB setup ---

func setupInboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "inbox_test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func createInboxOrg(t *testing.T, db *gorm.DB, slug string) *models.Org {
	t.Helper()
	org := &models.Org{Name: slug, Slug: slug, Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org
}

func makePlatformAdmin(t *testing.T, db *gorm.DB, userID string) {
	t.Helper()
	pa := &models.PlatformAdmin{UserID: userID, GrantedBy: "bootstrap", IsActive: true}
	require.NoError(t, db.Create(pa).Error)
}

func withInboxAuthCtx(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

func withInboxOrgParam(r *http.Request, orgID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", orgID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withInboxOrgAndIDParam(r *http.Request, orgID, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", orgID)
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- Repository tests ---

func TestEmailInboxRepository_CreateAndFind(t *testing.T) {
	db := setupInboxTestDB(t)
	org := createInboxOrg(t, db, "repo-inbox-org")
	repo := NewEmailInboxRepository(db)
	ctx := context.Background()

	inbox := &models.EmailInbox{
		OrgID:         org.ID,
		Name:          "Support",
		EmailAddress:  "support@acme.com",
		IMAPHost:      "imap.gmail.com",
		IMAPPort:      993,
		Username:      "support@acme.com",
		Password:      "app-password",
		Mailbox:       "INBOX",
		RoutingAction: models.RoutingActionSupportTicket,
		Enabled:       true,
	}
	require.NoError(t, repo.Create(ctx, inbox))
	assert.NotEmpty(t, inbox.ID)

	found, err := repo.FindByID(ctx, inbox.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, inbox.ID, found.ID)
	assert.Equal(t, "Support", found.Name)
	assert.Equal(t, models.RoutingActionSupportTicket, found.RoutingAction)
}

func TestEmailInboxRepository_FindByID_NotFound(t *testing.T) {
	db := setupInboxTestDB(t)
	repo := NewEmailInboxRepository(db)
	found, err := repo.FindByID(context.Background(), "nonexistent-id")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestEmailInboxRepository_ListByOrg(t *testing.T) {
	db := setupInboxTestDB(t)
	org1 := createInboxOrg(t, db, "repo-list-org1")
	org2 := createInboxOrg(t, db, "repo-list-org2")
	repo := NewEmailInboxRepository(db)
	ctx := context.Background()

	for _, name := range []string{"Support", "Sales"} {
		require.NoError(t, repo.Create(ctx, &models.EmailInbox{
			OrgID:         org1.ID,
			Name:          name,
			IMAPHost:      "imap.test",
			IMAPPort:      993,
			Username:      "u",
			Password:      "p",
			Mailbox:       "INBOX",
			RoutingAction: models.RoutingActionSupportTicket,
		}))
	}
	require.NoError(t, repo.Create(ctx, &models.EmailInbox{
		OrgID:         org2.ID,
		Name:          "Other",
		IMAPHost:      "imap.test",
		IMAPPort:      993,
		Username:      "u",
		Password:      "p",
		Mailbox:       "INBOX",
		RoutingAction: models.RoutingActionGeneral,
	}))

	inboxes, err := repo.ListByOrg(ctx, org1.ID)
	require.NoError(t, err)
	assert.Len(t, inboxes, 2)

	// Isolation: org2 should only see its own inbox.
	org2Inboxes, err := repo.ListByOrg(ctx, org2.ID)
	require.NoError(t, err)
	assert.Len(t, org2Inboxes, 1)
}

func TestEmailInboxRepository_SaveAndSoftDelete(t *testing.T) {
	db := setupInboxTestDB(t)
	org := createInboxOrg(t, db, "repo-save-org")
	repo := NewEmailInboxRepository(db)
	ctx := context.Background()

	inbox := &models.EmailInbox{
		OrgID:         org.ID,
		Name:          "Support",
		IMAPHost:      "imap.test",
		IMAPPort:      993,
		Username:      "u",
		Password:      "p",
		Mailbox:       "INBOX",
		RoutingAction: models.RoutingActionSupportTicket,
		Enabled:       true,
	}
	require.NoError(t, repo.Create(ctx, inbox))

	// Update name via Save.
	inbox.Name = "Support Renamed"
	require.NoError(t, repo.Save(ctx, inbox))

	found, err := repo.FindByID(ctx, inbox.ID)
	require.NoError(t, err)
	assert.Equal(t, "Support Renamed", found.Name)

	// Soft delete.
	require.NoError(t, repo.SoftDelete(ctx, inbox))
	deleted, err := repo.FindByID(ctx, inbox.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

// --- Service tests ---

func TestEmailInboxService_Create_Valid(t *testing.T) {
	db := setupInboxTestDB(t)
	org := createInboxOrg(t, db, "svc-create-org")
	svc := NewEmailInboxService(NewEmailInboxRepository(db))

	inbox, err := svc.Create(context.Background(), org.ID, CreateInboxInput{
		Name:          "Support",
		EmailAddress:  "support@acme.com",
		IMAPHost:      "imap.gmail.com",
		IMAPPort:      993,
		Username:      "support@acme.com",
		Password:      "app-pass",
		RoutingAction: models.RoutingActionSupportTicket,
		Enabled:       true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, inbox.ID)
	assert.Equal(t, passwordRedacted, inbox.Password) // masked
	assert.Equal(t, "INBOX", inbox.Mailbox)           // default applied
}

func TestEmailInboxService_Create_Validation(t *testing.T) {
	db := setupInboxTestDB(t)
	svc := NewEmailInboxService(NewEmailInboxRepository(db))
	ctx := context.Background()

	tests := []struct {
		name    string
		input   CreateInboxInput
		wantErr string
	}{
		{"missing name", CreateInboxInput{IMAPHost: "h", IMAPPort: 993, Username: "u", Password: "p"}, "name is required"},
		{"missing host", CreateInboxInput{Name: "n", IMAPPort: 993, Username: "u", Password: "p"}, "imap_host is required"},
		{"zero port", CreateInboxInput{Name: "n", IMAPHost: "h", Username: "u", Password: "p"}, "imap_port must be a positive integer"},
		{"missing username", CreateInboxInput{Name: "n", IMAPHost: "h", IMAPPort: 993, Password: "p"}, "username is required"},
		{"missing password", CreateInboxInput{Name: "n", IMAPHost: "h", IMAPPort: 993, Username: "u"}, "password is required"},
		{"invalid action", CreateInboxInput{Name: "n", IMAPHost: "h", IMAPPort: 993, Username: "u", Password: "p", RoutingAction: "bad"}, "is not valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, "org-1", tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestEmailInboxService_ListMasksPasswords(t *testing.T) {
	db := setupInboxTestDB(t)
	org := createInboxOrg(t, db, "svc-list-org")
	svc := NewEmailInboxService(NewEmailInboxRepository(db))
	ctx := context.Background()

	_, err := svc.Create(ctx, org.ID, CreateInboxInput{
		Name:     "Support",
		IMAPHost: "imap.test",
		IMAPPort: 993,
		Username: "user",
		Password: "secret",
	})
	require.NoError(t, err)

	inboxes, err := svc.List(ctx, org.ID)
	require.NoError(t, err)
	require.Len(t, inboxes, 1)
	assert.Equal(t, passwordRedacted, inboxes[0].Password)
	assert.NotEqual(t, "secret", inboxes[0].Password)
}

func TestEmailInboxService_Update_KeepsPasswordWhenBlank(t *testing.T) {
	db := setupInboxTestDB(t)
	org := createInboxOrg(t, db, "svc-update-org")
	repo := NewEmailInboxRepository(db)
	svc := NewEmailInboxService(repo)
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInboxInput{
		Name:     "Support",
		IMAPHost: "imap.test",
		IMAPPort: 993,
		Username: "user",
		Password: "original-secret",
	})
	require.NoError(t, err)

	// Update name only — blank password should not overwrite stored secret.
	updated, err := svc.Update(ctx, org.ID, created.ID, UpdateInboxInput{
		Name:     "Renamed Support",
		IMAPHost: "imap.test",
		IMAPPort: 993,
		Username: "user",
		Password: "", // blank = keep existing
		Enabled:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Renamed Support", updated.Name)

	// Verify stored password is preserved.
	raw, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "original-secret", raw.Password)
}

func TestEmailInboxService_Update_NotFound(t *testing.T) {
	db := setupInboxTestDB(t)
	svc := NewEmailInboxService(NewEmailInboxRepository(db))
	updated, err := svc.Update(context.Background(), "org-1", "nonexistent", UpdateInboxInput{})
	require.NoError(t, err)
	assert.Nil(t, updated)
}

func TestEmailInboxService_Delete(t *testing.T) {
	db := setupInboxTestDB(t)
	org := createInboxOrg(t, db, "svc-delete-org")
	svc := NewEmailInboxService(NewEmailInboxRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInboxInput{
		Name:     "To Delete",
		IMAPHost: "imap.test",
		IMAPPort: 993,
		Username: "user",
		Password: "pass",
	})
	require.NoError(t, err)

	deleted, err := svc.Delete(ctx, org.ID, created.ID)
	require.NoError(t, err)
	assert.True(t, deleted)

	// Second delete returns false.
	deleted, err = svc.Delete(ctx, org.ID, created.ID)
	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestEmailInboxService_CrossOrgIsolation(t *testing.T) {
	db := setupInboxTestDB(t)
	org1 := createInboxOrg(t, db, "svc-iso-org1")
	org2 := createInboxOrg(t, db, "svc-iso-org2")
	svc := NewEmailInboxService(NewEmailInboxRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org1.ID, CreateInboxInput{
		Name:     "Support",
		IMAPHost: "imap.test",
		IMAPPort: 993,
		Username: "u",
		Password: "p",
	})
	require.NoError(t, err)

	// org2 should not see org1's inbox.
	inbox, err := svc.Get(ctx, org2.ID, created.ID)
	require.NoError(t, err)
	assert.Nil(t, inbox)

	// org2 should not be able to delete org1's inbox.
	deleted, err := svc.Delete(ctx, org2.ID, created.ID)
	require.NoError(t, err)
	assert.False(t, deleted)
}

// --- RoutingAction model tests ---

func TestRoutingAction_IsValid(t *testing.T) {
	assert.True(t, models.RoutingActionSupportTicket.IsValid())
	assert.True(t, models.RoutingActionSalesLead.IsValid())
	assert.True(t, models.RoutingActionGeneral.IsValid())
	assert.False(t, models.RoutingAction("unknown").IsValid())
	assert.False(t, models.RoutingAction("").IsValid())
}

// --- Handler tests ---

// mockRestarter is a test double for InboxRestarter.
type mockRestarter struct {
	calls []models.EmailInbox
}

func (m *mockRestarter) RestartInbox(inbox models.EmailInbox) {
	m.calls = append(m.calls, inbox)
}

func buildInboxHandler(db *gorm.DB) (*EmailInboxHandler, *Service) {
	repo := NewEmailInboxRepository(db)
	svc := NewEmailInboxService(repo)
	channelSvc := NewService(NewRepository(db))
	h := NewEmailInboxHandler(svc, channelSvc, &mockRestarter{})
	return h, channelSvc
}

func TestEmailInboxHandler_List_Unauthenticated(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-list-unauth-org")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withInboxOrgParam(req, org.ID)
	w := httptest.NewRecorder()
	h.ListInboxes(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailInboxHandler_List_Forbidden(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-list-forbid-org")

	// User is a viewer — no admin membership, not a platform admin.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withInboxAuthCtx(req, "viewer-uid")
	req = withInboxOrgParam(req, org.ID)
	w := httptest.NewRecorder()
	h.ListInboxes(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestEmailInboxHandler_List_PlatformAdmin(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-list-pa-org")
	makePlatformAdmin(t, db, "pa-uid")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withInboxAuthCtx(req, "pa-uid")
	req = withInboxOrgParam(req, org.ID)
	w := httptest.NewRecorder()
	h.ListInboxes(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body, "data")
}

func TestEmailInboxHandler_Create_Success(t *testing.T) {
	db := setupInboxTestDB(t)
	restarter := &mockRestarter{}
	repo := NewEmailInboxRepository(db)
	svc := NewEmailInboxService(repo)
	channelSvc := NewService(NewRepository(db))
	h := NewEmailInboxHandler(svc, channelSvc, restarter)

	org := createInboxOrg(t, db, "hdl-create-org")
	makePlatformAdmin(t, db, "pa-create-uid")

	body := `{"name":"Support","email_address":"support@acme.com","imap_host":"imap.gmail.com","imap_port":993,"username":"support@acme.com","password":"app-pass","routing_action":"support_ticket","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req = withInboxAuthCtx(req, "pa-create-uid")
	req = withInboxOrgParam(req, org.ID)
	w := httptest.NewRecorder()
	h.CreateInbox(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var created models.EmailInbox
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, passwordRedacted, created.Password)
	assert.Equal(t, models.RoutingActionSupportTicket, created.RoutingAction)

	// Watcher should have been notified.
	assert.Len(t, restarter.calls, 1)
}

func TestEmailInboxHandler_Create_BadBody(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-create-bad-org")
	makePlatformAdmin(t, db, "pa-bad-uid")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))
	req = withInboxAuthCtx(req, "pa-bad-uid")
	req = withInboxOrgParam(req, org.ID)
	w := httptest.NewRecorder()
	h.CreateInbox(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailInboxHandler_Create_Validation(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-create-val-org")
	makePlatformAdmin(t, db, "pa-val-uid")

	// Missing required fields.
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Support"}`))
	req = withInboxAuthCtx(req, "pa-val-uid")
	req = withInboxOrgParam(req, org.ID)
	w := httptest.NewRecorder()
	h.CreateInbox(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailInboxHandler_Update_NotFound(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-update-nf-org")
	makePlatformAdmin(t, db, "pa-upd-uid")

	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"name":"Renamed"}`))
	req = withInboxAuthCtx(req, "pa-upd-uid")
	req = withInboxOrgAndIDParam(req, org.ID, "nonexistent")
	w := httptest.NewRecorder()
	h.UpdateInbox(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEmailInboxHandler_Delete_Success(t *testing.T) {
	db := setupInboxTestDB(t)
	repo := NewEmailInboxRepository(db)
	svc := NewEmailInboxService(repo)
	channelSvc := NewService(NewRepository(db))
	restarter := &mockRestarter{}
	h := NewEmailInboxHandler(svc, channelSvc, restarter)

	org := createInboxOrg(t, db, "hdl-delete-org")
	makePlatformAdmin(t, db, "pa-del-uid")

	// Create an inbox to delete.
	created, err := svc.Create(context.Background(), org.ID, CreateInboxInput{
		Name:     "To Delete",
		IMAPHost: "imap.test",
		IMAPPort: 993,
		Username: "u",
		Password: "p",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withInboxAuthCtx(req, "pa-del-uid")
	req = withInboxOrgAndIDParam(req, org.ID, created.ID)
	w := httptest.NewRecorder()
	h.DeleteInbox(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestEmailInboxHandler_Delete_NotFound(t *testing.T) {
	db := setupInboxTestDB(t)
	h, _ := buildInboxHandler(db)
	org := createInboxOrg(t, db, "hdl-delete-nf-org")
	makePlatformAdmin(t, db, "pa-dnf-uid")

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withInboxAuthCtx(req, "pa-dnf-uid")
	req = withInboxOrgAndIDParam(req, org.ID, "nonexistent")
	w := httptest.NewRecorder()
	h.DeleteInbox(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
