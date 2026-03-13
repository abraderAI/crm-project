package admin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Additional Fuzz Tests (user management) ---

func FuzzListUsersEmail(f *testing.F) {
	f.Add("test@example.com")
	f.Add("")
	f.Add("%")
	f.Add("'; DROP TABLE--")
	f.Add(strings.Repeat("a", 2000))
	f.Add("<script>alert(1)</script>")
	f.Add("user+tag@domain.co")
	f.Add("日本語@テスト.jp")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, email string) {
		params := UserListParams{Params: pagination.Params{Limit: 10}, Email: email}
		_, _, _ = svc.ListUsers(ctx, params)
	})
}

func FuzzListUsersName(f *testing.F) {
	f.Add("John Doe")
	f.Add("")
	f.Add("%")
	f.Add(strings.Repeat("x", 2000))
	f.Add("name<script>")
	f.Add("日本語名前")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, name string) {
		params := UserListParams{Params: pagination.Params{Limit: 10}, Name: name}
		_, _, _ = svc.ListUsers(ctx, params)
	})
}

func FuzzListUsersUserID(f *testing.F) {
	f.Add("user_123")
	f.Add("")
	f.Add("'; DROP TABLE--")
	f.Add(strings.Repeat("u", 2000))
	f.Add("日本語ユーザ")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		params := UserListParams{Params: pagination.Params{Limit: 10}, UserID: userID}
		_, _, _ = svc.ListUsers(ctx, params)
	})
}

func FuzzListUsersOrgSlug(f *testing.F) {
	f.Add("my-org")
	f.Add("")
	f.Add("'; DROP TABLE--")
	f.Add(strings.Repeat("o", 2000))
	f.Add("slug with spaces")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, slug string) {
		params := UserListParams{Params: pagination.Params{Limit: 10}, OrgSlug: slug}
		_, _, _ = svc.ListUsers(ctx, params)
	})
}

func FuzzGetUser(f *testing.F) {
	f.Add("user_abc")
	f.Add("")
	f.Add(strings.Repeat("x", 5000))
	f.Add("日本語")
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		_, _ = svc.GetUser(ctx, userID)
	})
}

func FuzzSyncUserShadow(f *testing.F) {
	f.Add("uid1", "user@test.com", "User Name")
	f.Add("", "", "")
	f.Add("uid", "bad-email", strings.Repeat("n", 2000))
	f.Add("uid", "<script>", "日本語名前")
	f.Add(strings.Repeat("x", 500), "x@x.x", "x")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID, email, displayName string) {
		svc.SyncUserShadow(ctx, userID, email, displayName)
	})
}

func FuzzIsUserBanned(f *testing.F) {
	f.Add("user_123")
	f.Add("")
	f.Add(strings.Repeat("x", 5000))
	f.Add("'; DROP TABLE--")
	f.Add("日本語")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		_, _ = svc.IsUserBanned(ctx, userID)
	})
}

func FuzzUnbanUser(f *testing.F) {
	f.Add("user_abc")
	f.Add("")
	f.Add(strings.Repeat("u", 2000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		_ = svc.UnbanUser(ctx, userID)
	})
}

// --- Additional Fuzz Tests (org management) ---

func FuzzListOrgsSlug(f *testing.F) {
	f.Add("my-org")
	f.Add("")
	f.Add("%")
	f.Add("'; DROP TABLE--")
	f.Add(strings.Repeat("s", 2000))
	f.Add("<img src=x>")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, slug string) {
		params := OrgListParams{Params: pagination.Params{Limit: 10}, Slug: slug}
		_, _, _ = svc.ListOrgs(ctx, params)
	})
}

func FuzzListOrgsName(f *testing.F) {
	f.Add("Test Org")
	f.Add("")
	f.Add("%")
	f.Add(strings.Repeat("n", 2000))
	f.Add("日本語組織")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, name string) {
		params := OrgListParams{Params: pagination.Params{Limit: 10}, Name: name}
		_, _, _ = svc.ListOrgs(ctx, params)
	})
}

func FuzzListOrgsBillingTier(f *testing.F) {
	f.Add("free")
	f.Add("pro")
	f.Add("enterprise")
	f.Add("")
	f.Add("invalid_tier")
	f.Add(strings.Repeat("t", 2000))

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, tier string) {
		params := OrgListParams{Params: pagination.Params{Limit: 10}, BillingTier: tier}
		_, _, _ = svc.ListOrgs(ctx, params)
	})
}

func FuzzGetOrgDetail(f *testing.F) {
	f.Add("my-org")
	f.Add("")
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("invalid-uuid")
	f.Add(strings.Repeat("x", 5000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.Space{}, &models.Board{}, &models.Thread{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, orgIDOrSlug string) {
		_, _ = svc.GetOrgDetail(ctx, orgIDOrSlug)
	})
}

func FuzzIsOrgSuspended(f *testing.F) {
	f.Add("test-org")
	f.Add("")
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add(strings.Repeat("x", 5000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, orgIDOrSlug string) {
		_, _ = svc.IsOrgSuspended(ctx, orgIDOrSlug)
	})
}

func FuzzUnsuspendOrg(f *testing.F) {
	f.Add("test-org")
	f.Add("")
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add(strings.Repeat("x", 5000))

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, orgIDOrSlug string) {
		_ = svc.UnsuspendOrg(ctx, orgIDOrSlug)
	})
}

func FuzzTransferOrgOwnership(f *testing.F) {
	f.Add("test-org", "new-owner")
	f.Add("", "")
	f.Add("550e8400-e29b-41d4-a716-446655440000", "user_123")
	f.Add(strings.Repeat("x", 2000), strings.Repeat("y", 2000))
	f.Add("'; DROP TABLE--", "<script>")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, orgIDOrSlug, newOwner string) {
		_ = svc.TransferOrgOwnership(ctx, orgIDOrSlug, newOwner)
	})
}

// --- Additional Fuzz Tests (platform admin) ---

func FuzzIsPlatformAdmin(f *testing.F) {
	f.Add("admin_user")
	f.Add("")
	f.Add(strings.Repeat("a", 5000))
	f.Add("'; DROP TABLE--")
	f.Add("日本語")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		_, _ = svc.IsPlatformAdmin(ctx, userID)
	})
}

func FuzzRemovePlatformAdmin(f *testing.F) {
	f.Add("admin_user")
	f.Add("")
	f.Add(strings.Repeat("a", 5000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	// Ensure at least 2 admins so removal doesn't fail on last-admin check.
	_ = db.Create(&models.PlatformAdmin{UserID: "keep1", GrantedBy: "bootstrap", IsActive: true}).Error
	_ = db.Create(&models.PlatformAdmin{UserID: "keep2", GrantedBy: "bootstrap", IsActive: true}).Error
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		_ = svc.RemovePlatformAdmin(ctx, userID)
	})
}

func FuzzBootstrapAdmin(f *testing.F) {
	f.Add("bootstrap_user")
	f.Add("")
	f.Add(strings.Repeat("b", 5000))
	f.Add("'; DROP TABLE--")
	f.Add("日本語管理者")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		_ = svc.BootstrapAdmin(ctx, userID)
		// Clean up for repeated runs.
		db.Where("user_id = ?", userID).Delete(&models.PlatformAdmin{})
	})
}

// --- Additional Fuzz Tests (audit log) ---

func FuzzListAuditLogsUserID(f *testing.F) {
	f.Add("user_abc")
	f.Add("")
	f.Add("'; DROP TABLE--")
	f.Add(strings.Repeat("u", 2000))
	f.Add("日本語")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		params := AuditListParams{Params: pagination.Params{Limit: 10}, UserID: userID}
		_, _, _ = svc.ListAuditLogs(ctx, params)
	})
}

func FuzzListAuditLogsAction(f *testing.F) {
	f.Add("ban")
	f.Add("create")
	f.Add("")
	f.Add("'; DROP TABLE--")
	f.Add(strings.Repeat("a", 2000))

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, action string) {
		params := AuditListParams{Params: pagination.Params{Limit: 10}, Action: action}
		_, _, _ = svc.ListAuditLogs(ctx, params)
	})
}

func FuzzListAuditLogsEntityType(f *testing.F) {
	f.Add("user")
	f.Add("org")
	f.Add("platform_admin")
	f.Add("")
	f.Add(strings.Repeat("e", 2000))

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, entityType string) {
		params := AuditListParams{Params: pagination.Params{Limit: 10}, EntityType: entityType}
		_, _, _ = svc.ListAuditLogs(ctx, params)
	})
}

func FuzzListAuditLogsIPAddress(f *testing.F) {
	f.Add("192.168.1.1")
	f.Add("")
	f.Add("::1")
	f.Add("invalid-ip")
	f.Add(strings.Repeat("1", 2000))

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, ip string) {
		params := AuditListParams{Params: pagination.Params{Limit: 10}, IPAddress: ip}
		_, _, _ = svc.ListAuditLogs(ctx, params)
	})
}

// --- Additional Fuzz Tests (settings & flags) ---

func FuzzGetSetting(f *testing.F) {
	f.Add("file_upload_limits")
	f.Add("")
	f.Add("unknown_key")
	f.Add(strings.Repeat("k", 2000))
	f.Add("'; DROP TABLE--")
	f.Add("日本語")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, key string) {
		_, _ = svc.GetSetting(ctx, key)
	})
}

func FuzzToggleFeatureFlag(f *testing.F) {
	f.Add("maintenance_mode", true)
	f.Add("", false)
	f.Add(strings.Repeat("f", 2000), true)
	f.Add("'; DROP TABLE--", false)
	f.Add("日本語フラグ", true)

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.SeedFeatureFlags(ctx)

	f.Fuzz(func(t *testing.T, key string, enabled bool) {
		_ = svc.ToggleFeatureFlag(ctx, key, enabled, nil)
	})
}

func FuzzIsFeatureEnabled(f *testing.F) {
	f.Add("maintenance_mode")
	f.Add("")
	f.Add("nonexistent_flag")
	f.Add(strings.Repeat("f", 2000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.SeedFeatureFlags(ctx)

	f.Fuzz(func(t *testing.T, key string) {
		_, _ = svc.IsFeatureEnabled(ctx, key)
	})
}

func FuzzUpdateSettingsKey(f *testing.F) {
	f.Add("notification_defaults")
	f.Add("")
	f.Add(strings.Repeat("k", 2000))
	f.Add("key with special <>&\"' chars")
	f.Add("日本語設定")
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, key string) {
		patch := map[string]json.RawMessage{
			key: json.RawMessage(`{"enabled":true}`),
		}
		_ = svc.UpdateSettings(ctx, patch, "fuzzer")
	})
}

// --- Additional Fuzz Tests (export) ---

func FuzzExportFormat(f *testing.F) {
	f.Add("csv")
	f.Add("json")
	f.Add("")
	f.Add("xml")
	f.Add("CSV")
	f.Add(strings.Repeat("f", 2000))
	f.Add("'; DROP TABLE--")
	f.Add("日本語")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AdminExport{}, &models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, format string) {
		_, _ = svc.CreateExport(ctx, "users", format, "{}", "admin1", nil)
	})
}

func FuzzExportRequestedBy(f *testing.F) {
	f.Add("admin1")
	f.Add("")
	f.Add(strings.Repeat("a", 2000))
	f.Add("'; DROP TABLE--")
	f.Add("日本語ユーザ")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AdminExport{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, requestedBy string) {
		_, _, _ = svc.ListExports(ctx, requestedBy, pagination.Params{Limit: 10})
	})
}

func FuzzGetExport(f *testing.F) {
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("")
	f.Add("invalid-id")
	f.Add(strings.Repeat("x", 5000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.AdminExport{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, id string) {
		_, _ = svc.GetExport(ctx, id)
	})
}

// --- Additional Fuzz Tests (LLM usage + login events) ---

func FuzzGetLLMUsage(f *testing.F) {
	f.Add(10)
	f.Add(0)
	f.Add(-1)
	f.Add(1000000)
	f.Add(1)

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.LLMUsageLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, limit int) {
		_, _ = svc.GetLLMUsage(ctx, limit)
	})
}

func FuzzGetRecentLogins(f *testing.F) {
	f.Add(10, "")
	f.Add(0, "")
	f.Add(-1, "cursor")
	f.Add(100, strings.Repeat("c", 2000))

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.LoginEvent{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, limit int, cursor string) {
		params := pagination.Params{Limit: limit, Cursor: cursor}
		_, _, _ = svc.GetRecentLogins(ctx, params)
	})
}

// --- Additional Fuzz Tests (RBAC) ---

func FuzzRBACOverrideJSON(f *testing.F) {
	f.Add(`{"roles":{"permissions":{"viewer":["read"]}}}`)
	f.Add(`{}`)
	f.Add(`{"defaults":{}}`)
	f.Add(`invalid json`)
	f.Add(strings.Repeat("{", 500))
	f.Add(`null`)
	f.Add(`{"roles":{"permissions":{"admin":["` + strings.Repeat("x", 1000) + `"]}}}`)

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, input string) {
		var override RBACOverride
		if err := json.Unmarshal([]byte(input), &override); err != nil {
			return
		}
		_ = svc.UpdateRBACOverride(ctx, override, "fuzzer")
	})
}

func FuzzListOrgsPaymentStatus(f *testing.F) {
	f.Add("active")
	f.Add("past_due")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("p", 2000))
	f.Add("'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, status string) {
		params := OrgListParams{Params: pagination.Params{Limit: 10}, PaymentStatus: status}
		_, _, _ = svc.ListOrgs(ctx, params)
	})
}

func FuzzSuspendOrgByUUID(f *testing.F) {
	f.Add("550e8400-e29b-41d4-a716-446655440000", "reason", "admin1")
	f.Add("", "", "")
	f.Add("invalid-uuid", strings.Repeat("r", 2000), "admin")
	f.Add("日本語", "<script>", "'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, orgID, reason, suspendedBy string) {
		_ = svc.SuspendOrg(ctx, orgID, reason, suspendedBy)
	})
}

func FuzzBanUserFull(f *testing.F) {
	f.Add("user1", "spam", "admin1")
	f.Add("", "", "")
	f.Add(strings.Repeat("u", 500), strings.Repeat("r", 2000), strings.Repeat("a", 500))
	f.Add("日本語", "<script>alert(1)</script>", "'; DROP TABLE--")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID, reason, bannedBy string) {
		_ = svc.BanUser(ctx, userID, reason, bannedBy)
		_ = svc.UnbanUser(ctx, userID)
	})
}

func FuzzImpersonationReason(f *testing.F) {
	f.Add("support ticket #1234")
	f.Add("")
	f.Add(strings.Repeat("r", 5000))
	f.Add("<script>alert(1)</script>")
	f.Add("reason with\nnewlines\tand\ttabs")
	f.Add("日本語の理由")

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	svc := NewService(db)
	svc.SyncUserShadow(context.Background(), "target", "t@t.com", "T")
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, reason string) {
		_, _, _ = svc.ImpersonateUser(ctx, "admin1", "target", reason, 30)
	})
}

func FuzzImpersonationDuration(f *testing.F) {
	f.Add(0)
	f.Add(-100)
	f.Add(30)
	f.Add(120)
	f.Add(999999)
	f.Add(1)

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	svc := NewService(db)
	svc.SyncUserShadow(context.Background(), "target", "t@t.com", "T")
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, duration int) {
		_, _, _ = svc.ImpersonateUser(ctx, "admin1", "target", "test", duration)
	})
}

func FuzzListWebhookDeliveries(f *testing.F) {
	f.Add(10, "")
	f.Add(0, "")
	f.Add(-1, "invalid-cursor")
	f.Add(100, strings.Repeat("c", 2000))

	db := setupFuzzDB(f)
	_ = db.AutoMigrate(&models.WebhookDelivery{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, limit int, cursor string) {
		params := pagination.Params{Limit: limit, Cursor: cursor}
		_, _, _ = svc.ListAllWebhookDeliveries(ctx, params)
	})
}
