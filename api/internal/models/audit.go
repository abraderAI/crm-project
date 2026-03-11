package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditAction represents the type of audited operation.
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
)

// IsValid checks if the audit action is a recognized value.
func (a AuditAction) IsValid() bool {
	switch a {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete:
		return true
	}
	return false
}

// AuditLog records an immutable audit trail for every mutation.
// It does not use soft deletes — entries are permanent.
type AuditLog struct {
	ID          string      `gorm:"type:text;primaryKey" json:"id"`
	CreatedAt   time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UserID      string      `gorm:"type:text;not null;index" json:"user_id"`
	Action      AuditAction `gorm:"type:text;not null" json:"action"`
	EntityType  string      `gorm:"type:text;not null;index:idx_audit_entity" json:"entity_type"`
	EntityID    string      `gorm:"type:text;not null;index:idx_audit_entity" json:"entity_id"`
	BeforeState string      `gorm:"type:text" json:"before_state,omitempty"`
	AfterState  string      `gorm:"type:text" json:"after_state,omitempty"`
	IPAddress   string      `gorm:"type:text" json:"ip_address,omitempty"`
	RequestID   string      `gorm:"type:text" json:"request_id,omitempty"`
}

// BeforeCreate generates a UUIDv7 for the ID field if not already set.
func (a *AuditLog) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		a.ID = id.String()
	}
	return nil
}
