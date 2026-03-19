package channel

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Test helpers ---

func setupTestDB(t *testing.T) *gorm.DB {
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
	return db
}

// createTestOrg creates an Org and returns it.
func createTestOrg(t *testing.T, db *gorm.DB, slug string) *models.Org {
	t.Helper()
	org := &models.Org{Name: slug, Slug: slug, Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org
}

// createTestSpaceAndBoard creates a Space and Board under the org.
func createTestSpaceAndBoard(t *testing.T, db *gorm.DB, orgID string, spaceType models.SpaceType) (*models.Space, *models.Board) {
	t.Helper()
	space := &models.Space{OrgID: orgID, Name: "Test Space", Slug: "test-space-" + orgID[:8], Type: spaceType, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Test Board", Slug: "test-board-" + orgID[:8], Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return space, board
}

// createAdminMembership gives userID admin role in orgID.
func createAdminMembership(t *testing.T, db *gorm.DB, orgID, userID string) {
	t.Helper()
	m := &models.OrgMembership{OrgID: orgID, UserID: userID, Role: models.RoleAdmin}
	require.NoError(t, db.Create(m).Error)
}

// createPlatformAdmin inserts an active platform admin record for userID.
func createPlatformAdmin(t *testing.T, db *gorm.DB, userID string) {
	t.Helper()
	pa := &models.PlatformAdmin{UserID: userID, GrantedBy: "bootstrap", IsActive: true}
	require.NoError(t, db.Create(pa).Error)
}

// withAuthCtx injects a UserContext into the request.
func withAuthCtx(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

// withChiParams adds chi URL params to the request.
func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withAuthAndChi combines auth context and chi params in a single call.
func withAuthAndChi(r *http.Request, userID string, params map[string]string) *http.Request {
	return withChiParams(withAuthCtx(r, userID), params)
}

// --- Repository tests ---

func TestRepository_UpsertConfig_Create(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-upsert-org")
	repo := NewRepository(db)
	ctx := context.Background()

	cfg := &models.ChannelConfig{
		OrgID:       org.ID,
		ChannelType: models.ChannelTypeEmail,
		Settings:    `{"imap_host":"mail.example.com"}`,
		Enabled:     true,
	}
	require.NoError(t, repo.UpsertConfig(ctx, cfg))
	assert.NotEmpty(t, cfg.ID)
	assert.Equal(t, org.ID, cfg.OrgID)
}

func TestRepository_UpsertConfig_Update(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-upsert-upd-org")
	repo := NewRepository(db)
	ctx := context.Background()

	cfg := &models.ChannelConfig{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, Settings: `{}`, Enabled: false}
	require.NoError(t, repo.UpsertConfig(ctx, cfg))
	firstID := cfg.ID

	// Update same org+type should update in place.
	cfg2 := &models.ChannelConfig{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, Settings: `{"imap_host":"new.host"}`, Enabled: true}
	require.NoError(t, repo.UpsertConfig(ctx, cfg2))
	assert.Equal(t, firstID, cfg2.ID) // Same record.
	assert.True(t, cfg2.Enabled)
}

func TestRepository_FindConfig_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	cfg, err := repo.FindConfig(context.Background(), "no-org", models.ChannelTypeEmail)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestRepository_FindConfig_Found(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-find-org")
	repo := NewRepository(db)
	ctx := context.Background()

	saved := &models.ChannelConfig{OrgID: org.ID, ChannelType: models.ChannelTypeVoice, Settings: `{"livekit_api_key":"key"}`, Enabled: true}
	require.NoError(t, repo.UpsertConfig(ctx, saved))

	found, err := repo.FindConfig(ctx, org.ID, models.ChannelTypeVoice)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, saved.ID, found.ID)
}

func TestRepository_ListConfigs(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-list-org")
	repo := NewRepository(db)
	ctx := context.Background()

	for _, ct := range models.ValidChannelTypes() {
		cfg := &models.ChannelConfig{OrgID: org.ID, ChannelType: ct, Settings: "{}"}
		require.NoError(t, repo.UpsertConfig(ctx, cfg))
	}

	cfgs, err := repo.ListConfigs(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, cfgs, 3)
}

func TestRepository_DLQ_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-dlq-org")
	repo := NewRepository(db)
	ctx := context.Background()

	now := time.Now()
	evt := &models.DeadLetterEvent{
		OrgID:         org.ID,
		ChannelType:   models.ChannelTypeEmail,
		EventPayload:  `{"key":"val"}`,
		ErrorMessage:  "connection refused",
		Attempts:      5,
		LastAttemptAt: &now,
		Status:        models.DLQStatusFailed,
	}
	require.NoError(t, repo.CreateDLQEvent(ctx, evt))
	assert.NotEmpty(t, evt.ID)

	found, err := repo.FindDLQEvent(ctx, evt.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, evt.ID, found.ID)
	assert.Equal(t, "connection refused", found.ErrorMessage)
}

func TestRepository_DLQ_FindNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	found, err := repo.FindDLQEvent(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_DLQ_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-dlq-upd-org")
	repo := NewRepository(db)
	ctx := context.Background()

	evt := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeChat, EventPayload: "{}", Status: models.DLQStatusFailed}
	require.NoError(t, repo.CreateDLQEvent(ctx, evt))

	evt.Status = models.DLQStatusDismissed
	require.NoError(t, repo.UpdateDLQEvent(ctx, evt))

	found, err := repo.FindDLQEvent(ctx, evt.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DLQStatusDismissed, found.Status)
}

func TestRepository_DLQ_ListWithFilters(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-dlq-list-org")
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		e := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
		require.NoError(t, repo.CreateDLQEvent(ctx, e))
	}
	voice := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeVoice, EventPayload: "{}", Status: models.DLQStatusFailed}
	require.NoError(t, repo.CreateDLQEvent(ctx, voice))

	// Filter by channel type.
	evts, pi, err := repo.ListDLQEvents(ctx, org.ID, models.ChannelTypeEmail, "", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, evts, 3)
	assert.False(t, pi.HasMore)

	// Filter by status — dismissed → should return 0.
	evts, _, err = repo.ListDLQEvents(ctx, org.ID, "", models.DLQStatusDismissed, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Empty(t, evts)

	// No filter — all 4 events.
	evts, _, err = repo.ListDLQEvents(ctx, org.ID, "", "", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, evts, 4)
}

func TestRepository_DLQ_Pagination(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-dlq-page-org")
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		e := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
		require.NoError(t, repo.CreateDLQEvent(ctx, e))
	}

	evts, pi, err := repo.ListDLQEvents(ctx, org.ID, "", "", pagination.Params{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, evts, 2)
	assert.True(t, pi.HasMore)
}

func TestRepository_CountRecentDLQEvents(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-count-dlq-org")
	repo := NewRepository(db)
	ctx := context.Background()

	// 3 recent failed email events.
	for i := 0; i < 3; i++ {
		e := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
		require.NoError(t, repo.CreateDLQEvent(ctx, e))
	}
	// 1 dismissed (should not count).
	dismissed := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusDismissed}
	require.NoError(t, repo.CreateDLQEvent(ctx, dismissed))

	count, err := repo.CountRecentDLQEvents(ctx, org.ID, models.ChannelTypeEmail)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestRepository_IsOrgAdmin(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "repo-admin-org")
	repo := NewRepository(db)
	ctx := context.Background()

	createAdminMembership(t, db, org.ID, "admin-user")
	viewerM := &models.OrgMembership{OrgID: org.ID, UserID: "viewer-user", Role: models.RoleViewer}
	require.NoError(t, db.Create(viewerM).Error)

	isAdmin, err := repo.IsOrgAdmin(ctx, org.ID, "admin-user")
	require.NoError(t, err)
	assert.True(t, isAdmin)

	isViewer, err := repo.IsOrgAdmin(ctx, org.ID, "viewer-user")
	require.NoError(t, err)
	assert.False(t, isViewer)

	isNone, err := repo.IsOrgAdmin(ctx, org.ID, "no-such-user")
	require.NoError(t, err)
	assert.False(t, isNone)
}

func TestRepository_IsPlatformAdmin(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Non-existent user → false.
	isAdmin, err := repo.IsPlatformAdmin(ctx, "no-such-user")
	require.NoError(t, err)
	assert.False(t, isAdmin)

	// Active platform admin → true.
	createPlatformAdmin(t, db, "pa-user")
	isAdmin, err = repo.IsPlatformAdmin(ctx, "pa-user")
	require.NoError(t, err)
	assert.True(t, isAdmin)

	// Inactive platform admin → false.
	// Use create-then-update to avoid GORM skipping the zero-value bool (false) on Create
	// when the column has gorm:"default:true".
	inactive := &models.PlatformAdmin{UserID: "inactive-pa", GrantedBy: "bootstrap", IsActive: true}
	require.NoError(t, db.Create(inactive).Error)
	require.NoError(t, db.Model(inactive).Update("is_active", false).Error)
	isAdmin, err = repo.IsPlatformAdmin(ctx, "inactive-pa")
	require.NoError(t, err)
	assert.False(t, isAdmin)
}

// --- Service tests ---

func TestService_UpsertConfig_Validation(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-val-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	tests := []struct {
		name        string
		channelType models.ChannelType
		settings    string
		wantErr     bool
		errContains string
	}{
		{"valid email empty settings", models.ChannelTypeEmail, "{}", false, ""},
		{"valid voice empty settings", models.ChannelTypeVoice, "", false, ""},
		{"valid chat empty settings", models.ChannelTypeChat, "{}", false, ""},
		{"invalid channel type", "sms", "{}", true, "invalid channel type"},
		{"invalid email settings JSON", models.ChannelTypeEmail, "not-json", true, "invalid email settings"},
		{"email requires imap_host", models.ChannelTypeEmail, `{"imap_host":"h.com","imap_port":993,"username":"u"}`, false, ""},
		{"email missing username", models.ChannelTypeEmail, `{"imap_host":"h.com","imap_port":993}`, true, "username is required"},
		{"voice requires api key", models.ChannelTypeVoice, `{"livekit_api_key":"k","livekit_project_url":"wss://p"}`, false, ""},
		{"voice missing project url", models.ChannelTypeVoice, `{"livekit_api_key":"k"}`, true, "livekit_project_url is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpsertConfig(ctx, org.ID, tt.channelType, PutConfigInput{Settings: tt.settings})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_UpsertConfig_SecretsMasked(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-mask-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	settings := `{"imap_host":"mail.test","imap_port":993,"username":"user","password":"s3cr3t"}`
	cfg, err := svc.UpsertConfig(ctx, org.ID, models.ChannelTypeEmail, PutConfigInput{Settings: settings, Enabled: true})
	require.NoError(t, err)
	assert.NotContains(t, cfg.Settings, "s3cr3t")
	assert.Contains(t, cfg.Settings, "[REDACTED]")
}

func TestService_UpsertConfig_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-idem-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	cfg1, err := svc.UpsertConfig(ctx, org.ID, models.ChannelTypeChat, PutConfigInput{Settings: `{"embed_key":"k1"}`, Enabled: false})
	require.NoError(t, err)

	cfg2, err := svc.UpsertConfig(ctx, org.ID, models.ChannelTypeChat, PutConfigInput{Settings: `{"embed_key":"k2"}`, Enabled: true})
	require.NoError(t, err)
	assert.Equal(t, cfg1.ID, cfg2.ID, "should update the same record")
	assert.True(t, cfg2.Enabled)
}

func TestService_GetConfig_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db))
	cfg, err := svc.GetConfig(context.Background(), "no-org", models.ChannelTypeEmail)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestService_GetConfig_MasksSecrets(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-get-mask-org")
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	// Insert directly with secret in settings.
	raw := &models.ChannelConfig{
		OrgID:       org.ID,
		ChannelType: models.ChannelTypeVoice,
		Settings:    `{"livekit_api_key":"key","livekit_api_secret":"topsecret","livekit_project_url":"wss://p"}`,
		Enabled:     true,
	}
	require.NoError(t, repo.UpsertConfig(ctx, raw))

	cfg, err := svc.GetConfig(ctx, org.ID, models.ChannelTypeVoice)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.NotContains(t, cfg.Settings, "topsecret")
	assert.Contains(t, cfg.Settings, "[REDACTED]")
}

func TestService_GetHealth(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-health-org")
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	// Enable email channel.
	emailCfg := &models.ChannelConfig{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, Settings: "{}", Enabled: true}
	require.NoError(t, repo.UpsertConfig(ctx, emailCfg))

	// Add 3 recent email DLQ errors → degraded.
	for i := 0; i < 3; i++ {
		e := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
		require.NoError(t, repo.CreateDLQEvent(ctx, e))
	}

	health, err := svc.GetHealth(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, health, 3) // One per channel type.

	byType := make(map[models.ChannelType]ChannelHealth)
	for _, h := range health {
		byType[h.ChannelType] = h
	}

	assert.Equal(t, HealthStatusDegraded, byType[models.ChannelTypeEmail].Status)
	assert.True(t, byType[models.ChannelTypeEmail].Enabled)
	assert.Equal(t, int64(3), byType[models.ChannelTypeEmail].ErrorCount)

	// Voice and chat are disabled → down.
	assert.Equal(t, HealthStatusDown, byType[models.ChannelTypeVoice].Status)
	assert.Equal(t, HealthStatusDown, byType[models.ChannelTypeChat].Status)
}

func TestService_GetHealth_HighErrorRate_Down(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-health-down-org")
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	emailCfg := &models.ChannelConfig{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, Settings: "{}", Enabled: true}
	require.NoError(t, repo.UpsertConfig(ctx, emailCfg))

	// 6 errors → down.
	for i := 0; i < 6; i++ {
		e := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
		require.NoError(t, repo.CreateDLQEvent(ctx, e))
	}

	health, err := svc.GetHealth(ctx, org.ID)
	require.NoError(t, err)
	for _, h := range health {
		if h.ChannelType == models.ChannelTypeEmail {
			assert.Equal(t, HealthStatusDown, h.Status)
		}
	}
}

func TestService_DLQ_RetryAndDismiss(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-dlq-rd-org")
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	evt := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
	require.NoError(t, repo.CreateDLQEvent(ctx, evt))

	// Retry.
	updated, err := svc.RetryDLQEvent(ctx, org.ID, evt.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, models.DLQStatusRetrying, updated.Status)

	// Dismiss.
	dismissed, err := svc.DismissDLQEvent(ctx, org.ID, evt.ID)
	require.NoError(t, err)
	require.NotNil(t, dismissed)
	assert.Equal(t, models.DLQStatusDismissed, dismissed.Status)
}

func TestService_DLQ_CrossOrgIsolation(t *testing.T) {
	db := setupTestDB(t)
	org1 := createTestOrg(t, db, "svc-dlq-iso-org1")
	org2 := createTestOrg(t, db, "svc-dlq-iso-org2")
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	evt := &models.DeadLetterEvent{OrgID: org1.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
	require.NoError(t, repo.CreateDLQEvent(ctx, evt))

	// org2 should not see org1's event.
	result, err := svc.RetryDLQEvent(ctx, org2.ID, evt.ID)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// --- Handler tests ---

func TestHandler_GetConfig_Default(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-getcfg-org")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiParams(req, map[string]string{"org": org.ID, "type": "email"})
	w := httptest.NewRecorder()
	h.GetConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "email", body["channel_type"])
}

func TestHandler_GetConfig_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiParams(req, map[string]string{"org": "org1", "type": "sms"})
	w := httptest.NewRecorder()
	h.GetConfig(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_PutConfig_Unauthenticated(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-putcfg-unauth-org")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"enabled":true}`))
	req = withChiParams(req, map[string]string{"org": org.ID, "type": "email"})
	w := httptest.NewRecorder()
	h.PutConfig(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_PutConfig_Forbidden(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-putcfg-forbid-org")
	// User has viewer role (not admin).
	viewer := &models.OrgMembership{OrgID: org.ID, UserID: "viewer-uid", Role: models.RoleViewer}
	require.NoError(t, db.Create(viewer).Error)

	h := NewHandler(NewService(NewRepository(db)))
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"enabled":true}`))
	req = withAuthAndChi(req, "viewer-uid", map[string]string{"org": org.ID, "type": "email"})
	w := httptest.NewRecorder()
	h.PutConfig(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_PutConfig_AdminSuccess(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-putcfg-ok-org")
	createAdminMembership(t, db, org.ID, "admin-uid")
	h := NewHandler(NewService(NewRepository(db)))

	body := `{"settings":"{\"embed_key\":\"abc\"}","enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req = withAuthAndChi(req, "admin-uid", map[string]string{"org": org.ID, "type": "chat"})
	w := httptest.NewRecorder()
	h.PutConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PutConfig_PlatformAdminSuccess(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-putcfg-pa-org")
	// Platform admin has no org membership but must still be allowed.
	createPlatformAdmin(t, db, "platform-admin-uid")
	h := NewHandler(NewService(NewRepository(db)))

	body := `{"settings":"{}","enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req = withAuthAndChi(req, "platform-admin-uid", map[string]string{"org": org.ID, "type": "email"})
	w := httptest.NewRecorder()
	h.PutConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PutConfig_BadBody(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-putcfg-bad-org")
	createAdminMembership(t, db, org.ID, "admin-uid2")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader("not-json"))
	req = withAuthAndChi(req, "admin-uid2", map[string]string{"org": org.ID, "type": "email"})
	w := httptest.NewRecorder()
	h.PutConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PutConfig_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-putcfg-badtype-org")
	createAdminMembership(t, db, org.ID, "admin-uid3")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{}`))
	req = withAuthAndChi(req, "admin-uid3", map[string]string{"org": org.ID, "type": "sms"})
	w := httptest.NewRecorder()
	h.PutConfig(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ListDLQ_AdminOnly(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-dlq-list-org")
	createAdminMembership(t, db, org.ID, "admin-uid4")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withAuthAndChi(req, "admin-uid4", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.ListDLQ(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RetryDLQ_NotFound(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-dlq-retry-org")
	createAdminMembership(t, db, org.ID, "admin-uid5")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withAuthAndChi(req, "admin-uid5", map[string]string{"org": org.ID, "id": "nonexistent-id"})
	w := httptest.NewRecorder()
	h.RetryDLQ(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RetryDLQ_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-dlq-retry-ok-org")
	createAdminMembership(t, db, org.ID, "admin-uid6")
	repo := NewRepository(db)
	h := NewHandler(NewService(repo))
	ctx := context.Background()

	evt := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, EventPayload: "{}", Status: models.DLQStatusFailed}
	require.NoError(t, repo.CreateDLQEvent(ctx, evt))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withAuthAndChi(req, "admin-uid6", map[string]string{"org": org.ID, "id": evt.ID})
	w := httptest.NewRecorder()
	h.RetryDLQ(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_DismissDLQ_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-dlq-dismiss-org")
	createAdminMembership(t, db, org.ID, "admin-uid7")
	repo := NewRepository(db)
	h := NewHandler(NewService(repo))
	ctx := context.Background()

	evt := &models.DeadLetterEvent{OrgID: org.ID, ChannelType: models.ChannelTypeVoice, EventPayload: "{}", Status: models.DLQStatusFailed}
	require.NoError(t, repo.CreateDLQEvent(ctx, evt))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withAuthAndChi(req, "admin-uid7", map[string]string{"org": org.ID, "id": evt.ID})
	w := httptest.NewRecorder()
	h.DismissDLQ(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, string(models.DLQStatusDismissed), body["status"])
}

func TestHandler_GetHealth(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-health-org")
	h := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiParams(req, map[string]string{"org": org.ID})
	w := httptest.NewRecorder()
	h.GetHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body, "channels")
}

// --- Gateway tests ---

func TestGateway_Process_CreatesLeadThread(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "gw-new-lead-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	bus := eventbus.New()
	gw := NewGateway(db, bus)
	ctx := context.Background()

	evt := &InboundEvent{
		ChannelType:      models.ChannelTypeEmail,
		OrgID:            org.ID,
		ExternalID:       "msg-001",
		SenderIdentifier: "alice@example.com",
		Subject:          "Interested in your product",
		Body:             "Hi, I want to know more.",
	}

	err := gw.Process(ctx, evt)
	require.NoError(t, err)
	assert.NotEmpty(t, evt.ID)

	// Verify a thread was created.
	var threads []models.Thread
	require.NoError(t, db.Find(&threads).Error)
	assert.Len(t, threads, 1)
	assert.Contains(t, threads[0].Title, "Interested in your product")

	// Verify a message was created on the thread.
	var messages []models.Message
	require.NoError(t, db.Where("thread_id = ?", threads[0].ID).Find(&messages).Error)
	assert.Len(t, messages, 1)
	assert.Equal(t, models.MessageTypeEmail, messages[0].Type)
}

func TestGateway_Process_MatchesExistingThreadByExternalID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "gw-match-ext-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	// Create existing thread with external_id in metadata.
	existingThread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Existing Lead",
		Slug:     "existing-lead",
		Metadata: `{"external_id":"msg-existing","contact_email":"bob@example.com"}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(existingThread).Error)

	bus := eventbus.New()
	gw := NewGateway(db, bus)
	ctx := context.Background()

	evt := &InboundEvent{
		ChannelType:      models.ChannelTypeEmail,
		OrgID:            org.ID,
		ExternalID:       "msg-existing",
		SenderIdentifier: "bob@example.com",
		Body:             "Follow-up message.",
	}

	err := gw.Process(ctx, evt)
	require.NoError(t, err)

	// Should have added a message to existing thread, not created a new one.
	var threadCount int64
	require.NoError(t, db.Model(&models.Thread{}).Count(&threadCount).Error)
	assert.Equal(t, int64(1), threadCount)

	var msgCount int64
	require.NoError(t, db.Model(&models.Message{}).Where("thread_id = ?", existingThread.ID).Count(&msgCount).Error)
	assert.Equal(t, int64(1), msgCount)
}

func TestGateway_Process_MatchesExistingThreadBySender(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "gw-match-sender-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	// Existing thread with contact_email matching sender.
	existingThread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Existing Sender Lead",
		Slug:     "existing-sender-lead",
		Metadata: `{"contact_email":"carol@example.com"}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(existingThread).Error)

	bus := eventbus.New()
	gw := NewGateway(db, bus)
	ctx := context.Background()

	// No external_id match, but sender matches.
	evt := &InboundEvent{
		ChannelType:      models.ChannelTypeEmail,
		OrgID:            org.ID,
		ExternalID:       "new-unique-id",
		SenderIdentifier: "carol@example.com",
		Body:             "Second email from carol.",
	}

	err := gw.Process(ctx, evt)
	require.NoError(t, err)

	var threadCount int64
	require.NoError(t, db.Model(&models.Thread{}).Count(&threadCount).Error)
	assert.Equal(t, int64(1), threadCount, "should reuse existing thread")
}

func TestGateway_Process_VoiceMessageType(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "gw-voice-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeGeneral)

	bus := eventbus.New()
	gw := NewGateway(db, bus)
	ctx := context.Background()

	evt := &InboundEvent{
		ChannelType:      models.ChannelTypeVoice,
		OrgID:            org.ID,
		SenderIdentifier: "+15555551234",
		Body:             "Call transcript.",
	}
	require.NoError(t, gw.Process(ctx, evt))

	var msg models.Message
	require.NoError(t, db.First(&msg).Error)
	assert.Equal(t, models.MessageTypeCallLog, msg.Type)
}

func TestGateway_Process_ChatMessageType(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "gw-chat-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeGeneral)

	bus := eventbus.New()
	gw := NewGateway(db, bus)
	ctx := context.Background()

	evt := &InboundEvent{
		ChannelType:      models.ChannelTypeChat,
		OrgID:            org.ID,
		SenderIdentifier: "session-abc",
		Body:             "Help me with X.",
	}
	require.NoError(t, gw.Process(ctx, evt))

	var msg models.Message
	require.NoError(t, db.First(&msg).Error)
	assert.Equal(t, models.MessageTypeComment, msg.Type)
}

func TestGateway_Process_NilEvent(t *testing.T) {
	db := setupTestDB(t)
	gw := NewGateway(db, nil)
	err := gw.Process(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event is required")
}

func TestGateway_Process_MissingOrgID(t *testing.T) {
	db := setupTestDB(t)
	gw := NewGateway(db, nil)
	evt := &InboundEvent{ChannelType: models.ChannelTypeEmail}
	err := gw.Process(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OrgID is required")
}

func TestGateway_Process_InvalidChannelType(t *testing.T) {
	db := setupTestDB(t)
	gw := NewGateway(db, nil)
	evt := &InboundEvent{OrgID: "org1", ChannelType: "sms"}
	err := gw.Process(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel type")
}

func TestGateway_Process_PublishesToEventBus(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "gw-bus-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	bus := eventbus.New()
	t.Cleanup(bus.Close)
	ch, unsub := bus.Subscribe("channel.inbound", 16)
	t.Cleanup(unsub)

	gw := NewGateway(db, bus)
	evt := &InboundEvent{
		ChannelType:      models.ChannelTypeEmail,
		OrgID:            org.ID,
		SenderIdentifier: "dave@example.com",
		Body:             "Test.",
	}
	require.NoError(t, gw.Process(context.Background(), evt))

	select {
	case busEvt := <-ch:
		assert.Equal(t, "channel.inbound", busEvt.Type)
		assert.Equal(t, "message", busEvt.EntityType)
	case <-time.After(2 * time.Second):
		t.Fatal("expected event on event bus but timed out")
	}
}

// --- Retry engine tests ---

func TestRetryEngine_SuccessOnFirstAttempt(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "retry-ok-org")
	repo := NewRepository(db)
	re := newRetryEngineWithSleep(repo, func(time.Duration) {}) // no-op sleep
	ctx := context.Background()

	attempts := 0
	fn := func(_ context.Context, _ *InboundEvent) error {
		attempts++
		return nil
	}

	evt := &InboundEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail}
	err := re.ProcessWithRetry(ctx, evt, fn)
	require.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetryEngine_SuccessOnSecondAttempt(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "retry-2nd-org")
	repo := NewRepository(db)
	re := newRetryEngineWithSleep(repo, func(time.Duration) {})
	ctx := context.Background()

	attempts := 0
	fn := func(_ context.Context, _ *InboundEvent) error {
		attempts++
		if attempts < 2 {
			return fmt.Errorf("transient error")
		}
		return nil
	}

	evt := &InboundEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail}
	err := re.ProcessWithRetry(ctx, evt, fn)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetryEngine_ExhaustRetries_InsertsDLQ(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "retry-exhaust-org")
	repo := NewRepository(db)
	re := newRetryEngineWithSleep(repo, func(time.Duration) {})
	ctx := context.Background()

	attempts := 0
	fn := func(_ context.Context, _ *InboundEvent) error {
		attempts++
		return fmt.Errorf("permanent error")
	}

	evt := &InboundEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail, Metadata: `{"test":true}`}
	err := re.ProcessWithRetry(ctx, evt, fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ")
	assert.Equal(t, MaxRetries, attempts)

	// Verify DLQ event was created.
	var dlqEvts []models.DeadLetterEvent
	require.NoError(t, db.Find(&dlqEvts).Error)
	assert.Len(t, dlqEvts, 1)
	assert.Equal(t, models.DLQStatusFailed, dlqEvts[0].Status)
	assert.Equal(t, MaxRetries, dlqEvts[0].Attempts)
	assert.Equal(t, "permanent error", dlqEvts[0].ErrorMessage)
}

func TestRetryEngine_ContextCancelled(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "retry-cancel-org")
	repo := NewRepository(db)

	// Sleep function that respects context cancellation.
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled.

	re := newRetryEngineWithSleep(repo, func(d time.Duration) { time.Sleep(d) })

	fn := func(_ context.Context, _ *InboundEvent) error {
		return fmt.Errorf("fail")
	}

	evt := &InboundEvent{OrgID: org.ID, ChannelType: models.ChannelTypeEmail}
	err := re.ProcessWithRetry(cancelledCtx, evt, fn)
	// Should return context.Canceled on the second attempt's select.
	assert.Error(t, err)
}

func TestComputeBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{1, 750 * time.Millisecond, 1250 * time.Millisecond},  // ~1s ±25%
		{2, 1500 * time.Millisecond, 2500 * time.Millisecond}, // ~2s ±25%
		{3, 3 * time.Second, 5 * time.Second},                 // ~4s ±25%
		{4, 6 * time.Second, 10 * time.Second},                // ~8s ±25%
		{5, 12 * time.Second, 20 * time.Second},               // ~16s (cap) ±25%
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			// Run multiple times to account for jitter randomness.
			for i := 0; i < 10; i++ {
				d := computeBackoff(tt.attempt)
				assert.GreaterOrEqual(t, d, tt.minDelay, "attempt %d delay too small: %v", tt.attempt, d)
				assert.LessOrEqual(t, d, tt.maxDelay, "attempt %d delay too large: %v", tt.attempt, d)
			}
		})
	}
}

// --- Config type unit tests ---

func TestEmailConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     EmailConfig
		wantErr bool
	}{
		{"valid", EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "u"}, false},
		{"missing host", EmailConfig{IMAPPort: 993, Username: "u"}, true},
		{"zero port", EmailConfig{IMAPHost: "mail.test", IMAPPort: 0, Username: "u"}, true},
		{"missing username", EmailConfig{IMAPHost: "mail.test", IMAPPort: 993}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVoiceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     VoiceConfig
		wantErr bool
	}{
		{"valid", VoiceConfig{LiveKitAPIKey: "k", LiveKitProjectURL: "wss://p"}, false},
		{"missing api key", VoiceConfig{LiveKitProjectURL: "wss://p"}, true},
		{"missing project url", VoiceConfig{LiveKitAPIKey: "k"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMaskSettingsSecrets_Email(t *testing.T) {
	settings := `{"imap_host":"mail.test","imap_port":993,"username":"u","password":"s3cr3t","oauth_refresh_token":"supersecretrefreshtoken"}`
	masked := MaskSettingsSecrets(models.ChannelTypeEmail, settings)
	assert.NotContains(t, masked, "s3cr3t")
	assert.NotContains(t, masked, "supersecretrefreshtoken")
	assert.Contains(t, masked, "[REDACTED]")
	assert.Contains(t, masked, "mail.test") // non-secret preserved
}

func TestMaskSettingsSecrets_Voice(t *testing.T) {
	settings := `{"livekit_api_key":"mykey","livekit_api_secret":"mysecret","livekit_project_url":"wss://p"}`
	masked := MaskSettingsSecrets(models.ChannelTypeVoice, settings)
	assert.NotContains(t, masked, "mysecret")
	assert.Contains(t, masked, "[REDACTED]")
	assert.Contains(t, masked, "mykey") // API key is not a secret
}

func TestMaskSettingsSecrets_Chat_NoSecrets(t *testing.T) {
	settings := `{"embed_key":"public-key","widget_theme":{"primary_color":"#3B82F6"}}`
	masked := MaskSettingsSecrets(models.ChannelTypeChat, settings)
	assert.Equal(t, settings, masked) // unchanged
}

func TestMaskSettingsSecrets_InvalidJSON(t *testing.T) {
	// Invalid JSON should return original.
	original := "not-valid-json"
	result := MaskSettingsSecrets(models.ChannelTypeEmail, original)
	assert.Equal(t, original, result)
}

func TestValidateSettings_EmptyIsValid(t *testing.T) {
	assert.NoError(t, ValidateSettings(models.ChannelTypeEmail, ""))
	assert.NoError(t, ValidateSettings(models.ChannelTypeEmail, "{}"))
}

func TestChannelTypeMessageTypeMapping(t *testing.T) {
	assert.Equal(t, models.MessageTypeEmail, channelTypeToMessageType(models.ChannelTypeEmail))
	assert.Equal(t, models.MessageTypeCallLog, channelTypeToMessageType(models.ChannelTypeVoice))
	assert.Equal(t, models.MessageTypeComment, channelTypeToMessageType(models.ChannelTypeChat))
	assert.Equal(t, models.MessageTypeSystem, channelTypeToMessageType("unknown"))
}

func TestComputeHealthStatus(t *testing.T) {
	assert.Equal(t, HealthStatusDown, computeHealthStatus(0, false))
	assert.Equal(t, HealthStatusHealthy, computeHealthStatus(0, true))
	assert.Equal(t, HealthStatusDegraded, computeHealthStatus(3, true))
	assert.Equal(t, HealthStatusDegraded, computeHealthStatus(5, true))
	assert.Equal(t, HealthStatusDown, computeHealthStatus(6, true))
}

func TestChannelType_IsValid(t *testing.T) {
	assert.True(t, models.ChannelTypeEmail.IsValid())
	assert.True(t, models.ChannelTypeVoice.IsValid())
	assert.True(t, models.ChannelTypeChat.IsValid())
	assert.False(t, models.ChannelType("sms").IsValid())
	assert.False(t, models.ChannelType("").IsValid())
}

func TestDLQStatus_IsValid(t *testing.T) {
	assert.True(t, models.DLQStatusFailed.IsValid())
	assert.True(t, models.DLQStatusRetrying.IsValid())
	assert.True(t, models.DLQStatusResolved.IsValid())
	assert.True(t, models.DLQStatusDismissed.IsValid())
	assert.False(t, models.DLQStatus("unknown").IsValid())
}

// --- Fuzz tests ---

// FuzzValidateSettings fuzzes the Settings JSON validation for each channel type.
func FuzzValidateSettings(f *testing.F) {
	// Seed corpus: valid and invalid JSON for each channel type.
	seeds := []string{
		"{}",
		`{"imap_host":"mail.test","imap_port":993,"username":"u"}`,
		`{"livekit_api_key":"k","livekit_project_url":"wss://p"}`,
		`{"embed_key":"abc"}`,
		"not-json",
		`{"imap_port":-1}`,
		`{"imap_host":""}`,
		`null`,
		`[]`,
		`{"imap_host":123}`,
		`{"livekit_api_key":null}`,
	}
	for _, s := range seeds {
		f.Add("email", s)
		f.Add("voice", s)
		f.Add("chat", s)
		f.Add("sms", s)
		f.Add("", s)
	}
	f.Fuzz(func(t *testing.T, channelTypeStr, settingsJSON string) {
		ct := models.ChannelType(channelTypeStr)
		// ValidateSettings must not panic regardless of input.
		_ = ValidateSettings(ct, settingsJSON)
		// MaskSettingsSecrets must not panic regardless of input.
		_ = MaskSettingsSecrets(ct, settingsJSON)
	})
}

// FuzzInboundEvent fuzzes InboundEvent processing via the Gateway.
func FuzzInboundEvent(f *testing.F) {
	// Seed corpus for fuzz targets.
	seeds := []struct {
		channelType string
		orgID       string
		externalID  string
		sender      string
		subject     string
		body        string
	}{
		{"email", "org-1", "msg-001", "alice@example.com", "Hello", "Body text"},
		{"voice", "org-1", "call-001", "+15555551234", "", "Transcript"},
		{"chat", "org-1", "sess-001", "anon-session", "", "Chat message"},
		{"", "org-1", "", "", "", ""},
		{"email", "", "msg-002", "bob@test.com", "", ""},
		{"sms", "org-1", "msg-003", "charlie@test.com", "Subj", "Body"},
		{"email", "org-1", "<script>", `" OR 1=1; --`, `'; DROP TABLE threads; --`, "injection"},
		{"email", "org-1", strings.Repeat("x", 1000), strings.Repeat("y", 1000), strings.Repeat("z", 1000), strings.Repeat("w", 1000)},
	}
	for _, s := range seeds {
		f.Add(s.channelType, s.orgID, s.externalID, s.sender, s.subject, s.body)
	}
	f.Fuzz(func(t *testing.T, channelTypeStr, orgID, externalID, sender, subject, body string) {
		// Gateway.Process must not panic regardless of input.
		// We use an in-memory DB and expect either success or a controlled error.
		db := setupTestDB(t)
		gw := NewGateway(db, nil)
		evt := &InboundEvent{
			ChannelType:      models.ChannelType(channelTypeStr),
			OrgID:            orgID,
			ExternalID:       externalID,
			SenderIdentifier: sender,
			Subject:          subject,
			Body:             body,
		}
		// Must not panic.
		_ = gw.Process(context.Background(), evt)
	})
}
