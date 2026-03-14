package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// Gateway is the unified ChannelGateway service.
// It accepts InboundEvents from any channel adapter, routes each event to an
// existing thread or creates a new lead thread, creates a typed message, and
// publishes a domain event on the event bus.
type Gateway struct {
	db       *gorm.DB
	eventBus *eventbus.Bus
}

// NewGateway creates a new ChannelGateway.
func NewGateway(db *gorm.DB, eventBus *eventbus.Bus) *Gateway {
	return &Gateway{db: db, eventBus: eventBus}
}

// Process accepts an InboundEvent, routes it to the appropriate thread, creates a
// typed message, and publishes the result to the event bus.
// Returns an error when the event cannot be fully processed.
func (g *Gateway) Process(ctx context.Context, evt *InboundEvent) error {
	if evt == nil {
		return fmt.Errorf("event is required")
	}
	if evt.OrgID == "" {
		return fmt.Errorf("event.OrgID is required")
	}
	if !evt.ChannelType.IsValid() {
		return fmt.Errorf("invalid channel type: %s", evt.ChannelType)
	}

	// Assign ID if not already set.
	if evt.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generating event ID: %w", err)
		}
		evt.ID = id.String()
	}
	if evt.ReceivedAt.IsZero() {
		evt.ReceivedAt = time.Now()
	}
	if evt.Metadata == "" {
		evt.Metadata = "{}"
	}

	thread, err := g.resolveThread(ctx, evt)
	if err != nil {
		return fmt.Errorf("resolving thread: %w", err)
	}

	msgType := channelTypeToMessageType(evt.ChannelType)
	msgMeta := fmt.Sprintf(`{"channel_type":%q,"external_id":%q,"sender":%q,"event_id":%q}`,
		evt.ChannelType, evt.ExternalID, evt.SenderIdentifier, evt.ID)

	msg, err := g.createMessage(ctx, thread.ID, evt.Body, "system", msgType, msgMeta)
	if err != nil {
		return fmt.Errorf("creating message: %w", err)
	}

	if g.eventBus != nil {
		g.eventBus.Publish(eventbus.Event{
			Type:       "channel.inbound",
			EntityType: "message",
			EntityID:   msg.ID,
			Payload: map[string]any{
				"channel_type": string(evt.ChannelType),
				"thread_id":    thread.ID,
				"org_id":       evt.OrgID,
				"event_id":     evt.ID,
			},
		})
	}
	return nil
}

// resolveThread finds an existing thread that matches the event, or creates a new lead thread.
// Thread matching priority:
//  1. Match by ExternalID stored in thread metadata.
//  2. Match by SenderIdentifier (contact_email) in thread metadata.
//  3. No match — create a new lead thread.
func (g *Gateway) resolveThread(ctx context.Context, evt *InboundEvent) (*models.Thread, error) {
	if evt.ExternalID != "" {
		thread, err := g.findThreadByExternalID(ctx, evt.OrgID, evt.ExternalID)
		if err != nil {
			return nil, err
		}
		if thread != nil {
			return thread, nil
		}
	}

	if evt.SenderIdentifier != "" {
		thread, err := g.findThreadBySender(ctx, evt.OrgID, evt.SenderIdentifier)
		if err != nil {
			return nil, err
		}
		if thread != nil {
			return thread, nil
		}
	}

	return g.createLeadThread(ctx, evt)
}

// findThreadByExternalID looks for a thread where metadata.external_id equals externalID,
// scoped to the org via the board→space→org join chain.
func (g *Gateway) findThreadByExternalID(ctx context.Context, orgID, externalID string) (*models.Thread, error) {
	var thread models.Thread
	err := g.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = threads.board_id AND boards.deleted_at IS NULL").
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Joins("JOIN orgs ON orgs.id = spaces.org_id AND orgs.deleted_at IS NULL").
		Where("orgs.id = ?", orgID).
		Where("json_extract(threads.metadata, '$.external_id') = ?", externalID).
		Where("threads.deleted_at IS NULL").
		First(&thread).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread by external ID: %w", err)
	}
	return &thread, nil
}

// findThreadBySender looks for the most recent thread where metadata.contact_email equals sender,
// scoped to the org.
func (g *Gateway) findThreadBySender(ctx context.Context, orgID, sender string) (*models.Thread, error) {
	var thread models.Thread
	err := g.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = threads.board_id AND boards.deleted_at IS NULL").
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Joins("JOIN orgs ON orgs.id = spaces.org_id AND orgs.deleted_at IS NULL").
		Where("orgs.id = ?", orgID).
		Where("json_extract(threads.metadata, '$.contact_email') = ?", sender).
		Where("threads.deleted_at IS NULL").
		Order("threads.created_at DESC").
		First(&thread).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread by sender: %w", err)
	}
	return &thread, nil
}

// createLeadThread creates a new lead thread in the org's first CRM (or any) space/board.
func (g *Gateway) createLeadThread(ctx context.Context, evt *InboundEvent) (*models.Thread, error) {
	// Prefer a CRM space; fall back to any space.
	var space models.Space
	err := g.db.WithContext(ctx).
		Where("org_id = ? AND type = ? AND deleted_at IS NULL", evt.OrgID, models.SpaceTypeCRM).
		First(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = g.db.WithContext(ctx).
			Where("org_id = ? AND deleted_at IS NULL", evt.OrgID).
			First(&space).Error
	}
	if err != nil {
		return nil, fmt.Errorf("finding space for org %s: %w", evt.OrgID, err)
	}

	var board models.Board
	if err := g.db.WithContext(ctx).
		Where("space_id = ? AND deleted_at IS NULL", space.ID).
		First(&board).Error; err != nil {
		return nil, fmt.Errorf("finding board for space %s: %w", space.ID, err)
	}

	title := evt.Subject
	if title == "" {
		title = fmt.Sprintf("Inbound %s from %s", evt.ChannelType, evt.SenderIdentifier)
	}

	meta := fmt.Sprintf(
		`{"source":"inbound_channel","contact_email":%q,"external_id":%q,"channel_type":%q}`,
		evt.SenderIdentifier, evt.ExternalID, string(evt.ChannelType),
	)

	// Generate a short unique slug for the lead thread.
	slugID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating slug: %w", err)
	}
	slugStr := fmt.Sprintf("lead-%s", slugID.String()[:8])

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    title,
		Body:     evt.Body,
		Slug:     slugStr,
		Metadata: meta,
		AuthorID: "system",
	}
	if err := g.db.WithContext(ctx).Create(thread).Error; err != nil {
		return nil, fmt.Errorf("creating lead thread: %w", err)
	}
	return thread, nil
}

// createMessage inserts a new message on the given thread.
func (g *Gateway) createMessage(ctx context.Context, threadID, body, authorID string, msgType models.MessageType, metadata string) (*models.Message, error) {
	if body == "" {
		body = "[empty]"
	}
	msg := &models.Message{
		ThreadID: threadID,
		Body:     body,
		AuthorID: authorID,
		Type:     msgType,
		Metadata: metadata,
	}
	if err := g.db.WithContext(ctx).Create(msg).Error; err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}
	return msg, nil
}

// channelTypeToMessageType maps a ChannelType to the appropriate MessageType.
func channelTypeToMessageType(ct models.ChannelType) models.MessageType {
	switch ct {
	case models.ChannelTypeEmail:
		return models.MessageTypeEmail
	case models.ChannelTypeVoice:
		return models.MessageTypeCallLog
	case models.ChannelTypeChat:
		return models.MessageTypeComment
	default:
		return models.MessageTypeSystem
	}
}
