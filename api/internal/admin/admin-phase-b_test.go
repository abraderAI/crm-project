package admin

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Test Helper ---

func testRBACPolicy(t *testing.T) *config.RBACPolicy {
	t.Helper()
	yamlData := `
resolution:
  strategy: "explicit_override_with_parent_fallback"
  order:
    - board
    - space
    - org
roles:
  hierarchy:
    - viewer
    - commenter
    - contributor
    - moderator
    - admin
    - owner
  permissions:
    viewer:
      - read
    commenter:
      - read
      - comment
    contributor:
      - read
      - comment
      - create
      - update_own
    moderator:
      - read
      - comment
      - create
      - update_own
      - update_any
      - moderate
    admin:
      - read
      - comment
      - create
      - update_own
      - update_any
      - moderate
      - manage_members
      - manage_settings
    owner:
      - read
      - comment
      - create
      - update_own
      - update_any
      - moderate
      - manage_members
      - manage_settings
      - delete_entity
defaults:
  org_member_role: "viewer"
  space_member_role: "viewer"
  board_member_role: "viewer"
`
	policy, err := config.ParseRBACPolicy([]byte(yamlData))
	require.NoError(t, err)
	return policy
}

func setupTestHandlerWithPolicy(t *testing.T) (*Handler, *Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc := NewService(db)
	auditSvc := audit.NewService(db)
	gdprSvc := gdpr.NewService(db)
	policy := testRBACPolicy(t)
	h := NewHandler(svc, auditSvc, gdprSvc, policy)
	return h, svc, db
}

// --- System Settings Service Tests ---

func TestService_GetAllSettings_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	settings, err := svc.GetAllSettings(context.Background())
	require.NoError(t, err)
	assert.Empty(t, settings)
}

func TestService_UpdateSettings_CreateNew(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	patch := map[string]json.RawMessage{
		"file_upload_limits": json.RawMessage(`{"max_size":10485760,"allowed_types":["image/png","image/jpeg"]}`),
	}
	err := svc.UpdateSettings(ctx, patch, "admin1")
	require.NoError(t, err)

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	assert.Contains(t, settings, "file_upload_limits")
	assert.Contains(t, string(settings["file_upload_limits"]), "10485760")
}

func TestService_UpdateSettings_DeepMerge(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create initial setting.
	patch1 := map[string]json.RawMessage{
		"webhook_retry_policy": json.RawMessage(`{"max_attempts":3,"backoff_multiplier":2.0}`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch1, "admin1"))

	// Deep-merge update.
	patch2 := map[string]json.RawMessage{
		"webhook_retry_policy": json.RawMessage(`{"max_attempts":5}`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch2, "admin1"))

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(settings["webhook_retry_policy"], &result))
	assert.Equal(t, float64(5), result["max_attempts"])
	assert.Equal(t, float64(2.0), result["backoff_multiplier"]) // Preserved from original.
}

func TestService_UpdateSettings_UnknownKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	patch := map[string]json.RawMessage{
		"unknown_key": json.RawMessage(`{}`),
	}
	err := svc.UpdateSettings(context.Background(), patch, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown setting key")
}

func TestService_UpdateSettings_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	patch := map[string]json.RawMessage{
		"file_upload_limits": json.RawMessage(`{invalid`),
	}
	err := svc.UpdateSettings(context.Background(), patch, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestService_UpdateSettings_MultipleKeys(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	patch := map[string]json.RawMessage{
		"file_upload_limits":      json.RawMessage(`{"max_size":1024}`),
		"default_pipeline_stages": json.RawMessage(`["new_lead","contacted","qualified"]`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch, "admin1"))

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	assert.Len(t, settings, 2)
}

func TestService_GetSetting(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Not found.
	setting, err := svc.GetSetting(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, setting)

	// Create and retrieve.
	patch := map[string]json.RawMessage{
		"llm_rate_limits": json.RawMessage(`{"requests_per_minute":60}`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch, "admin1"))

	setting, err = svc.GetSetting(ctx, "llm_rate_limits")
	require.NoError(t, err)
	require.NotNil(t, setting)
	assert.Equal(t, "admin1", setting.UpdatedBy)
}

// --- System Settings Handler Tests ---

func TestHandler_GetSettings(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/settings", nil)
	h.GetSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PatchSettings(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"file_upload_limits":{"max_size":5242880}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "5242880")
}

func TestHandler_PatchSettings_InvalidBody(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader("invalid"))
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchSettings_EmptyPatch(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchSettings_UnknownKey(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"bad_key":{"value":1}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- RBAC Override Service Tests ---

func TestService_GetEffectivePolicy_NoOverrides(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	policy := testRBACPolicy(t)

	effective, err := svc.GetEffectivePolicy(context.Background(), policy)
	require.NoError(t, err)
	require.NotNil(t, effective)
	assert.Equal(t, "viewer", effective.Defaults.OrgMemberRole)
	assert.Contains(t, effective.Roles.Permissions["owner"], "manage_settings")
}

func TestService_GetEffectivePolicy_NilPolicy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.GetEffectivePolicy(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestService_UpdateRBACOverride_Valid(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	policy := testRBACPolicy(t)

	override := RBACOverride{
		Roles: &RBACRolesOverride{
			Permissions: map[string][]string{
				"viewer": {"read", "comment"},
			},
		},
		Defaults: &RBACDefaultsOverride{
			OrgMemberRole: "commenter",
		},
	}
	require.NoError(t, svc.UpdateRBACOverride(ctx, override, "admin1"))

	effective, err := svc.GetEffectivePolicy(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, "commenter", effective.Defaults.OrgMemberRole)
	assert.Equal(t, []string{"read", "comment"}, effective.Roles.Permissions["viewer"])
}

func TestService_UpdateRBACOverride_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	override := RBACOverride{
		Roles: &RBACRolesOverride{
			Permissions: map[string][]string{
				"superadmin": {"everything"},
			},
		},
	}
	err := svc.UpdateRBACOverride(context.Background(), override, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown role")
}

func TestService_UpdateRBACOverride_InvalidDefaultRole(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	override := RBACOverride{
		Defaults: &RBACDefaultsOverride{
			OrgMemberRole: "nonexistent",
		},
	}
	err := svc.UpdateRBACOverride(context.Background(), override, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown role in defaults")
}

func TestService_UpdateRBACOverride_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	override := RBACOverride{
		Defaults: &RBACDefaultsOverride{
			OrgMemberRole: "contributor",
		},
	}
	require.NoError(t, svc.UpdateRBACOverride(ctx, override, "admin1"))
	require.NoError(t, svc.UpdateRBACOverride(ctx, override, "admin1"))

	// Should only have one setting.
	var count int64
	db.Model(&models.SystemSetting{}).Where("key = ?", rbacOverrideKey).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestService_GetRBACOverride_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	override, err := svc.GetRBACOverride(context.Background())
	require.NoError(t, err)
	assert.Nil(t, override)
}

func TestService_PreviewRBACRole(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	policy := testRBACPolicy(t)

	// Create org and membership.
	org := models.Org{Name: "Preview Org", Slug: "preview-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: "preview_user", Role: models.RoleAdmin,
	}).Error)

	role, permissions, err := svc.PreviewRBACRole(ctx, policy, "preview_user", "org", org.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "admin", role)
	assert.Contains(t, permissions, "manage_settings")
}

func TestService_PreviewRBACRole_WithOverride(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	policy := testRBACPolicy(t)

	org := models.Org{Name: "Preview Org", Slug: "preview-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: "preview_user", Role: models.RoleAdmin,
	}).Error)

	override := &RBACOverride{
		Roles: &RBACRolesOverride{
			Permissions: map[string][]string{
				"admin": {"read", "write", "custom_perm"},
			},
		},
	}

	role, permissions, err := svc.PreviewRBACRole(ctx, policy, "preview_user", "org", org.ID, override)
	require.NoError(t, err)
	assert.Equal(t, "admin", role)
	assert.Contains(t, permissions, "custom_perm")
}

func TestService_PreviewRBACRole_NoMembership(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	policy := testRBACPolicy(t)

	org := models.Org{Name: "Empty Org", Slug: "empty-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	role, permissions, err := svc.PreviewRBACRole(ctx, policy, "nonmember", "org", org.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "", role)
	assert.Nil(t, permissions)
}

// --- RBAC Handler Tests ---

func TestHandler_GetRBACPolicy(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/rbac-policy", nil)
	h.GetRBACPolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "hierarchy")
	assert.Contains(t, w.Body.String(), "permissions")
}

func TestHandler_PatchRBACPolicy(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"defaults":{"org_member_role":"contributor"}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/rbac-policy", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.PatchRBACPolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "contributor")
}

func TestHandler_PatchRBACPolicy_InvalidBody(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/rbac-policy", strings.NewReader("invalid"))
	h.PatchRBACPolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchRBACPolicy_InvalidRole(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"roles":{"permissions":{"megaadmin":["everything"]}}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/rbac-policy", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PatchRBACPolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PreviewRBACPolicy(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)

	org := models.Org{Name: "Prev", Slug: "prev", Metadata: "{}"}
	db.Create(&org)
	db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "u1", Role: models.RoleViewer})

	body := `{"user_id":"u1","entity_type":"org","entity_id":"` + org.ID + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/rbac-policy/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PreviewRBACPolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "viewer")
}

func TestHandler_PreviewRBACPolicy_MissingFields(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"user_id":"u1"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/rbac-policy/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PreviewRBACPolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PreviewRBACPolicy_InvalidBody(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/rbac-policy/preview", strings.NewReader("invalid"))
	h.PreviewRBACPolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Feature Flags Service Tests ---

func TestService_SeedFeatureFlags(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	flags, err := svc.ListFeatureFlags(ctx)
	require.NoError(t, err)
	assert.Len(t, flags, 3)

	// Idempotent.
	require.NoError(t, svc.SeedFeatureFlags(ctx))
	flags, _ = svc.ListFeatureFlags(ctx)
	assert.Len(t, flags, 3)
}

func TestService_ListFeatureFlags(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	flags, err := svc.ListFeatureFlags(ctx)
	require.NoError(t, err)
	assert.Len(t, flags, 3)
	// Should be sorted by key.
	assert.Equal(t, "community_voting", flags[0].Key)
}

func TestService_GetFeatureFlag(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	flag, err := svc.GetFeatureFlag(ctx, "maintenance_mode")
	require.NoError(t, err)
	require.NotNil(t, flag)
	assert.False(t, flag.Enabled)

	// Not found.
	flag, err = svc.GetFeatureFlag(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, flag)
}

func TestService_ToggleFeatureFlag(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	// Enable maintenance_mode.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", true, nil))

	flag, _ := svc.GetFeatureFlag(ctx, "maintenance_mode")
	assert.True(t, flag.Enabled)

	// Disable.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", false, nil))
	flag, _ = svc.GetFeatureFlag(ctx, "maintenance_mode")
	assert.False(t, flag.Enabled)
}

func TestService_ToggleFeatureFlag_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	err := svc.ToggleFeatureFlag(context.Background(), "nonexistent", true, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_ToggleFeatureFlag_WithOrgScope(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	orgScope := "org-123"
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "community_voting", true, &orgScope))

	flag, _ := svc.GetFeatureFlag(ctx, "community_voting")
	require.NotNil(t, flag.OrgScope)
	assert.Equal(t, "org-123", *flag.OrgScope)

	// Clear org scope.
	emptyScope := ""
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "community_voting", true, &emptyScope))
	flag, _ = svc.GetFeatureFlag(ctx, "community_voting")
	assert.Nil(t, flag.OrgScope)
}

func TestService_IsFeatureEnabled(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	enabled, err := svc.IsFeatureEnabled(ctx, "community_voting")
	require.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = svc.IsFeatureEnabled(ctx, "maintenance_mode")
	require.NoError(t, err)
	assert.False(t, enabled)

	// Nonexistent flag.
	enabled, err = svc.IsFeatureEnabled(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, enabled)
}

// --- Feature Flags Handler Tests ---

func TestHandler_ListFeatureFlags(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/feature-flags", nil)
	h.ListFeatureFlags(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "community_voting")
	assert.Contains(t, w.Body.String(), "maintenance_mode")
}

func TestHandler_PatchFeatureFlag(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"enabled":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/maintenance_mode", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "maintenance_mode")
	r = r.WithContext(context.WithValue(adminCtx(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":true`)
}

func TestHandler_PatchFeatureFlag_NotFound(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"enabled":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/nonexistent", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_PatchFeatureFlag_MissingEnabled(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"org_scope":"org1"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/maintenance_mode", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "maintenance_mode")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchFeatureFlag_InvalidBody(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/x", strings.NewReader("invalid"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "x")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchFeatureFlag_EmptyKey(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/", strings.NewReader(`{"enabled":true}`))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Maintenance Mode Middleware Tests ---

func TestMaintenanceMode_Disabled(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_Enabled_BlocksPost(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "maintenance mode")
}

func TestMaintenanceMode_Enabled_AllowsGet(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/orgs", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_Enabled_AllowsAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/settings", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_NoFlagTable(t *testing.T) {
	// When flag not found, should fail open.
	db := setupTestDB(t)
	svc := NewService(db)
	// Don't seed flags — maintenance_mode doesn't exist.

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Monitoring Stats Tests ---

func TestService_GetPlatformStats(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create test data.
	now := time.Now()
	db.Create(&models.Org{Name: "Org1", Slug: "org1", Metadata: "{}"})
	db.Create(&models.Org{Name: "Org2", Slug: "org2", Metadata: "{}"})
	db.Create(&models.UserShadow{ClerkUserID: "u1", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u2", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u3", LastSeenAt: now, SyncedAt: now})

	stats, err := svc.GetPlatformStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Orgs.Total)
	assert.Equal(t, int64(3), stats.Users.Total)
	assert.True(t, stats.DBSizeBytes > 0)
}

func TestService_GetPlatformStats_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	stats, err := svc.GetPlatformStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.Orgs.Total)
	assert.Equal(t, int64(0), stats.Users.Total)
	assert.True(t, stats.DBSizeBytes > 0) // DB always has some pages.
}

func TestHandler_GetStats(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	db.Create(&models.Org{Name: "StatOrg", Slug: "stat-org", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/stats", nil)
	h.GetStats(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "orgs")
	assert.Contains(t, w.Body.String(), "db_size_bytes")
}

// --- Webhook Deliveries Tests ---

func TestService_ListAllWebhookDeliveries_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	deliveries, pageInfo, err := svc.ListAllWebhookDeliveries(context.Background(), pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Empty(t, deliveries)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListAllWebhookDeliveries_WithData(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create org and subscription (for FK).
	org := models.Org{Name: "WH Org", Slug: "wh-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	sub := models.WebhookSubscription{
		OrgID: org.ID, URL: "http://example.com", Secret: "sec",
		ScopeType: "org", ScopeID: org.ID, EventFilter: "[]",
	}
	require.NoError(t, db.Create(&sub).Error)

	for i := 0; i < 3; i++ {
		db.Create(&models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventType:      "test.event",
			Payload:        "{}",
			StatusCode:     200,
		})
	}

	deliveries, pageInfo, err := svc.ListAllWebhookDeliveries(ctx, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, deliveries, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestHandler_ListWebhookDeliveries(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/webhooks/deliveries", nil)
	h.ListWebhookDeliveries(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

// --- Integration Health Tests ---

func TestService_GetIntegrationStatus_Unconfigured(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Ensure env vars are not set.
	t.Setenv("CLERK_SECRET_KEY", "")
	t.Setenv("RESEND_API_KEY", "")
	t.Setenv("FLEXPOINT_API_KEY", "")

	status := svc.GetIntegrationStatus(context.Background())
	assert.Equal(t, "unconfigured", status.Clerk)
	assert.Equal(t, "unconfigured", status.Resend)
	assert.Equal(t, "unconfigured", status.FlexPoint)
}

func TestService_GetIntegrationStatus_Configured(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	t.Setenv("CLERK_SECRET_KEY", "test_key")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("FLEXPOINT_API_KEY", "fp_test")

	status := svc.GetIntegrationStatus(context.Background())
	assert.Equal(t, "ok", status.Clerk)
	assert.Equal(t, "ok", status.Resend)
	assert.Equal(t, "ok", status.FlexPoint)
}

func TestHandler_GetIntegrationHealth(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/integrations/status", nil)
	h.GetIntegrationHealth(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "clerk")
	assert.Contains(t, w.Body.String(), "resend")
	assert.Contains(t, w.Body.String(), "flexpoint")
}

// --- isValidRole Tests ---

func TestIsValidRole(t *testing.T) {
	assert.True(t, isValidRole("viewer"))
	assert.True(t, isValidRole("commenter"))
	assert.True(t, isValidRole("contributor"))
	assert.True(t, isValidRole("moderator"))
	assert.True(t, isValidRole("admin"))
	assert.True(t, isValidRole("owner"))
	assert.False(t, isValidRole("superadmin"))
	assert.False(t, isValidRole(""))
	assert.False(t, isValidRole("nonexistent"))
}

// --- isValidationErr Tests ---

func TestIsValidationErr(t *testing.T) {
	assert.True(t, isValidationErr(fmt.Errorf("unknown setting key: foo")))
	assert.True(t, isValidationErr(fmt.Errorf("invalid JSON value")))
	assert.False(t, isValidationErr(fmt.Errorf("database error")))
}

// --- MarshalRBACPolicyToYAML Tests ---

func TestMarshalRBACPolicyToYAML(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	policy := testRBACPolicy(t)

	effective, err := svc.GetEffectivePolicy(context.Background(), policy)
	require.NoError(t, err)

	yamlStr, err := MarshalRBACPolicyToYAML(effective)
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "hierarchy")
	assert.Contains(t, yamlStr, "viewer")
}

// --- End-to-End Feature Flag Toggle + Maintenance Mode ---

func TestFeatureFlagToggle_MaintenanceMode_E2E(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	// Initially disabled.
	enabled, _ := svc.IsFeatureEnabled(ctx, "maintenance_mode")
	assert.False(t, enabled)

	// Enable maintenance mode.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", true, nil))

	// Verify non-GET blocked.
	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Disable maintenance mode.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", false, nil))

	// Verify requests work again.
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Additional Coverage Tests ---

func TestService_ListAllWebhookDeliveries_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Pag Org", Slug: "pag-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	sub := models.WebhookSubscription{
		OrgID: org.ID, URL: "http://example.com", Secret: "sec",
		ScopeType: "org", ScopeID: org.ID, EventFilter: "[]",
	}
	require.NoError(t, db.Create(&sub).Error)

	for i := 0; i < 5; i++ {
		db.Create(&models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventType:      "test.event",
			Payload:        "{}",
			StatusCode:     200,
		})
	}

	// First page.
	deliveries, pageInfo, err := svc.ListAllWebhookDeliveries(ctx, pagination.Params{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, deliveries, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)

	// Second page.
	deliveries2, pageInfo2, err := svc.ListAllWebhookDeliveries(ctx, pagination.Params{Limit: 3, Cursor: pageInfo.NextCursor})
	require.NoError(t, err)
	assert.Len(t, deliveries2, 2)
	assert.False(t, pageInfo2.HasMore)
}

func TestHandler_GetSettings_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/settings", nil)
	h.GetSettings(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PatchSettings_ServiceError(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)

	// Create a valid setting, then close DB.
	body := `{"file_upload_limits":{"max_size":1}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r2.Header.Set("Content-Type", "application/json")
	r2 = r2.WithContext(adminCtx())
	h.PatchSettings(w2, r2)
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestHandler_GetRBACPolicy_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/rbac-policy", nil)
	h.GetRBACPolicy(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PatchRBACPolicy_ServiceError(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	body := `{"defaults":{"org_member_role":"admin"}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/rbac-policy", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PatchRBACPolicy(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PreviewRBACPolicy_ServiceError(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	body := `{"user_id":"u1","entity_type":"org","entity_id":"some-id"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/rbac-policy/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PreviewRBACPolicy(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ListFeatureFlags_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/feature-flags", nil)
	h.ListFeatureFlags(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PatchFeatureFlag_ServiceError(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	body := `{"enabled":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/maintenance_mode", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "maintenance_mode")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetStats_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/stats", nil)
	h.GetStats(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ListWebhookDeliveries_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/webhooks/deliveries", nil)
	h.ListWebhookDeliveries(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMaintenanceMode_Enabled_BlocksPut(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/v1/orgs/test", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMaintenanceMode_Enabled_BlocksDelete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/orgs/test", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMaintenanceMode_Enabled_AllowsHead(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/v1/orgs", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestService_PreviewRBACRole_SpaceEntity(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	policy := testRBACPolicy(t)

	org := models.Org{Name: "Sp Org", Slug: "sp-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	space := models.Space{OrgID: org.ID, Name: "Space1", Slug: "space1"}
	require.NoError(t, db.Create(&space).Error)
	db.Create(&models.SpaceMembership{SpaceID: space.ID, UserID: "sp_user", Role: models.RoleContributor})

	role, permissions, err := svc.PreviewRBACRole(ctx, policy, "sp_user", "space", space.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "contributor", role)
	assert.Contains(t, permissions, "create")
}

func TestService_GetRBACOverride_AfterUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	override := RBACOverride{
		Defaults: &RBACDefaultsOverride{
			OrgMemberRole:   "contributor",
			SpaceMemberRole: "commenter",
		},
	}
	require.NoError(t, svc.UpdateRBACOverride(ctx, override, "admin1"))

	result, err := svc.GetRBACOverride(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "contributor", result.Defaults.OrgMemberRole)
	assert.Equal(t, "commenter", result.Defaults.SpaceMemberRole)
}

func TestService_UpdateRBACOverride_EmptyRolesOverride(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	override := RBACOverride{}
	err := svc.UpdateRBACOverride(context.Background(), override, "admin1")
	require.NoError(t, err)
}

func TestService_UpdateSettings_AllValidKeys(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for key := range KnownSettingKeys {
		patch := map[string]json.RawMessage{key: json.RawMessage(`{"test":true}`)}
		require.NoError(t, svc.UpdateSettings(ctx, patch, "admin1"), "failed for key: %s", key)
	}

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	assert.Len(t, settings, len(KnownSettingKeys))
}

func TestService_GetEffectivePolicy_AllOverrides(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	policy := testRBACPolicy(t)

	override := RBACOverride{
		Roles: &RBACRolesOverride{
			Permissions: map[string][]string{
				"viewer":    {"read", "comment"},
				"commenter": {"read", "comment", "react"},
			},
		},
		Defaults: &RBACDefaultsOverride{
			OrgMemberRole:   "commenter",
			SpaceMemberRole: "contributor",
			BoardMemberRole: "commenter",
		},
	}
	require.NoError(t, svc.UpdateRBACOverride(ctx, override, "admin1"))

	effective, err := svc.GetEffectivePolicy(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, "commenter", effective.Defaults.OrgMemberRole)
	assert.Equal(t, "contributor", effective.Defaults.SpaceMemberRole)
	assert.Equal(t, "commenter", effective.Defaults.BoardMemberRole)
	assert.Equal(t, []string{"read", "comment"}, effective.Roles.Permissions["viewer"])
	assert.NotNil(t, effective.Overrides)
}

func TestHandler_PatchSettings_NoAuthContext(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"file_upload_limits":{"max_size":1}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	// No adminCtx — updatedBy will be empty.
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PatchRBACPolicy_NoAuthContext(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"defaults":{"org_member_role":"contributor"}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/rbac-policy", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PatchRBACPolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PatchFeatureFlag_WithOrgScope(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"enabled":true,"org_scope":"org-x"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/community_voting", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "community_voting")
	r = r.WithContext(context.WithValue(adminCtx(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "org-x")
}

func TestHandler_ListUsers_DateFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users?seen_after="+now.Add(-time.Hour).Format(time.RFC3339)+"&seen_before="+now.Add(time.Hour).Format(time.RFC3339)+"&is_banned=true", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListOrgs_DateFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Org1", Slug: "org1", Metadata: "{}"})

	w := httptest.NewRecorder()
	now := time.Now()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs?created_after="+now.Add(-24*time.Hour).Format(time.RFC3339)+"&created_before="+now.Add(time.Hour).Format(time.RFC3339), nil)
	h.ListOrgs(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListAuditLog_DateFilters(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	now := time.Now()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-log?after="+now.Add(-24*time.Hour).Format(time.RFC3339)+"&before="+now.Add(time.Hour).Format(time.RFC3339)+"&action=create&entity_type=org&user=u1&ip=127.0.0.1", nil)
	h.ListAuditLog(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListPlatformAdmins_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/platform-admins", nil)
	h.ListPlatformAdmins(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ListUsers_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ListOrgs_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs", nil)
	h.ListOrgs(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ListAuditLog_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-log", nil)
	h.ListAuditLog(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestService_GetPlatformStats_WithWebhooksAndNotifications(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create org + subscription + deliveries.
	org := models.Org{Name: "SO", Slug: "so", Metadata: "{}"}
	db.Create(&org)
	sub := models.WebhookSubscription{
		OrgID: org.ID, URL: "http://example.com", Secret: "sec",
		ScopeType: "org", ScopeID: org.ID, EventFilter: "[]",
	}
	db.Create(&sub)
	db.Create(&models.WebhookDelivery{
		SubscriptionID: sub.ID, EventType: "test", Payload: "{}", StatusCode: 500,
	})
	db.Create(&models.WebhookDelivery{
		SubscriptionID: sub.ID, EventType: "test", Payload: "{}", StatusCode: 200,
	})

	// Create unread notification.
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "nu", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.Notification{UserID: "nu", Type: "mention", Title: "Test", IsRead: false})

	stats, err := svc.GetPlatformStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.FailedWebhooks24h)
	assert.Equal(t, int64(1), stats.PendingNotifications)
}

// --- Fuzz Tests ---

func FuzzUpdateSettingsValues(f *testing.F) {
	f.Add(`{"max_size":100}`)
	f.Add(`{}`)
	f.Add(`{"a":"b","c":{"d":1}}`)
	f.Add(`[]`)
	f.Add(`"string"`)
	f.Add(`null`)
	f.Add(strings.Repeat(`{"a":`, 100) + `1` + strings.Repeat(`}`, 100))
	f.Add(`{"max_size":-1,"allowed_types":[]}`)

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, value string) {
		patch := map[string]json.RawMessage{
			"file_upload_limits": json.RawMessage(value),
		}
		// Should not panic.
		_ = svc.UpdateSettings(ctx, patch, "fuzzer")
	})
}

func FuzzRBACOverride(f *testing.F) {
	f.Add(`{"roles":{"permissions":{"viewer":["read","write"]}}}`)
	f.Add(`{}`)
	f.Add(`{"defaults":{"org_member_role":"admin"}}`)
	f.Add(`{"roles":{"permissions":{"fake_role":["perm"]}}}`)
	f.Add(`invalid json`)
	f.Add(`{"roles":null}`)

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, input string) {
		var override RBACOverride
		if err := json.Unmarshal([]byte(input), &override); err != nil {
			return // Skip invalid JSON.
		}
		// Should not panic.
		_ = svc.UpdateRBACOverride(ctx, override, "fuzzer")
	})
}

func FuzzFeatureFlagKey(f *testing.F) {
	f.Add("maintenance_mode")
	f.Add("community_voting")
	f.Add("")
	f.Add(strings.Repeat("x", 1000))
	f.Add("<script>alert('xss')</script>")
	f.Add("flag with spaces")
	f.Add("flag\nwith\nnewlines")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.SeedFeatureFlags(ctx)

	f.Fuzz(func(t *testing.T, key string) {
		// Should not panic.
		_, _ = svc.GetFeatureFlag(ctx, key)
		_ = svc.ToggleFeatureFlag(ctx, key, true, nil)
		_, _ = svc.IsFeatureEnabled(ctx, key)
	})
}

func FuzzSettingsKey(f *testing.F) {
	f.Add("file_upload_limits")
	f.Add("unknown_key")
	f.Add("")
	f.Add(strings.Repeat("a", 500))
	f.Add("key with special <>&\"' chars")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, key string) {
		patch := map[string]json.RawMessage{
			key: json.RawMessage(`{}`),
		}
		// Should not panic.
		_ = svc.UpdateSettings(ctx, patch, "fuzzer")
		_, _ = svc.GetSetting(ctx, key)
	})
}

// setupFuzzDB creates a minimal test DB for fuzz tests.
func setupFuzzDB(f interface{ Fatal(...any) }) *gorm.DB {
	db := setupFuzzDBInner()
	if db == nil {
		f.Fatal("failed to setup fuzz DB")
	}
	return db
}

func setupFuzzDBInner() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil
	}
	_ = db.AutoMigrate(
		&models.SystemSetting{},
		&models.FeatureFlag{},
		&models.Org{},
		&models.UserShadow{},
		&models.OrgMembership{},
	)
	return db
}
