package llm

import (
	"context"
	"encoding/json"
	"log/slog"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/notification"
)

// EmailReceivedPayload is the JSON structure published on email.received events.
type EmailReceivedPayload struct {
	MessageID       string `json:"message_id"`
	EntityThreadID  string `json:"entity_thread_id"`
	RecipientUserID string `json:"recipient_user_id"`
}

// EmailSummarySubscriber listens for email.received events and generates
// AI summaries delivered as in-app notifications. Processing is async and
// non-blocking — the webhook handler publishes the event and returns
// immediately.
type EmailSummarySubscriber struct {
	provider  LLMProvider
	db        *gorm.DB
	notifRepo *notification.Repository
	logger    *slog.Logger
}

// NewEmailSummarySubscriber creates a new subscriber.
func NewEmailSummarySubscriber(provider LLMProvider, db *gorm.DB, notifRepo *notification.Repository, logger *slog.Logger) *EmailSummarySubscriber {
	return &EmailSummarySubscriber{
		provider:  provider,
		db:        db,
		notifRepo: notifRepo,
		logger:    logger,
	}
}

// Subscribe registers the handler on the event bus. The bus calls handlers
// asynchronously (goroutine) so this is inherently non-blocking.
func (s *EmailSummarySubscriber) Subscribe(bus *event.Bus) {
	bus.Subscribe(event.EmailReceived, s.handleEmailReceived)
}

// handleEmailReceived processes an email.received event. It runs in a
// goroutine spawned by the event bus, so it does not block the webhook handler.
func (s *EmailSummarySubscriber) handleEmailReceived(evt event.Event) {
	var payload EmailReceivedPayload
	if err := json.Unmarshal([]byte(evt.Payload), &payload); err != nil {
		s.logger.Error("email subscriber: invalid payload", slog.String("error", err.Error()))
		return
	}

	if payload.EntityThreadID == "" {
		// Unmatched email (no entity thread); skip summary.
		return
	}

	ctx := context.Background()

	// Load the email message.
	var msg models.Message
	if err := s.db.WithContext(ctx).Where("id = ?", payload.MessageID).First(&msg).Error; err != nil {
		s.logger.Error("email subscriber: message not found",
			slog.String("message_id", payload.MessageID),
			slog.String("error", err.Error()))
		return
	}

	// Load the entity thread.
	var thread models.Thread
	if err := s.db.WithContext(ctx).Where("id = ?", payload.EntityThreadID).First(&thread).Error; err != nil {
		s.logger.Error("email subscriber: thread not found",
			slog.String("thread_id", payload.EntityThreadID),
			slog.String("error", err.Error()))
		return
	}

	// Call LLM for summary.
	summary, err := s.provider.EmailSummary(ctx, msg, thread)
	if err != nil {
		s.logger.Error("email subscriber: LLM call failed", slog.String("error", err.Error()))
		return
	}

	// Deliver summary as in-app notification to the recipient.
	notif := &models.Notification{
		UserID:     payload.RecipientUserID,
		Type:       "email_summary",
		Title:      "Email Summary",
		Body:       summary,
		EntityType: "thread",
		EntityID:   payload.EntityThreadID,
	}
	if err := s.notifRepo.Create(ctx, notif); err != nil {
		s.logger.Error("email subscriber: notification create failed", slog.String("error", err.Error()))
	}
}
