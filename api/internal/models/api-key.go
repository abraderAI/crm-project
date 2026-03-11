package models

import "time"

// APIKey stores hashed API keys for programmatic access.
type APIKey struct {
	BaseModel
	OrgID       string     `gorm:"type:text;not null;index" json:"org_id"`
	Name        string     `gorm:"type:text;not null" json:"name"`
	KeyHash     string     `gorm:"type:text;not null;uniqueIndex" json:"-"`
	KeyPrefix   string     `gorm:"type:text;not null" json:"key_prefix"`
	Permissions string     `gorm:"type:text;default:'{}'" json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Associations.
	Org Org `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
}
