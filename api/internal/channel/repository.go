package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for channel configs and DLQ events.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new channel Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- Channel Config ---

// UpsertConfig creates or updates the ChannelConfig for (orgID, channelType).
// If a config already exists it is updated in place; otherwise a new one is created.
func (r *Repository) UpsertConfig(ctx context.Context, cfg *models.ChannelConfig) error {
	var existing models.ChannelConfig
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND channel_type = ?", cfg.OrgID, cfg.ChannelType).
		First(&existing).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("finding channel config: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if createErr := r.db.WithContext(ctx).Create(cfg).Error; createErr != nil {
			return fmt.Errorf("creating channel config: %w", createErr)
		}
		return nil
	}

	existing.Settings = cfg.Settings
	existing.Enabled = cfg.Enabled
	if saveErr := r.db.WithContext(ctx).Save(&existing).Error; saveErr != nil {
		return fmt.Errorf("updating channel config: %w", saveErr)
	}
	*cfg = existing
	return nil
}

// FindConfig retrieves a ChannelConfig by org ID and channel type.
// Returns nil, nil when no record exists.
func (r *Repository) FindConfig(ctx context.Context, orgID string, channelType models.ChannelType) (*models.ChannelConfig, error) {
	var cfg models.ChannelConfig
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND channel_type = ?", orgID, channelType).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding channel config: %w", err)
	}
	return &cfg, nil
}

// ListConfigs returns all ChannelConfig records for an org.
func (r *Repository) ListConfigs(ctx context.Context, orgID string) ([]models.ChannelConfig, error) {
	var cfgs []models.ChannelConfig
	if err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&cfgs).Error; err != nil {
		return nil, fmt.Errorf("listing channel configs: %w", err)
	}
	return cfgs, nil
}

// --- Dead Letter Queue ---

// CreateDLQEvent inserts a new DeadLetterEvent.
func (r *Repository) CreateDLQEvent(ctx context.Context, evt *models.DeadLetterEvent) error {
	if err := r.db.WithContext(ctx).Create(evt).Error; err != nil {
		return fmt.Errorf("creating DLQ event: %w", err)
	}
	return nil
}

// FindDLQEvent retrieves a DeadLetterEvent by its ID.
// Returns nil, nil when no record exists.
func (r *Repository) FindDLQEvent(ctx context.Context, id string) (*models.DeadLetterEvent, error) {
	var evt models.DeadLetterEvent
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&evt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding DLQ event: %w", err)
	}
	return &evt, nil
}

// UpdateDLQEvent saves changes to an existing DeadLetterEvent.
func (r *Repository) UpdateDLQEvent(ctx context.Context, evt *models.DeadLetterEvent) error {
	if err := r.db.WithContext(ctx).Save(evt).Error; err != nil {
		return fmt.Errorf("updating DLQ event: %w", err)
	}
	return nil
}

// ListDLQEvents returns a paginated list of DLQ events for an org.
// Optionally filtered by channelType and/or status (pass empty strings to skip filters).
func (r *Repository) ListDLQEvents(
	ctx context.Context,
	orgID string,
	channelType models.ChannelType,
	status models.DLQStatus,
	params pagination.Params,
) ([]models.DeadLetterEvent, *pagination.PageInfo, error) {
	query := r.db.WithContext(ctx).Where("org_id = ?", orgID)
	if channelType != "" {
		query = query.Where("channel_type = ?", channelType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query = query.Order("created_at DESC")

	var evts []models.DeadLetterEvent
	if err := query.Limit(params.Limit + 1).Find(&evts).Error; err != nil {
		return nil, nil, fmt.Errorf("listing DLQ events: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(evts) > params.Limit {
		pageInfo.HasMore = true
		evts = evts[:params.Limit]
	}
	return evts, pageInfo, nil
}

// CountRecentDLQEvents counts failed or retrying events created within the last 24 hours
// for a specific org and channel type. Used to compute channel health.
func (r *Repository) CountRecentDLQEvents(ctx context.Context, orgID string, channelType models.ChannelType) (int64, error) {
	since := time.Now().Add(-24 * time.Hour)
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DeadLetterEvent{}).
		Where("org_id = ? AND channel_type = ? AND status IN ? AND created_at > ?",
			orgID, channelType, []models.DLQStatus{models.DLQStatusFailed, models.DLQStatusRetrying}, since).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting DLQ events: %w", err)
	}
	return count, nil
}

// GetLatestEventTime returns the CreatedAt timestamp of the most recently created
// DLQ event for an org and channel type. Returns nil, nil when no events exist.
func (r *Repository) GetLatestEventTime(ctx context.Context, orgID string, channelType models.ChannelType) (*time.Time, error) {
	var evt models.DeadLetterEvent
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND channel_type = ?", orgID, channelType).
		Order("created_at DESC").
		First(&evt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting latest event time: %w", err)
	}
	return &evt.CreatedAt, nil
}

// IsOrgAdmin returns true when the given user has admin or owner role in the org.
func (r *Repository) IsOrgAdmin(ctx context.Context, orgID, userID string) (bool, error) {
	var m models.OrgMembership
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking org membership: %w", err)
	}
	return m.Role == models.RoleAdmin || m.Role == models.RoleOwner, nil
}
