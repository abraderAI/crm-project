package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for notifications and preferences.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new notification repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new notification.
func (r *Repository) Create(ctx context.Context, notif *models.Notification) error {
	if err := r.db.WithContext(ctx).Create(notif).Error; err != nil {
		return fmt.Errorf("creating notification: %w", err)
	}
	return nil
}

// FindByID retrieves a notification by ID.
func (r *Repository) FindByID(ctx context.Context, id string) (*models.Notification, error) {
	var notif models.Notification
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&notif).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding notification: %w", err)
	}
	return &notif, nil
}

// ListByUser returns paginated notifications for a user.
func (r *Repository) ListByUser(ctx context.Context, userID string, params pagination.Params) ([]models.Notification, *pagination.PageInfo, error) {
	var notifs []models.Notification
	query := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&notifs).Error; err != nil {
		return nil, nil, fmt.Errorf("listing notifications: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(notifs) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(notifs[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		notifs = notifs[:params.Limit]
	}

	return notifs, pageInfo, nil
}

// MarkRead marks a notification as read.
func (r *Repository) MarkRead(ctx context.Context, id, userID string) error {
	result := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if result.Error != nil {
		return fmt.Errorf("marking notification read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead marks all unread notifications as read for a user.
func (r *Repository) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true)
	if result.Error != nil {
		return 0, fmt.Errorf("marking all notifications read: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// CountUnread returns the count of unread notifications for a user.
func (r *Repository) CountUnread(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting unread: %w", err)
	}
	return count, nil
}

// GetPreferences returns notification preferences for a user.
func (r *Repository) GetPreferences(ctx context.Context, userID string) ([]models.NotificationPreference, error) {
	var prefs []models.NotificationPreference
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&prefs).Error; err != nil {
		return nil, fmt.Errorf("getting preferences: %w", err)
	}
	return prefs, nil
}

// UpsertPreference creates or updates a notification preference.
func (r *Repository) UpsertPreference(ctx context.Context, pref *models.NotificationPreference) error {
	var existing models.NotificationPreference
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND event_type = ? AND channel = ?", pref.UserID, pref.EventType, pref.Channel).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(pref).Error
	}
	if err != nil {
		return fmt.Errorf("checking existing preference: %w", err)
	}

	return r.db.WithContext(ctx).Model(&existing).Update("enabled", pref.Enabled).Error
}

// IsChannelEnabled checks if a specific channel is enabled for an event type for a user.
// Returns true by default if no preference exists.
func (r *Repository) IsChannelEnabled(ctx context.Context, userID, eventType, channel string) (bool, error) {
	var pref models.NotificationPreference
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND event_type = ? AND channel = ?", userID, eventType, channel).
		First(&pref).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil // Default: enabled.
	}
	if err != nil {
		return false, fmt.Errorf("checking channel enabled: %w", err)
	}
	return pref.Enabled, nil
}

// GetDigestSchedule returns the digest schedule for a user.
func (r *Repository) GetDigestSchedule(ctx context.Context, userID string) (*models.DigestSchedule, error) {
	var sched models.DigestSchedule
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&sched).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting digest schedule: %w", err)
	}
	return &sched, nil
}

// UpsertDigestSchedule creates or updates a digest schedule.
func (r *Repository) UpsertDigestSchedule(ctx context.Context, sched *models.DigestSchedule) error {
	var existing models.DigestSchedule
	err := r.db.WithContext(ctx).Where("user_id = ?", sched.UserID).First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(sched).Error
	}
	if err != nil {
		return fmt.Errorf("checking existing schedule: %w", err)
	}

	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"frequency": sched.Frequency,
		"enabled":   sched.Enabled,
	}).Error
}

// GetUnreadNotifications returns all unread notifications for a user.
func (r *Repository) GetUnreadNotifications(ctx context.Context, userID string) ([]models.Notification, error) {
	var notifs []models.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_read = ?", userID, false).
		Order("created_at DESC").
		Limit(100).
		Find(&notifs).Error; err != nil {
		return nil, fmt.Errorf("getting unread notifications: %w", err)
	}
	return notifs, nil
}

// GetUsersWithDigestEnabled returns user IDs that have digest enabled with the given frequency.
func (r *Repository) GetUsersWithDigestEnabled(ctx context.Context, frequency string) ([]string, error) {
	var schedules []models.DigestSchedule
	if err := r.db.WithContext(ctx).
		Where("enabled = ? AND frequency = ?", true, frequency).
		Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("getting digest users: %w", err)
	}
	userIDs := make([]string, len(schedules))
	for i, s := range schedules {
		userIDs[i] = s.UserID
	}
	return userIDs, nil
}
