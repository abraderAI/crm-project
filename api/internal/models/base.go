// Package models defines all GORM models for the DEFT Evolution CRM platform.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all entities: UUIDv7 primary key,
// timestamps, and soft delete support.
type BaseModel struct {
	ID        string         `gorm:"type:text;primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate generates a UUIDv7 for the ID field if not already set.
func (b *BaseModel) BeforeCreate(_ *gorm.DB) error {
	if b.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		b.ID = id.String()
	}
	return nil
}
