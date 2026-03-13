package models

import "time"

// PlatformAdmin grants platform-wide admin privileges to a user.
type PlatformAdmin struct {
	UserID    string    `gorm:"type:text;primaryKey" json:"user_id"`
	GrantedBy string    `gorm:"type:text" json:"granted_by"`
	GrantedAt time.Time `gorm:"autoCreateTime" json:"granted_at"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
}
