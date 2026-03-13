package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

const rbacOverrideKey = "rbac_policy_override"

// RBACOverride represents an override to the base RBAC policy.
type RBACOverride struct {
	Roles    *RBACRolesOverride    `json:"roles,omitempty" yaml:"roles,omitempty"`
	Defaults *RBACDefaultsOverride `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

// RBACRolesOverride overrides role permissions.
type RBACRolesOverride struct {
	Permissions map[string][]string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

// RBACDefaultsOverride overrides default role assignments.
type RBACDefaultsOverride struct {
	OrgMemberRole   string `json:"org_member_role,omitempty" yaml:"org_member_role,omitempty"`
	SpaceMemberRole string `json:"space_member_role,omitempty" yaml:"space_member_role,omitempty"`
	BoardMemberRole string `json:"board_member_role,omitempty" yaml:"board_member_role,omitempty"`
}

// EffectivePolicy represents the merged RBAC policy (base + overrides).
type EffectivePolicy struct {
	Resolution config.Resolution          `json:"resolution"`
	Roles      EffectiveRoles             `json:"roles"`
	Defaults   config.RBACDefaults        `json:"defaults"`
	Overrides  map[string]json.RawMessage `json:"overrides,omitempty"`
}

// EffectiveRoles holds hierarchy and merged permissions.
type EffectiveRoles struct {
	Hierarchy   []string            `json:"hierarchy"`
	Permissions map[string][]string `json:"permissions"`
}

// GetRBACOverride loads the stored RBAC override from system_settings.
func (s *Service) GetRBACOverride(ctx context.Context) (*RBACOverride, error) {
	var setting models.SystemSetting
	err := s.db.WithContext(ctx).Where("key = ?", rbacOverrideKey).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting RBAC override: %w", err)
	}
	var override RBACOverride
	if err := json.Unmarshal([]byte(setting.Value), &override); err != nil {
		return nil, fmt.Errorf("parsing RBAC override: %w", err)
	}
	return &override, nil
}

// GetEffectivePolicy returns the base RBAC policy merged with DB overrides.
func (s *Service) GetEffectivePolicy(ctx context.Context, basePolicy *config.RBACPolicy) (*EffectivePolicy, error) {
	if basePolicy == nil {
		return nil, fmt.Errorf("base RBAC policy is nil")
	}

	effective := &EffectivePolicy{
		Resolution: basePolicy.Resolution,
		Roles: EffectiveRoles{
			Hierarchy:   basePolicy.Roles.Hierarchy,
			Permissions: make(map[string][]string, len(basePolicy.Roles.Permissions)),
		},
		Defaults: basePolicy.Defaults,
	}
	// Copy base permissions.
	for role, perms := range basePolicy.Roles.Permissions {
		cpy := make([]string, len(perms))
		copy(cpy, perms)
		effective.Roles.Permissions[role] = cpy
	}

	// Load and merge overrides.
	override, err := s.GetRBACOverride(ctx)
	if err != nil {
		return nil, err
	}
	if override == nil {
		return effective, nil
	}

	// Apply role permission overrides.
	if override.Roles != nil {
		for role, perms := range override.Roles.Permissions {
			effective.Roles.Permissions[role] = perms
		}
	}
	// Apply default overrides.
	if override.Defaults != nil {
		if override.Defaults.OrgMemberRole != "" {
			effective.Defaults.OrgMemberRole = override.Defaults.OrgMemberRole
		}
		if override.Defaults.SpaceMemberRole != "" {
			effective.Defaults.SpaceMemberRole = override.Defaults.SpaceMemberRole
		}
		if override.Defaults.BoardMemberRole != "" {
			effective.Defaults.BoardMemberRole = override.Defaults.BoardMemberRole
		}
	}

	// Include raw overrides for transparency.
	overrideJSON, _ := json.Marshal(override)
	var rawOverrides map[string]json.RawMessage
	_ = json.Unmarshal(overrideJSON, &rawOverrides)
	effective.Overrides = rawOverrides

	return effective, nil
}

// UpdateRBACOverride validates and stores RBAC policy overrides.
func (s *Service) UpdateRBACOverride(ctx context.Context, override RBACOverride, updatedBy string) error {
	// Validate: any role in permissions override must be valid.
	if override.Roles != nil {
		for role := range override.Roles.Permissions {
			if !isValidRole(role) {
				return fmt.Errorf("unknown role in override: %s", role)
			}
		}
	}
	// Validate: default roles must be valid.
	if override.Defaults != nil {
		for _, role := range []string{
			override.Defaults.OrgMemberRole,
			override.Defaults.SpaceMemberRole,
			override.Defaults.BoardMemberRole,
		} {
			if role != "" && !isValidRole(role) {
				return fmt.Errorf("unknown role in defaults override: %s", role)
			}
		}
	}

	overrideJSON, err := json.Marshal(override)
	if err != nil {
		return fmt.Errorf("serializing RBAC override: %w", err)
	}

	setting := models.SystemSetting{
		Key:       rbacOverrideKey,
		Value:     string(overrideJSON),
		UpdatedBy: updatedBy,
	}
	return s.db.WithContext(ctx).Where("key = ?", rbacOverrideKey).
		Assign(map[string]any{
			"value":      setting.Value,
			"updated_by": setting.UpdatedBy,
		}).FirstOrCreate(&setting).Error
}

// PreviewRBACRole does a dry-run role resolution under a proposed override.
func (s *Service) PreviewRBACRole(ctx context.Context, basePolicy *config.RBACPolicy, userID, entityType, entityID string, override *RBACOverride) (string, []string, error) {
	// Build the effective policy with the proposed override.
	effective, err := s.GetEffectivePolicy(ctx, basePolicy)
	if err != nil {
		return "", nil, err
	}

	// Apply the proposed override on top if provided.
	if override != nil {
		if override.Roles != nil {
			for role, perms := range override.Roles.Permissions {
				effective.Roles.Permissions[role] = perms
			}
		}
		if override.Defaults != nil {
			if override.Defaults.OrgMemberRole != "" {
				effective.Defaults.OrgMemberRole = override.Defaults.OrgMemberRole
			}
			if override.Defaults.SpaceMemberRole != "" {
				effective.Defaults.SpaceMemberRole = override.Defaults.SpaceMemberRole
			}
			if override.Defaults.BoardMemberRole != "" {
				effective.Defaults.BoardMemberRole = override.Defaults.BoardMemberRole
			}
		}
	}

	// Resolve the effective role using the RBAC engine with the base policy.
	rbacEngine := auth.NewRBACEngine(basePolicy, s.db)
	role, err := rbacEngine.ResolveRole(ctx, userID, entityType, entityID)
	if err != nil {
		return "", nil, fmt.Errorf("resolving role: %w", err)
	}

	// Look up permissions from the effective (overridden) permissions.
	permissions := effective.Roles.Permissions[string(role)]

	return string(role), permissions, nil
}

// isValidRole checks if a role name is one of the known RBAC roles.
func isValidRole(role string) bool {
	validRoles := map[string]bool{
		"viewer": true, "commenter": true, "contributor": true,
		"moderator": true, "admin": true, "owner": true,
	}
	return validRoles[role]
}

// GetRBACPolicy handles GET /v1/admin/rbac-policy.
func (h *Handler) GetRBACPolicy(w http.ResponseWriter, r *http.Request) {
	effective, err := h.service.GetEffectivePolicy(r.Context(), h.rbacPolicy)
	if err != nil {
		apierrors.InternalError(w, "failed to get effective RBAC policy")
		return
	}
	response.JSON(w, http.StatusOK, effective)
}

// PatchRBACPolicy handles PATCH /v1/admin/rbac-policy.
func (h *Handler) PatchRBACPolicy(w http.ResponseWriter, r *http.Request) {
	var override RBACOverride
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	uc := auth.GetUserContext(r.Context())
	updatedBy := ""
	if uc != nil {
		updatedBy = uc.UserID
	}

	if err := h.service.UpdateRBACOverride(r.Context(), override, updatedBy); err != nil {
		if isValidationErr(err) {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		apierrors.InternalError(w, "failed to update RBAC policy")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "update", "rbac_policy", rbacOverrideKey, nil, override)

	effective, err := h.service.GetEffectivePolicy(r.Context(), h.rbacPolicy)
	if err != nil {
		apierrors.InternalError(w, "failed to get effective RBAC policy")
		return
	}
	response.JSON(w, http.StatusOK, effective)
}

// PreviewRBACPolicy handles POST /v1/admin/rbac-policy/preview.
func (h *Handler) PreviewRBACPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     string        `json:"user_id"`
		EntityType string        `json:"entity_type"`
		EntityID   string        `json:"entity_id"`
		Override   *RBACOverride `json:"override,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if req.UserID == "" || req.EntityType == "" || req.EntityID == "" {
		apierrors.ValidationError(w, "user_id, entity_type, and entity_id are required", nil)
		return
	}

	role, permissions, err := h.service.PreviewRBACRole(r.Context(), h.rbacPolicy, req.UserID, req.EntityType, req.EntityID, req.Override)
	if err != nil {
		apierrors.InternalError(w, "failed to preview RBAC role")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"user_id":     req.UserID,
		"entity_type": req.EntityType,
		"entity_id":   req.EntityID,
		"role":        role,
		"permissions": permissions,
	})
}

// MarshalRBACPolicyToYAML converts an effective policy to YAML for display purposes.
func MarshalRBACPolicyToYAML(p *EffectivePolicy) (string, error) {
	data, err := yaml.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshaling policy to YAML: %w", err)
	}
	return string(data), nil
}
