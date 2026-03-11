package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Revision tracks content versions for threads and messages.
type Revision struct {
	ID              string    `gorm:"type:text;primaryKey" json:"id"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	EntityType      string    `gorm:"type:text;not null;index:idx_revision_entity" json:"entity_type"`
	EntityID        string    `gorm:"type:text;not null;index:idx_revision_entity" json:"entity_id"`
	Version         int       `gorm:"not null" json:"version"`
	PreviousContent string    `gorm:"type:text" json:"previous_content,omitempty"`
	EditorID        string    `gorm:"type:text;not null" json:"editor_id"`
}

// BeforeCreate generates a UUIDv7 for the ID field if not already set.
func (r *Revision) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		r.ID = id.String()
	}
	return nil
}
