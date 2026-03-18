package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// initialFeatureFlags returns the feature flags seeded on startup.
// Returns a fresh slice on each call to prevent external mutation.
func initialFeatureFlags() []models.FeatureFlag {
	return []models.FeatureFlag{
		{Key: "community_voting", Enabled: true},
		{Key: "voice_module", Enabled: false},
		{Key: "maintenance_mode", Enabled: false},
	}
}

// SeedFeatureFlags creates the initial feature flags if they don't already exist.
func (s *Service) SeedFeatureFlags(ctx context.Context) error {
	for _, flag := range initialFeatureFlags() {
		var existing models.FeatureFlag
		err := s.db.WithContext(ctx).Where("key = ?", flag.Key).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.WithContext(ctx).Create(&flag).Error; err != nil {
				return fmt.Errorf("seeding feature flag %s: %w", flag.Key, err)
			}
		} else if err != nil {
			return fmt.Errorf("checking feature flag %s: %w", flag.Key, err)
		}
	}
	return nil
}

// ListFeatureFlags returns all feature flags.
func (s *Service) ListFeatureFlags(ctx context.Context) ([]models.FeatureFlag, error) {
	var flags []models.FeatureFlag
	if err := s.db.WithContext(ctx).Order("key ASC").Find(&flags).Error; err != nil {
		return nil, fmt.Errorf("listing feature flags: %w", err)
	}
	return flags, nil
}

// GetFeatureFlag returns a single feature flag by key.
func (s *Service) GetFeatureFlag(ctx context.Context, key string) (*models.FeatureFlag, error) {
	var flag models.FeatureFlag
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&flag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting feature flag %s: %w", key, err)
	}
	return &flag, nil
}

// ToggleFeatureFlag updates a feature flag's enabled state and optional org scope.
func (s *Service) ToggleFeatureFlag(ctx context.Context, key string, enabled bool, orgScope *string) error {
	var flag models.FeatureFlag
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&flag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("feature flag not found: %s", key)
		}
		return fmt.Errorf("getting feature flag: %w", err)
	}

	updates := map[string]any{"enabled": enabled}
	if orgScope != nil {
		if *orgScope == "" {
			updates["org_scope"] = nil
		} else {
			updates["org_scope"] = *orgScope
		}
	}

	if err := s.db.WithContext(ctx).Model(&flag).Updates(updates).Error; err != nil {
		return fmt.Errorf("updating feature flag: %w", err)
	}
	return nil
}

// IsFeatureEnabled checks if a feature flag is enabled (globally or for a specific org).
func (s *Service) IsFeatureEnabled(ctx context.Context, key string) (bool, error) {
	var flag models.FeatureFlag
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&flag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("checking feature flag %s: %w", key, err)
	}
	return flag.Enabled, nil
}

// ListFeatureFlagsHandler handles GET /v1/admin/feature-flags.
func (h *Handler) ListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := h.service.ListFeatureFlags(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to list feature flags")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": flags})
}

// PatchFeatureFlag handles PATCH /v1/admin/feature-flags/{key}.
func (h *Handler) PatchFeatureFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		apierrors.BadRequest(w, "feature flag key is required")
		return
	}

	var body struct {
		Enabled  *bool   `json:"enabled"`
		OrgScope *string `json:"org_scope,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if body.Enabled == nil {
		apierrors.ValidationError(w, "enabled field is required", nil)
		return
	}

	if err := h.service.ToggleFeatureFlag(r.Context(), key, *body.Enabled, body.OrgScope); err != nil {
		if fmt.Sprintf("%v", err) == fmt.Sprintf("feature flag not found: %s", key) {
			apierrors.NotFound(w, "feature flag not found")
			return
		}
		apierrors.InternalError(w, "failed to update feature flag")
		return
	}

	// Audit log.
	uc := auth.GetUserContext(r.Context())
	updatedBy := ""
	if uc != nil {
		updatedBy = uc.UserID
	}
	audit.CreateAuditEntry(r.Context(), h.auditService, "update", "feature_flag", key,
		nil, map[string]any{"enabled": *body.Enabled, "updated_by": updatedBy})

	// Return updated flag.
	flag, err := h.service.GetFeatureFlag(r.Context(), key)
	if err != nil || flag == nil {
		apierrors.InternalError(w, "failed to get updated feature flag")
		return
	}
	response.JSON(w, http.StatusOK, flag)
}
