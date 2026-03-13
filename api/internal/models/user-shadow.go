package models

import "time"

// UserShadow is a local cache of Clerk user data, synced on every
// authenticated request for eventual consistency.
type UserShadow struct {
	ClerkUserID string     `gorm:"type:text;primaryKey" json:"clerk_user_id"`
	Email       string     `gorm:"type:text;index" json:"email"`
	DisplayName string     `gorm:"type:text;index" json:"display_name"`
	AvatarURL   string     `gorm:"type:text" json:"avatar_url,omitempty"`
	LastSeenAt  time.Time  `gorm:"index" json:"last_seen_at"`
	IsBanned    bool       `gorm:"default:false;index" json:"is_banned"`
	BanReason   string     `gorm:"type:text" json:"ban_reason,omitempty"`
	SyncedAt    time.Time  `json:"synced_at"`
	BannedAt    *time.Time `json:"banned_at,omitempty"`
	BannedBy    string     `gorm:"type:text" json:"banned_by,omitempty"`
}
