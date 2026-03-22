package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// knownSettingKeys returns the valid top-level setting keys with their schema descriptions.
// Returns a fresh copy on each call to prevent external mutation.
func knownSettingKeys() map[string]string {
	return map[string]string{
		"default_pipeline_stages": "array of stage names for new CRM spaces",
		"default_templates":       "default Space/Board templates for provisioned customer Orgs",
		"notification_defaults":   "digest frequency, email from address, etc.",
		"file_upload_limits":      "max_size (bytes), allowed_types (array of MIME types)",
		"webhook_retry_policy":    "max_attempts (int), backoff_multiplier (float)",
		"llm_rate_limits":         "provider rate limits configuration",
	}
}

// initialSettingDefaults returns the default values for all known setting keys.
// Returns a fresh map on each call to prevent external mutation.
func initialSettingDefaults() map[string]string {
	return map[string]string{
		"default_pipeline_stages": `["new_lead","contacted","qualified","proposal","negotiation","closed_won","closed_lost"]`,
		"default_templates":       `{"spaces":["general","support"],"boards_per_space":["default"]}`,
		"notification_defaults":   `{"digest_frequency":"daily","email_from":"noreply@deft.dev"}`,
		"file_upload_limits":      `{"max_size":10485760,"allowed_types":["image/png","image/jpeg","application/pdf"]}`,
		"webhook_retry_policy":    `{"max_attempts":5,"backoff_multiplier":2.0}`,
		"llm_rate_limits":         `{"requests_per_minute":60,"tokens_per_minute":100000}`,
	}
}

// SeedSettings creates the initial system settings if they don't already exist.
func (s *Service) SeedSettings(ctx context.Context) error {
	for key, value := range initialSettingDefaults() {
		var existing models.SystemSetting
		err := s.db.WithContext(ctx).Where("key = ?", key).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			setting := models.SystemSetting{
				Key:       key,
				Value:     value,
				UpdatedBy: "system",
			}
			if err := s.db.WithContext(ctx).Create(&setting).Error; err != nil {
				return fmt.Errorf("seeding setting %s: %w", key, err)
			}
		} else if err != nil {
			return fmt.Errorf("checking setting %s: %w", key, err)
		}
	}
	return nil
}

// GetAllSettings returns all system settings as a single JSON object.
func (s *Service) GetAllSettings(ctx context.Context) (map[string]json.RawMessage, error) {
	var settings []models.SystemSetting
	if err := s.db.WithContext(ctx).Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("listing settings: %w", err)
	}

	result := make(map[string]json.RawMessage, len(settings))
	for _, setting := range settings {
		result[setting.Key] = json.RawMessage(setting.Value)
	}
	return result, nil
}

// GetSetting returns a single system setting by key.
func (s *Service) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	var setting models.SystemSetting
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting setting %s: %w", key, err)
	}
	return &setting, nil
}

// UpdateSettings deep-merges a patch into the existing settings.
// Only known setting keys are accepted; unknown keys are rejected.
func (s *Service) UpdateSettings(ctx context.Context, patch map[string]json.RawMessage, updatedBy string) error {
	// Validate keys.
	keys := knownSettingKeys()
	for key := range patch {
		if _, ok := keys[key]; !ok {
			return fmt.Errorf("unknown setting key: %s", key)
		}
	}

	// Validate each value is valid JSON.
	for key, val := range patch {
		if !json.Valid(val) {
			return fmt.Errorf("invalid JSON value for key %s", key)
		}
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, val := range patch {
			var existing models.SystemSetting
			err := tx.Where("key = ?", key).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				// Create new setting.
				setting := models.SystemSetting{
					Key:       key,
					Value:     string(val),
					UpdatedBy: updatedBy,
				}
				if err := tx.Create(&setting).Error; err != nil {
					return fmt.Errorf("creating setting %s: %w", key, err)
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("reading setting %s: %w", key, err)
			}

			// Deep-merge the existing value with the patch.
			merged, err := metadata.DeepMerge(existing.Value, string(val))
			if err != nil {
				return fmt.Errorf("merging setting %s: %w", key, err)
			}

			if err := tx.Model(&existing).Updates(map[string]any{
				"value":      merged,
				"updated_by": updatedBy,
			}).Error; err != nil {
				return fmt.Errorf("updating setting %s: %w", key, err)
			}
		}
		return nil
	})
}

// GetSettings handles GET /v1/admin/settings.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetAllSettings(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to get settings")
		return
	}
	response.JSON(w, http.StatusOK, settings)
}

// PatchSettings handles PATCH /v1/admin/settings.
func (h *Handler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if len(patch) == 0 {
		apierrors.ValidationError(w, "at least one setting key is required", nil)
		return
	}

	uc := auth.GetUserContext(r.Context())
	updatedBy := ""
	if uc != nil {
		updatedBy = uc.UserID
	}

	if err := h.service.UpdateSettings(r.Context(), patch, updatedBy); err != nil {
		if isValidationErr(err) {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		apierrors.InternalError(w, "failed to update settings")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "update", "system_settings", "settings", nil, patch)

	// Return updated settings.
	settings, err := h.service.GetAllSettings(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to get settings")
		return
	}
	response.JSON(w, http.StatusOK, settings)
}

// isValidationErr returns true if the error is a validation error (not internal).
func isValidationErr(err error) bool {
	msg := err.Error()
	return len(msg) > 0 && (msg[:7] == "unknown" || msg[:7] == "invalid")
}
