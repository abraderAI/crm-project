// Package admin provides platform administration endpoints and middleware
// for the DEFT Evolution CRM platform.
package admin

import (
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
)

// Service provides platform administration operations.
type Service struct {
	db          *gorm.DB
	clerkClient *auth.ClerkClient // optional; used to enrich user shadows from Clerk API
}

// NewService creates a new admin service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// WithClerkKey configures the service to call the Clerk Backend API when JWT
// tokens do not include identity claims. A nil client is set when key is empty.
// Returns the receiver for fluent chaining.
func (s *Service) WithClerkKey(key string) *Service {
	s.clerkClient = auth.NewClerkClient(key)
	return s
}

// withClerkClient injects a pre-built ClerkClient; used in unit tests.
func (s *Service) withClerkClient(c *auth.ClerkClient) *Service {
	s.clerkClient = c
	return s
}
