package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// TriggerEngine subscribes to the event bus and routes notifications.
type TriggerEngine struct {
	bus       *eventbus.Bus
	repo      *Repository
	providers []NotificationProvider
	logger    *slog.Logger
	unsub     func()
	done      chan struct{}
}

// NewTriggerEngine creates a new notification trigger engine.
func NewTriggerEngine(bus *eventbus.Bus, repo *Repository, providers []NotificationProvider, logger *slog.Logger) *TriggerEngine {
	return &TriggerEngine{
		bus:       bus,
		repo:      repo,
		providers: providers,
		logger:    logger,
		done:      make(chan struct{}),
	}
}

// Start begins listening for events.
func (t *TriggerEngine) Start() {
	events, unsub := t.bus.Subscribe("", 256)
	t.unsub = unsub

	go func() {
		defer close(t.done)
		for event := range events {
			t.handleEvent(event)
		}
	}()
}

// Stop stops the trigger engine.
func (t *TriggerEngine) Stop() {
	if t.unsub != nil {
		t.unsub()
	}
	<-t.done
}

// handleEvent maps an event to notifications and sends them via providers.
func (t *TriggerEngine) handleEvent(event eventbus.Event) {
	ctx := context.Background()

	notifType, title, body := mapEventToNotification(event)
	if notifType == "" {
		return
	}

	recipients := t.determineRecipients(event)
	if len(recipients) == 0 {
		return
	}

	for _, userID := range recipients {
		// Skip notifying the user who triggered the event.
		if userID == event.UserID {
			continue
		}

		notif := &models.Notification{
			UserID:     userID,
			Type:       notifType,
			Title:      title,
			Body:       body,
			EntityType: event.EntityType,
			EntityID:   event.EntityID,
		}

		for _, provider := range t.providers {
			// Check user preferences for this provider's channel.
			enabled, err := t.repo.IsChannelEnabled(ctx, userID, notifType, provider.Name())
			if err != nil {
				t.logger.Error("failed to check channel preference",
					slog.String("error", err.Error()),
					slog.String("user_id", userID),
				)
				continue
			}
			if !enabled {
				continue
			}

			if err := provider.Send(ctx, notif); err != nil {
				t.logger.Error("failed to send notification",
					slog.String("provider", provider.Name()),
					slog.String("error", err.Error()),
				)
			}
		}
	}
}

// mapEventToNotification maps an event type to a notification type, title, and body.
func mapEventToNotification(event eventbus.Event) (notifType, title, body string) {
	switch event.Type {
	case "message.created":
		return TypeNewMessage, "New message", formatBody(event, "A new message was posted")
	case "thread.updated":
		// Check if it's a stage change.
		if hasPayloadField(event.Payload, "stage") {
			return TypeStageChange, "Stage changed", formatBody(event, "Thread stage was updated")
		}
		// Check if it's an assignment change.
		if hasPayloadField(event.Payload, "assigned_to") {
			return TypeAssignment, "Assigned to you", formatBody(event, "A thread was assigned to you")
		}
		return "", "", ""
	case "mention":
		return TypeMention, "You were mentioned", formatBody(event, "Someone mentioned you")
	case "invite":
		return TypeInvite, "You're invited", formatBody(event, "You've been invited")
	default:
		return "", "", ""
	}
}

// determineRecipients determines which users should receive the notification.
func (t *TriggerEngine) determineRecipients(event eventbus.Event) []string {
	var recipients []string

	// Extract mentions from payload.
	if mentions, ok := extractPayloadStrings(event.Payload, "mentions"); ok {
		recipients = append(recipients, mentions...)
	}

	// Extract participants from payload.
	if participants, ok := extractPayloadStrings(event.Payload, "participants"); ok {
		recipients = append(recipients, participants...)
	}

	// Extract assigned_to from payload.
	if assignedTo, ok := extractPayloadString(event.Payload, "assigned_to"); ok && assignedTo != "" {
		recipients = append(recipients, assignedTo)
	}

	// Deduplicate.
	seen := make(map[string]bool)
	unique := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}

	return unique
}

// formatBody creates a notification body from the event.
func formatBody(event eventbus.Event, defaultMsg string) string {
	if title, ok := extractPayloadString(event.Payload, "title"); ok && title != "" {
		return fmt.Sprintf("%s: %s", defaultMsg, title)
	}
	return defaultMsg
}

// hasPayloadField checks if a payload has a specific field.
func hasPayloadField(payload any, field string) bool {
	if m, ok := payload.(map[string]any); ok {
		_, exists := m[field]
		return exists
	}
	return false
}

// extractPayloadString extracts a string field from a payload map.
func extractPayloadString(payload any, field string) (string, bool) {
	if m, ok := payload.(map[string]any); ok {
		if v, exists := m[field]; exists {
			if s, ok := v.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

// extractPayloadStrings extracts a string slice from a payload map.
func extractPayloadStrings(payload any, field string) ([]string, bool) {
	if m, ok := payload.(map[string]any); ok {
		if v, exists := m[field]; exists {
			// Try []string directly.
			if ss, ok := v.([]string); ok {
				return ss, true
			}
			// Try []any.
			if arr, ok := v.([]any); ok {
				result := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						result = append(result, s)
					}
				}
				return result, len(result) > 0
			}
			// Try comma-separated string.
			if s, ok := v.(string); ok && s != "" {
				return strings.Split(s, ","), true
			}
		}
	}
	return nil, false
}
