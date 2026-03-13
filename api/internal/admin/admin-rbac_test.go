package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/models"
)

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

// --- RBAC Tests ---

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

func TestIsValidationErr(t *testing.T) {
	assert.True(t, isValidationErr(fmt.Errorf("unknown setting key: foo")))
	assert.True(t, isValidationErr(fmt.Errorf("invalid JSON value")))
	assert.False(t, isValidationErr(fmt.Errorf("database error")))
}

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

func TestHandler_PatchRBACPolicy_NoAuthContext(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"defaults":{"org_member_role":"contributor"}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/rbac-policy", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PatchRBACPolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
