package support

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
)

// ErrImmutable is returned when an edit is attempted on a locked entry.
var ErrImmutable = errors.New("entry is immutable and cannot be edited")

// ErrNotDraft is returned when a publish is attempted on a non-draft entry.
var ErrNotDraft = errors.New("only draft entries can be published")

// ErrForbidden is returned when a caller lacks permission for the operation.
var ErrForbidden = errors.New("operation not permitted for this caller")

// Service provides business logic for support ticket entry operations.
type Service struct {
	repo *Repository
	bus  *event.Bus
}

// NewService creates a new support Service.
// bus may be nil; when provided, ticket-updated events are published on publish.
func NewService(repo *Repository, bus *event.Bus) *Service {
	return &Service{repo: repo, bus: bus}
}

// CreateEntryInput holds the fields required to create a new ticket entry.
type CreateEntryInput struct {
	// Type must be one of the support-specific MessageType values.
	Type models.MessageType
	// Body is the HTML-formatted entry content.
	Body string
	// IsDeftOnly marks the entry as visible to DEFT members only.
	IsDeftOnly bool
}

// ListEntries returns visible entries for a ticket identified by slug.
// isDeftMember controls whether DEFT-only and draft/context entries are
// included. ownerID is the authenticated caller's user ID; it is used to
// include the caller's own draft entries even when isDeftMember is false.
func (s *Service) ListEntries(ctx context.Context, slug string, isDeftMember bool, ownerID string) ([]models.Message, error) {
	ticket, err := s.repo.FindTicketBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, nil
	}
	return s.repo.ListEntries(ctx, ticket.ID, isDeftMember, ownerID)
}

// CreateEntry adds a new entry to the ticket identified by slug.
// The entry type is validated against the caller's permissions:
//   - Non-DEFT members may only create customer entries.
//   - DEFT members may create any support entry type.
//
// Customer and system_event entries are immediately published and immutable.
// Agent_reply entries are immediately published and immutable.
// Draft and context entries are not published and remain mutable (drafts only).
func (s *Service) CreateEntry(
	ctx context.Context,
	slug, authorID string,
	isDeftMember bool,
	input CreateEntryInput,
) (*models.Message, error) {
	if !input.Type.IsSupportType() {
		return nil, fmt.Errorf("invalid entry type: %s", input.Type)
	}
	if input.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	// Permission check: non-DEFT callers may only add customer entries.
	if !isDeftMember && input.Type != models.MessageTypeCustomer {
		return nil, ErrForbidden
	}

	ticket, err := s.repo.FindTicketBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, nil
	}

	msg := &models.Message{
		ThreadID:   ticket.ID,
		Body:       input.Body,
		AuthorID:   authorID,
		Metadata:   "{}",
		Type:       input.Type,
		IsDeftOnly: input.IsDeftOnly,
	}

	switch input.Type {
	case models.MessageTypeCustomer, models.MessageTypeAgentReply, models.MessageTypeSystemEvent:
		// These types are published and immutable immediately on creation.
		msg.IsPublished = true
		msg.IsImmutable = true
		now := time.Now()
		msg.PublishedAt = &now
	case models.MessageTypeDraft:
		// Drafts are not published and remain mutable until promoted.
		msg.IsPublished = false
		msg.IsImmutable = false
	case models.MessageTypeContext:
		// Context notes are DEFT-only by definition and immutable once created.
		msg.IsDeftOnly = true
		msg.IsPublished = true
		msg.IsImmutable = true
		now := time.Now()
		msg.PublishedAt = &now
	}

	if err := s.repo.CreateEntry(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// UpdateEntryBody replaces the body of a mutable (draft) entry.
// Returns ErrImmutable when the entry cannot be edited.
// Returns nil, nil when the entry does not exist.
func (s *Service) UpdateEntryBody(ctx context.Context, entryID, body string) (*models.Message, error) {
	msg, err := s.repo.FindEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	if msg.IsImmutable {
		return nil, ErrImmutable
	}
	msg.Body = body
	if err := s.repo.UpdateEntry(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// PublishDraft promotes a draft entry to agent_reply, marks it published and
// immutable, and publishes a ticket-entry.published event on the bus.
// Returns ErrNotDraft when the entry is not a draft.
// Returns nil, nil when the entry does not exist.
func (s *Service) PublishDraft(ctx context.Context, entryID, publisherID string) (*models.Message, error) {
	msg, err := s.repo.FindEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	if msg.Type != models.MessageTypeDraft {
		return nil, ErrNotDraft
	}

	now := time.Now()
	msg.Type = models.MessageTypeAgentReply
	msg.IsPublished = true
	msg.IsImmutable = true
	msg.PublishedAt = &now

	if err := s.repo.UpdateEntry(ctx, msg); err != nil {
		return nil, err
	}

	// Notify downstream (notification trigger, etc.) — best effort.
	if s.bus != nil {
		payloadBytes, _ := json.Marshal(map[string]any{
			"thread_id":  msg.ThreadID,
			"entry_type": string(msg.Type),
		})
		s.bus.Publish(event.Event{
			Type:       "ticket-entry.published",
			EntityType: "message",
			EntityID:   msg.ID,
			UserID:     publisherID,
			Payload:    string(payloadBytes),
		})
	}
	return msg, nil
}

// SetDeftVisibility toggles the IsDeftOnly flag on an entry.
// Only DEFT members may call this. Returns ErrForbidden otherwise.
// Returns nil, nil when the entry does not exist.
func (s *Service) SetDeftVisibility(ctx context.Context, entryID string, isDeftOnly, isDeftMember bool) (*models.Message, error) {
	if !isDeftMember {
		return nil, ErrForbidden
	}
	msg, err := s.repo.FindEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	msg.IsDeftOnly = isDeftOnly
	if err := s.repo.UpdateEntry(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// SetNotificationDetailLevel stores the caller's preference for email
// notification detail on the given ticket. level must be "full" or "privacy".
// Returns nil when the ticket does not exist.
func (s *Service) SetNotificationDetailLevel(ctx context.Context, slug, level string) error {
	if level != "full" && level != "privacy" {
		return fmt.Errorf("notification_detail_level must be 'full' or 'privacy'")
	}
	ticket, err := s.repo.FindTicketBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if ticket == nil {
		return nil
	}
	patch := fmt.Sprintf(`{"notification_detail_level":%q}`, level)
	merged, err := metadata.DeepMerge(ticket.Metadata, patch)
	if err != nil {
		return fmt.Errorf("merging metadata: %w", err)
	}
	return s.repo.UpdateThreadMetadata(ctx, ticket.ID, merged)
}

// IsDeftMember delegates the DEFT-member check to the repository.
func (s *Service) IsDeftMember(ctx context.Context, userID string) (bool, error) {
	return s.repo.IsDeftMember(ctx, userID)
}
