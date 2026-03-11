package models

// Notification represents an in-app notification for a user.
type Notification struct {
	BaseModel
	UserID     string `gorm:"type:text;not null;index" json:"user_id"`
	Type       string `gorm:"type:text;not null" json:"type"`
	Title      string `gorm:"type:text;not null" json:"title"`
	Body       string `gorm:"type:text" json:"body,omitempty"`
	EntityType string `gorm:"type:text" json:"entity_type,omitempty"`
	EntityID   string `gorm:"type:text" json:"entity_id,omitempty"`
	IsRead     bool   `gorm:"default:false" json:"is_read"`
}

// NotificationPreference stores per-channel per-event notification settings.
type NotificationPreference struct {
	BaseModel
	UserID    string `gorm:"type:text;not null;uniqueIndex:idx_notif_pref" json:"user_id"`
	EventType string `gorm:"type:text;not null;uniqueIndex:idx_notif_pref" json:"event_type"`
	Channel   string `gorm:"type:text;not null;uniqueIndex:idx_notif_pref" json:"channel"`
	Enabled   bool   `gorm:"not null" json:"enabled"`
}

// DigestSchedule stores digest email scheduling preferences per user.
type DigestSchedule struct {
	BaseModel
	UserID    string `gorm:"type:text;not null;uniqueIndex" json:"user_id"`
	Frequency string `gorm:"type:text;not null;default:'daily'" json:"frequency"`
	Enabled   bool   `gorm:"not null" json:"enabled"`
}
