package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_journal_mode=WAL"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.Org{},
		&models.Space{},
		&models.Board{},
		&models.Thread{},
		&models.Message{},
		&models.OrgMembership{},
		&models.SpaceMembership{},
		&models.BoardMembership{},
		&models.AuditLog{},
		&models.PlatformAdmin{},
		&models.UserShadow{},
		&models.Vote{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.DigestSchedule{},
		&models.Upload{},
		&models.CallLog{},
		&models.Revision{},
		&models.APIKey{},
		&models.WebhookSubscription{},
		&models.WebhookDelivery{},
		&models.SystemSetting{},
		&models.FeatureFlag{},
	))
	return db
}

// --- Service Tests ---

func TestService_IsPlatformAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Not an admin.
	isAdmin, err := svc.IsPlatformAdmin(ctx, "user1")
	require.NoError(t, err)
	assert.False(t, isAdmin)

	// Add admin.
	_, err = svc.AddPlatformAdmin(ctx, "user1", "bootstrap")
	require.NoError(t, err)

	isAdmin, err = svc.IsPlatformAdmin(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, isAdmin)
}

func TestService_AddPlatformAdmin_EmptyUserID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.AddPlatformAdmin(ctx, "", "granter")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestService_AddPlatformAdmin_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.AddPlatformAdmin(ctx, "user1", "bootstrap")
	require.NoError(t, err)

	_, err = svc.AddPlatformAdmin(ctx, "user1", "bootstrap")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestService_ListPlatformAdmins(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	_, _ = svc.AddPlatformAdmin(ctx, "admin2", "admin1")

	admins, err := svc.ListPlatformAdmins(ctx)
	require.NoError(t, err)
	assert.Len(t, admins, 2)
}

func TestService_RemovePlatformAdmin_CannotRemoveLast(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")

	err := svc.RemovePlatformAdmin(ctx, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove the last platform admin")
}

func TestService_RemovePlatformAdmin_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	_, _ = svc.AddPlatformAdmin(ctx, "admin2", "admin1")

	err := svc.RemovePlatformAdmin(ctx, "admin1")
	require.NoError(t, err)

	isAdmin, _ := svc.IsPlatformAdmin(ctx, "admin1")
	assert.False(t, isAdmin)
}

func TestService_RemovePlatformAdmin_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	_, _ = svc.AddPlatformAdmin(ctx, "admin2", "admin1")

	err := svc.RemovePlatformAdmin(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_BootstrapAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Empty user ID — no-op.
	err := svc.BootstrapAdmin(ctx, "")
	require.NoError(t, err)

	// Bootstrap first admin.
	err = svc.BootstrapAdmin(ctx, "bootstrap_user")
	require.NoError(t, err)

	isAdmin, _ := svc.IsPlatformAdmin(ctx, "bootstrap_user")
	assert.True(t, isAdmin)

	// Idempotent — calling again should not error.
	err = svc.BootstrapAdmin(ctx, "bootstrap_user")
	require.NoError(t, err)
}

func TestService_SyncUserShadow(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "user1@example.com", "User One")

	var shadow models.UserShadow
	require.NoError(t, db.Where("clerk_user_id = ?", "user1").First(&shadow).Error)
	assert.Equal(t, "user1@example.com", shadow.Email)
	assert.Equal(t, "User One", shadow.DisplayName)
	assert.False(t, shadow.IsBanned)

	// Update on re-sync.
	svc.SyncUserShadow(ctx, "user1", "new@example.com", "New Name")
	require.NoError(t, db.Where("clerk_user_id = ?", "user1").First(&shadow).Error)
	assert.Equal(t, "new@example.com", shadow.Email)
}

func TestService_BanUnban(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "user1@example.com", "User One")

	// Ban.
	err := svc.BanUser(ctx, "user1", "spam", "admin1")
	require.NoError(t, err)

	banned, err := svc.IsUserBanned(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, banned)

	// Unban.
	err = svc.UnbanUser(ctx, "user1")
	require.NoError(t, err)

	banned, err = svc.IsUserBanned(ctx, "user1")
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestService_BanUser_CreatesIfNotExists(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.BanUser(ctx, "new_user", "abuse", "admin1")
	require.NoError(t, err)

	banned, err := svc.IsUserBanned(ctx, "new_user")
	require.NoError(t, err)
	assert.True(t, banned)
}

func TestService_UnbanUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.UnbanUser(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_IsUserBanned_NoRecord(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	banned, err := svc.IsUserBanned(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestService_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")
	svc.SyncUserShadow(ctx, "user2", "bob@example.com", "Bob")
	svc.SyncUserShadow(ctx, "user3", "charlie@example.com", "Charlie")

	// No filters.
	users, pageInfo, err := svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
	})
	require.NoError(t, err)
	assert.Len(t, users, 3)
	assert.False(t, pageInfo.HasMore)

	// Filter by email.
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
		Email:  "alice",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice@example.com", users[0].Email)

	// Filter by name.
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
		Name:   "Bob",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)

	// Filter by is_banned.
	_ = svc.BanUser(ctx, "user2", "spam", "admin1")
	trueVal := true
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params:   pagination.Params{Limit: 50},
		IsBanned: &trueVal,
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "user2", users[0].ClerkUserID)

	// Filter by user_id prefix.
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
		UserID: "user1",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestService_ListUsers_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.SyncUserShadow(ctx, strings.Repeat("a", 10)+string(rune('0'+i)), "", "")
	}

	users, pageInfo, err := svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 3},
	})
	require.NoError(t, err)
	assert.Len(t, users, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)
}

func TestService_ListUsers_SeenDateRange(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "a@b.com", "A")

	now := time.Now()
	after := now.Add(-1 * time.Hour)
	before := now.Add(1 * time.Hour)

	users, _, err := svc.ListUsers(ctx, UserListParams{
		Params:    pagination.Params{Limit: 50},
		SeenAfter: &after,
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)

	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params:     pagination.Params{Limit: 50},
		SeenBefore: &before,
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestService_GetUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")

	detail, err := svc.GetUser(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "alice@example.com", detail.Email)
	assert.Empty(t, detail.Memberships)

	// Not found.
	detail, err = svc.GetUser(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, detail)
}

func TestService_GetUser_WithMemberships(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")
	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	m := models.OrgMembership{OrgID: org.ID, UserID: "user1", Role: models.RoleOwner}
	require.NoError(t, db.Create(&m).Error)

	detail, err := svc.GetUser(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Len(t, detail.Memberships, 1)
}

// --- Org Management Tests ---

func TestService_ListOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		org := models.Org{Name: "Org " + string(rune('A'+i)), Slug: "org-" + string(rune('a'+i)), Metadata: "{}"}
		require.NoError(t, db.Create(&org).Error)
	}

	orgs, pageInfo, err := svc.ListOrgs(ctx, OrgListParams{Params: pagination.Params{Limit: 50}})
	require.NoError(t, err)
	assert.Len(t, orgs, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListOrgs_Filters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.Org{Name: "Alpha", Slug: "alpha", Metadata: "{}"}).Error)
	require.NoError(t, db.Create(&models.Org{Name: "Beta", Slug: "beta", Metadata: "{}"}).Error)

	orgs, _, err := svc.ListOrgs(ctx, OrgListParams{
		Params: pagination.Params{Limit: 50},
		Slug:   "alpha",
	})
	require.NoError(t, err)
	assert.Len(t, orgs, 1)

	orgs, _, err = svc.ListOrgs(ctx, OrgListParams{
		Params: pagination.Params{Limit: 50},
		Name:   "Beta",
	})
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
}

func TestService_GetOrgDetail(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	detail, err := svc.GetOrgDetail(ctx, org.Slug)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "Test Org", detail.Name)

	// Not found.
	detail, err = svc.GetOrgDetail(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, detail)
}

func TestService_GetOrgDetail_ByID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	detail, err := svc.GetOrgDetail(ctx, org.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "Test Org", detail.Name)
}

func TestService_SuspendUnsuspendOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	// Suspend.
	err := svc.SuspendOrg(ctx, org.Slug, "violation", "admin1")
	require.NoError(t, err)

	suspended, err := svc.IsOrgSuspended(ctx, org.Slug)
	require.NoError(t, err)
	assert.True(t, suspended)

	// Unsuspend.
	err = svc.UnsuspendOrg(ctx, org.Slug)
	require.NoError(t, err)

	suspended, err = svc.IsOrgSuspended(ctx, org.Slug)
	require.NoError(t, err)
	assert.False(t, suspended)
}

func TestService_SuspendOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.SuspendOrg(ctx, "nonexistent", "reason", "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_UnsuspendOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.UnsuspendOrg(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_IsOrgSuspended_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	suspended, err := svc.IsOrgSuspended(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, suspended)
}

func TestService_TransferOrgOwnership(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Transfer Org", Slug: "transfer-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	// Create an owner.
	m := models.OrgMembership{OrgID: org.ID, UserID: "old_owner", Role: models.RoleOwner}
	require.NoError(t, db.Create(&m).Error)

	// Transfer ownership.
	err := svc.TransferOrgOwnership(ctx, org.Slug, "new_owner")
	require.NoError(t, err)

	// Verify old owner is demoted to admin.
	var oldMember models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", org.ID, "old_owner").First(&oldMember).Error)
	assert.Equal(t, models.RoleAdmin, oldMember.Role)

	// Verify new owner is owner.
	var newMember models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", org.ID, "new_owner").First(&newMember).Error)
	assert.Equal(t, models.RoleOwner, newMember.Role)
}

func TestService_TransferOrgOwnership_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.TransferOrgOwnership(ctx, "nonexistent", "user1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_TransferOrgOwnership_ExistingMember(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Org", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	// Create owner and an existing member.
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "owner", Role: models.RoleOwner}).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "member", Role: models.RoleViewer}).Error)

	err := svc.TransferOrgOwnership(ctx, org.Slug, "member")
	require.NoError(t, err)

	var m models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", org.ID, "member").First(&m).Error)
	assert.Equal(t, models.RoleOwner, m.Role)
}

// --- Audit Log Tests ---

func TestService_ListAuditLogs(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create some audit entries.
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&models.AuditLog{
			UserID:     "admin1",
			Action:     models.AuditActionCreate,
			EntityType: "user",
			EntityID:   "user" + string(rune('1'+i)),
		}).Error)
	}

	logs, pageInfo, err := svc.ListAuditLogs(ctx, AuditListParams{
		Params: pagination.Params{Limit: 50},
	})
	require.NoError(t, err)
	assert.Len(t, logs, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListAuditLogs_Filters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.AuditLog{
		UserID: "admin1", Action: "ban", EntityType: "user", EntityID: "u1",
	}).Error)
	require.NoError(t, db.Create(&models.AuditLog{
		UserID: "admin2", Action: "suspend", EntityType: "org", EntityID: "o1",
	}).Error)

	logs, _, err := svc.ListAuditLogs(ctx, AuditListParams{
		Params: pagination.Params{Limit: 50},
		Action: "ban",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	logs, _, err = svc.ListAuditLogs(ctx, AuditListParams{
		Params:     pagination.Params{Limit: 50},
		EntityType: "org",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	logs, _, err = svc.ListAuditLogs(ctx, AuditListParams{
		Params: pagination.Params{Limit: 50},
		UserID: "admin1",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

// --- Middleware Tests ---

func TestPlatformAdminOnly_NoAuth(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := PlatformAdminOnly(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlatformAdminOnly_NotAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := PlatformAdminOnly(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "regular_user", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPlatformAdminOnly_IsAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_, _ = svc.AddPlatformAdmin(context.Background(), "admin_user", "bootstrap")

	handler := PlatformAdminOnly(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin_user", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBanCheck_NotBanned(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := BanCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBanCheck_Banned(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.BanUser(context.Background(), "user1", "spam", "admin1")

	handler := BanCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestBanCheck_NoUserContext(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := BanCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code) // Passes through, auth middleware handles later.
}

func TestOrgSuspensionCheck_ReadMethod(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	org := models.Org{Name: "Test", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	_ = svc.SuspendOrg(context.Background(), org.Slug, "test", "admin1")

	// GET should be allowed even for suspended orgs.
	router := chi.NewRouter()
	router.Get("/orgs/{org}", func(w http.ResponseWriter, r *http.Request) {
		OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/orgs/"+org.Slug, nil)
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgSuspensionCheck_WriteMethodBlocked(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	org := models.Org{Name: "Test", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	_ = svc.SuspendOrg(context.Background(), org.Slug, "test", "admin1")

	router := chi.NewRouter()
	router.Post("/v1/orgs/{org}/spaces", func(w http.ResponseWriter, r *http.Request) {
		OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.Slug+"/spaces", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestOrgSuspensionCheck_AdminRouteAllowed(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	org := models.Org{Name: "Test", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	_ = svc.SuspendOrg(context.Background(), org.Slug, "test", "admin1")

	router := chi.NewRouter()
	router.Post("/v1/admin/orgs/{org}/unsuspend", func(w http.ResponseWriter, r *http.Request) {
		OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/"+org.Slug+"/unsuspend", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgSuspensionCheck_NoOrgParam(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/search", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserShadowSync_JWT(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := UserShadowSync(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "sync_user", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)

	// Give goroutine time to finish.
	time.Sleep(100 * time.Millisecond)

	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "sync_user").First(&shadow).Error
	require.NoError(t, err)
}

func TestUserShadowSync_APIKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := UserShadowSync(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "apikey_user", AuthMethod: auth.AuthMethodAPIKey})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)

	// API key auth should NOT trigger sync.
	time.Sleep(100 * time.Millisecond)
	var count int64
	db.Model(&models.UserShadow{}).Where("clerk_user_id = ?", "apikey_user").Count(&count)
	assert.Equal(t, int64(0), count)
}

// --- Handler Tests ---

func setupTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc := NewService(db)
	auditSvc := audit.NewService(db)
	gdprSvc := gdpr.NewService(db)
	h := NewHandler(svc, auditSvc, gdprSvc, nil)
	return h, db
}

func adminCtx() context.Context {
	return auth.SetUserContext(context.Background(), &auth.UserContext{
		UserID: "admin_user", AuthMethod: auth.AuthMethodJWT,
	})
}

func chiCtx(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestHandler_ListUsers(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u2", Email: "b@test.com", DisplayName: "B", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_ListUsers_WithFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "alice@test.com", DisplayName: "Alice", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users?email=alice&is_banned=false", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var body struct{ Data []json.RawMessage }
	_ = json.NewDecoder(w.Body).Decode(&body)
	assert.Len(t, body.Data, 1)
}

func TestHandler_GetUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "user@test.com", DisplayName: "User", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users/u1", nil)
	r = chiCtx(r, "user_id", "u1")
	h.GetUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user@test.com")
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users/nonexistent", nil)
	r = chiCtx(r, "user_id", "nonexistent")
	h.GetUser(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users/", nil)
	r = chiCtx(r, "user_id", "")
	h.GetUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BanUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "ban_u", Email: "ban@test.com", DisplayName: "Ban", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/ban_u/ban", strings.NewReader(`{"reason":"spam"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "ban_u")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.BanUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "banned")
}

func TestHandler_BanUser_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/ban", strings.NewReader("invalid"))
	r = chiCtx(r, "user_id", "u1")
	h.BanUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BanUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users//ban", strings.NewReader(`{}`))
	r = chiCtx(r, "user_id", "")
	h.BanUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UnbanUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "test@test.com", DisplayName: "T", IsBanned: true, LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/unban", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "u1")
	h.UnbanUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "unbanned")
}

func TestHandler_UnbanUser_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/nope/unban", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "nope")
	h.UnbanUser(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UnbanUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users//unban", strings.NewReader(`{}`))
	r = chiCtx(r, "user_id", "")
	h.UnbanUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "purge_u", Email: "p@test.com", DisplayName: "P", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/users/purge_u/purge", nil)
	r = chiCtx(r, "user_id", "purge_u")
	h.PurgeUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "purged")
}

func TestHandler_PurgeUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/users//purge", nil)
	r = chiCtx(r, "user_id", "")
	h.PurgeUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListOrgs(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Org A", Slug: "org-a", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs", nil)
	h.ListOrgs(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_GetOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Detail", Slug: "detail", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs/detail", nil)
	r = chiCtx(r, "org", "detail")
	h.GetOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Detail")
}

func TestHandler_GetOrg_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs/nope", nil)
	r = chiCtx(r, "org", "nope")
	h.GetOrg(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs/", nil)
	r = chiCtx(r, "org", "")
	h.GetOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SuspendOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Susp", Slug: "susp", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/susp/suspend", strings.NewReader(`{"reason":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "susp")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "suspended")
}

func TestHandler_SuspendOrg_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/nope/suspend", strings.NewReader(`{"reason":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "nope")
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SuspendOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs//suspend", strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SuspendOrg_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/s/suspend", strings.NewReader("invalid"))
	r = chiCtx(r, "org", "s")
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UnsuspendOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.Org{Name: "Susp", Slug: "susp", Metadata: "{}", SuspendedAt: &now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/susp/unsuspend", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "susp")
	h.UnsuspendOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "unsuspended")
}

func TestHandler_UnsuspendOrg_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/nope/unsuspend", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "nope")
	h.UnsuspendOrg(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UnsuspendOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs//unsuspend", strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.UnsuspendOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferOwnership(t *testing.T) {
	h, db := setupTestHandler(t)
	org := models.Org{Name: "Xfer", Slug: "xfer", Metadata: "{}"}
	db.Create(&org)
	db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "old", Role: models.RoleOwner})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/xfer/transfer-ownership",
		strings.NewReader(`{"new_owner_user_id":"new"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "xfer")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "transferred")
}

func TestHandler_TransferOwnership_EmptyNewOwner(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/x/transfer-ownership",
		strings.NewReader(`{"new_owner_user_id":""}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "x")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferOwnership_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs//transfer-ownership",
		strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferOwnership_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/x/transfer-ownership",
		strings.NewReader("invalid"))
	r = chiCtx(r, "org", "x")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	org := models.Org{Name: "POrg", Slug: "porg", Metadata: "{}"}
	db.Create(&org)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/"+org.ID+"/purge",
		strings.NewReader(`{"confirm":"purge `+org.ID+`"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", org.ID)
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "purged")
}

func TestHandler_PurgeOrg_BadConfirm(t *testing.T) {
	h, db := setupTestHandler(t)
	org := models.Org{Name: "POrg", Slug: "porg2", Metadata: "{}"}
	db.Create(&org)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/"+org.ID+"/purge",
		strings.NewReader(`{"confirm":"wrong"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", org.ID)
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs//purge", strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/x/purge", strings.NewReader("invalid"))
	r = chiCtx(r, "org", "x")
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAuditLog(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.AuditLog{UserID: "admin", Action: "ban", EntityType: "user", EntityID: "u1"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-log", nil)
	h.ListAuditLog(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_ListPlatformAdmins(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "bootstrap", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/platform-admins", nil)
	h.ListPlatformAdmins(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_AddPlatformAdmin(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":"new_admin"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddPlatformAdmin_EmptyUserID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":""}`))
	r.Header.Set("Content-Type", "application/json")
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddPlatformAdmin_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins", strings.NewReader("invalid"))
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddPlatformAdmin_Duplicate(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "dup", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":"dup"}`))
	r.Header.Set("Content-Type", "application/json")
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_RemovePlatformAdmin(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "test", IsActive: true})
	db.Create(&models.PlatformAdmin{UserID: "a2", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/a1", nil)
	r = chiCtx(r, "user_id", "a1")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_RemovePlatformAdmin_LastAdmin(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/a1", nil)
	r = chiCtx(r, "user_id", "a1")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemovePlatformAdmin_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "test", IsActive: true})
	db.Create(&models.PlatformAdmin{UserID: "a2", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/nope", nil)
	r = chiCtx(r, "user_id", "nope")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RemovePlatformAdmin_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/", nil)
	r = chiCtx(r, "user_id", "")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Fuzz Tests ---

func FuzzBanUserReason(f *testing.F) {
	f.Add("spam")
	f.Add("")
	f.Add("a very long reason " + strings.Repeat("x", 1000))
	f.Add("<script>alert('xss')</script>")
	f.Add("reason with\nnewlines\tand\ttabs")
	f.Add("emoji 🚫 reason")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.UserShadow{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, reason string) {
		// Should not panic for any input.
		_ = svc.BanUser(ctx, "fuzz_user", reason, "admin")
		_ = svc.UnbanUser(ctx, "fuzz_user")
	})
}

func FuzzSuspendOrgReason(f *testing.F) {
	f.Add("violation")
	f.Add("")
	f.Add(strings.Repeat("a", 2000))
	f.Add("reason with special chars: <>&\"'")
	f.Add("unicode: 日本語テスト")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.Org{})
	svc := NewService(db)
	ctx := context.Background()

	// Create a test org.
	_ = db.Create(&models.Org{Name: "Fuzz Org", Slug: "fuzz-org", Metadata: "{}"}).Error

	f.Fuzz(func(t *testing.T, reason string) {
		_ = svc.SuspendOrg(ctx, "fuzz-org", reason, "admin")
		_ = svc.UnsuspendOrg(ctx, "fuzz-org")
	})
}

func FuzzAddPlatformAdmin(f *testing.F) {
	f.Add("user_123")
	f.Add("")
	f.Add(strings.Repeat("a", 500))
	f.Add("user with spaces")
	f.Add("user<>special")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		// Should not panic.
		_, _ = svc.AddPlatformAdmin(ctx, userID, "fuzzer")
		// Clean up to allow repeated adds.
		db.Where("user_id = ?", userID).Delete(&models.PlatformAdmin{})
	})
}
