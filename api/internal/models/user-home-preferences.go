package models

import "time"

// WidgetPreference represents a single widget's position and visibility in a layout.
type WidgetPreference struct {
	WidgetID string `json:"widget_id"`
	Visible  bool   `json:"visible"`
}

// UserHomePreferences stores per-user home screen layout configuration.
// The layout is a JSON-encoded ordered list of widget preferences.
type UserHomePreferences struct {
	UserID    string    `gorm:"type:text;primaryKey" json:"user_id"`
	Tier      int       `gorm:"not null" json:"tier"`
	Layout    string    `gorm:"type:text;not null" json:"layout"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
