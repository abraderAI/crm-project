// Package admin provides platform administration endpoints and middleware
// for the DEFT Evolution CRM platform.
package admin

import (
	"gorm.io/gorm"
)

// Service provides platform administration operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new admin service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}
