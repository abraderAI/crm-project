package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminExport tracks async data export requests.
type AdminExport struct {
	ID          string     `gorm:"type:text;primaryKey" json:"id"`
	Type        string     `gorm:"type:text;not null" json:"type"`        // users, orgs, audit
	Filters     string     `gorm:"type:text;default:'{}'" json:"filters"` // JSON filters
	Format      string     `gorm:"type:text;not null" json:"format"`      // csv, json
	Status      string     `gorm:"type:text;not null" json:"status"`      // pending, processing, completed, failed
	FilePath    string     `gorm:"type:text" json:"file_path,omitempty"`
	RequestedBy string     `gorm:"type:text;not null" json:"requested_by"`
	ErrorMsg    string     `gorm:"type:text" json:"error_msg,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// BeforeCreate generates a UUIDv7 for the ID field if not already set.
func (e *AdminExport) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		e.ID = id.String()
	}
	return nil
}
