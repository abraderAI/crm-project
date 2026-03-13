package models

import "time"

// SystemSetting stores a configurable platform setting as a key-value pair.
// The value is stored as JSON to support complex nested structures.
type SystemSetting struct {
	Key       string    `gorm:"type:text;primaryKey" json:"key"`
	Value     string    `gorm:"type:text;not null;default:'{}'" json:"value"`
	UpdatedBy string    `gorm:"type:text" json:"updated_by"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (SystemSetting) TableName() string {
	return "system_settings"
}
