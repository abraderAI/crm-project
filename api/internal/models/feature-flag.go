package models

import "time"

// FeatureFlag represents a lightweight feature toggle.
// OrgScope is NULL for global flags, or set to an org ID for org-scoped flags.
type FeatureFlag struct {
	Key       string    `gorm:"type:text;primaryKey" json:"key"`
	Enabled   bool      `gorm:"not null;default:false" json:"enabled"`
	OrgScope  *string   `gorm:"type:text" json:"org_scope,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (FeatureFlag) TableName() string {
	return "feature_flags"
}
