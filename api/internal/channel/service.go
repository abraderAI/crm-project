package channel

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Service provides business logic for channel configuration and DLQ management.
type Service struct {
	repo *Repository
}

// NewService creates a new channel Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// PutConfigInput holds the data for creating or updating a channel config.
type PutConfigInput struct {
	Settings string `json:"settings"`
	Enabled  bool   `json:"enabled"`
}

// UpsertConfig validates and creates or updates a channel config for an org.
// The returned config has secrets masked in the Settings field.
func (s *Service) UpsertConfig(ctx context.Context, orgID string, channelType models.ChannelType, input PutConfigInput) (*models.ChannelConfig, error) {
	if !channelType.IsValid() {
		return nil, fmt.Errorf("invalid channel type: %s", channelType)
	}

	settings := input.Settings
	if settings == "" {
		settings = "{}"
	}
	if err := ValidateSettings(channelType, settings); err != nil {
		return nil, err
	}

	cfg := &models.ChannelConfig{
		OrgID:       orgID,
		ChannelType: channelType,
		Settings:    settings,
		Enabled:     input.Enabled,
	}
	if err := s.repo.UpsertConfig(ctx, cfg); err != nil {
		return nil, err
	}
	// Mask secrets before returning.
	cfg.Settings = MaskSettingsSecrets(channelType, cfg.Settings)
	return cfg, nil
}

// GetConfig retrieves a channel config for an org, with secrets masked.
// Returns nil, nil when no config has been saved yet.
func (s *Service) GetConfig(ctx context.Context, orgID string, channelType models.ChannelType) (*models.ChannelConfig, error) {
	if !channelType.IsValid() {
		return nil, fmt.Errorf("invalid channel type: %s", channelType)
	}
	cfg, err := s.repo.FindConfig(ctx, orgID, channelType)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	cfg.Settings = MaskSettingsSecrets(channelType, cfg.Settings)
	return cfg, nil
}

// GetHealth returns per-channel health status for all known channel types in an org.
func (s *Service) GetHealth(ctx context.Context, orgID string) ([]ChannelHealth, error) {
	cfgs, err := s.repo.ListConfigs(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Index existing configs by channel type.
	cfgMap := make(map[models.ChannelType]*models.ChannelConfig, len(cfgs))
	for i := range cfgs {
		cfgMap[cfgs[i].ChannelType] = &cfgs[i]
	}

	health := make([]ChannelHealth, 0, len(models.ValidChannelTypes()))
	for _, ct := range models.ValidChannelTypes() {
		h := ChannelHealth{ChannelType: ct}
		if cfg, ok := cfgMap[ct]; ok {
			h.Enabled = cfg.Enabled
		}

		errCount, err := s.repo.CountRecentDLQEvents(ctx, orgID, ct)
		if err != nil {
			return nil, err
		}
		h.ErrorCount = errCount

		lastEvt, err := s.repo.GetLatestEventTime(ctx, orgID, ct)
		if err != nil {
			return nil, err
		}
		h.LastEventAt = lastEvt

		h.Status = computeHealthStatus(errCount, h.Enabled)
		health = append(health, h)
	}
	return health, nil
}

// computeHealthStatus derives a HealthStatus from error rate and enabled state.
// A disabled channel is always Down. 0 errors → Healthy; 1-5 → Degraded; >5 → Down.
func computeHealthStatus(errorCount24h int64, enabled bool) HealthStatus {
	if !enabled {
		return HealthStatusDown
	}
	switch {
	case errorCount24h == 0:
		return HealthStatusHealthy
	case errorCount24h <= 5:
		return HealthStatusDegraded
	default:
		return HealthStatusDown
	}
}

// ListDLQEventsInput holds filter and pagination parameters for listing DLQ events.
type ListDLQEventsInput struct {
	ChannelType models.ChannelType
	Status      models.DLQStatus
	Params      pagination.Params
}

// ListDLQEvents returns a paginated list of DLQ events for an org.
func (s *Service) ListDLQEvents(ctx context.Context, orgID string, input ListDLQEventsInput) ([]models.DeadLetterEvent, *pagination.PageInfo, error) {
	return s.repo.ListDLQEvents(ctx, orgID, input.ChannelType, input.Status, input.Params)
}

// RetryDLQEvent marks a DLQ event as retrying so an operator can reprocess it.
// Returns nil, nil when the event does not exist or belongs to a different org.
func (s *Service) RetryDLQEvent(ctx context.Context, orgID, id string) (*models.DeadLetterEvent, error) {
	evt, err := s.repo.FindDLQEvent(ctx, id)
	if err != nil {
		return nil, err
	}
	if evt == nil || evt.OrgID != orgID {
		return nil, nil
	}
	evt.Status = models.DLQStatusRetrying
	if err := s.repo.UpdateDLQEvent(ctx, evt); err != nil {
		return nil, err
	}
	return evt, nil
}

// DismissDLQEvent marks a DLQ event as dismissed so it will not be retried.
// Returns nil, nil when the event does not exist or belongs to a different org.
func (s *Service) DismissDLQEvent(ctx context.Context, orgID, id string) (*models.DeadLetterEvent, error) {
	evt, err := s.repo.FindDLQEvent(ctx, id)
	if err != nil {
		return nil, err
	}
	if evt == nil || evt.OrgID != orgID {
		return nil, nil
	}
	evt.Status = models.DLQStatusDismissed
	if err := s.repo.UpdateDLQEvent(ctx, evt); err != nil {
		return nil, err
	}
	return evt, nil
}

// IsPlatformAdmin returns true when the user is an active platform admin.
func (s *Service) IsPlatformAdmin(ctx context.Context, userID string) (bool, error) {
	return s.repo.IsPlatformAdmin(ctx, userID)
}

// IsOrgAdmin returns true when the user has admin or owner role in the org.
func (s *Service) IsOrgAdmin(ctx context.Context, orgID, userID string) (bool, error) {
	return s.repo.IsOrgAdmin(ctx, orgID, userID)
}
