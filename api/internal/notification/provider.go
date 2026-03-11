// Package notification provides notification provider abstraction and implementations.
package notification

import (
	"context"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// NotificationProvider is the interface for sending notifications.
type NotificationProvider interface {
	// Send delivers a notification to the user.
	Send(ctx context.Context, notif *models.Notification) error
	// Name returns the provider name (e.g. "in_app", "email").
	Name() string
}

// NotificationInput holds the data needed to create a notification.
type NotificationInput struct {
	UserID     string `json:"user_id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Body       string `json:"body,omitempty"`
	EntityType string `json:"entity_type,omitempty"`
	EntityID   string `json:"entity_id,omitempty"`
}

// Validate checks that required fields are present.
func (n *NotificationInput) Validate() error {
	if n.UserID == "" {
		return ErrUserIDRequired
	}
	if n.Type == "" {
		return ErrTypeRequired
	}
	if n.Title == "" {
		return ErrTitleRequired
	}
	return nil
}
