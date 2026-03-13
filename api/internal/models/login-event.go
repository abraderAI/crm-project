package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginEvent records authenticated request events (debounced per user per hour).
type LoginEvent struct {
	ID        string    `gorm:"type:text;primaryKey" json:"id"`
	UserID    string    `gorm:"type:text;not null;index" json:"user_id"`
	IPAddress string    `gorm:"type:text;index" json:"ip_address"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

// BeforeCreate generates a UUIDv7 for the ID field if not already set.
func (l *LoginEvent) BeforeCreate(_ *gorm.DB) error {
	if l.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		l.ID = id.String()
	}
	return nil
}

// FailedAuth tracks failed authentication attempts per IP/user per hour.
type FailedAuth struct {
	IPAddress string `gorm:"type:text;primaryKey" json:"ip_address"`
	UserID    string `gorm:"type:text;primaryKey" json:"user_id"` // empty string for unknown users
	Hour      string `gorm:"type:text;primaryKey" json:"hour"`    // YYYY-MM-DD-HH
	Count     int64  `gorm:"default:0" json:"count"`
}
