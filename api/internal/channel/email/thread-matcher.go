package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// ThreadMatcher resolves inbound emails to existing CRM threads or creates new leads.
// Matching priority:
//  1. In-Reply-To/References match against Message-IDs stored in thread metadata
//  2. Sender email match against existing lead threads (contact_email)
//  3. No match — create new lead thread
type ThreadMatcher struct {
	db *gorm.DB
}

// NewThreadMatcher creates a new thread matcher.
func NewThreadMatcher(db *gorm.DB) *ThreadMatcher {
	return &ThreadMatcher{db: db}
}

// MatchResult describes how a thread was resolved.
type MatchResult struct {
	Thread   *models.Thread
	IsNew    bool
	MatchBy  string // "message_id", "sender_email", or "new_lead"
	MatchRef string // the reference that matched (message ID or email)
}

// Match finds or creates a thread for the given parsed email within the org.
// routingAction controls which space type is targeted when creating a new thread.
func (m *ThreadMatcher) Match(ctx context.Context, orgID string, parsed *ParsedEmail, routingAction models.RoutingAction) (*MatchResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("parsed email is nil")
	}

	// Strategy 1: Match by In-Reply-To or References against stored Message-IDs.
	messageIDs := make([]string, 0)
	if parsed.InReplyTo != "" {
		messageIDs = append(messageIDs, parsed.InReplyTo)
	}
	messageIDs = append(messageIDs, parsed.References...)

	for _, msgID := range messageIDs {
		thread, err := m.findThreadByMessageID(ctx, orgID, msgID)
		if err != nil {
			return nil, err
		}
		if thread != nil {
			return &MatchResult{
				Thread:   thread,
				IsNew:    false,
				MatchBy:  "message_id",
				MatchRef: msgID,
			}, nil
		}
	}

	// Strategy 2: Fallback to sender email match.
	if parsed.From != "" {
		thread, err := m.findThreadBySenderEmail(ctx, orgID, parsed.From)
		if err != nil {
			return nil, err
		}
		if thread != nil {
			return &MatchResult{
				Thread:   thread,
				IsNew:    false,
				MatchBy:  "sender_email",
				MatchRef: parsed.From,
			}, nil
		}
	}

	// Strategy 3: No match — create new lead thread routed by routingAction.
	thread, err := m.createLeadThread(ctx, orgID, parsed, routingAction)
	if err != nil {
		return nil, err
	}
	return &MatchResult{
		Thread:  thread,
		IsNew:   true,
		MatchBy: "new_lead",
	}, nil
}

// findThreadByMessageID looks for a thread where the metadata message_ids array
// contains the given message ID, scoped to the org.
func (m *ThreadMatcher) findThreadByMessageID(ctx context.Context, orgID, messageID string) (*models.Thread, error) {
	var thread models.Thread

	// Use json_extract to search the message_ids array in thread metadata.
	// Also search for direct message_id match for backwards compatibility.
	err := m.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = threads.board_id AND boards.deleted_at IS NULL").
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Joins("JOIN orgs ON orgs.id = spaces.org_id AND orgs.deleted_at IS NULL").
		Where("orgs.id = ?", orgID).
		Where("threads.deleted_at IS NULL").
		Where(`(
			json_extract(threads.metadata, '$.message_ids') LIKE ?
			OR json_extract(threads.metadata, '$.message_id') = ?
		)`, "%"+messageID+"%", messageID).
		Order("threads.created_at DESC").
		First(&thread).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread by message ID: %w", err)
	}
	return &thread, nil
}

// findThreadBySenderEmail looks for threads with contact_email or email_address
// matching the sender, scoped to the org.
func (m *ThreadMatcher) findThreadBySenderEmail(ctx context.Context, orgID, senderEmail string) (*models.Thread, error) {
	var thread models.Thread
	err := m.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = threads.board_id AND boards.deleted_at IS NULL").
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Joins("JOIN orgs ON orgs.id = spaces.org_id AND orgs.deleted_at IS NULL").
		Where("orgs.id = ?", orgID).
		Where("threads.deleted_at IS NULL").
		Where(`(
			json_extract(threads.metadata, '$.contact_email') = ?
			OR json_extract(threads.metadata, '$.email_address') = ?
		)`, senderEmail, senderEmail).
		Order("threads.created_at DESC").
		First(&thread).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread by sender email: %w", err)
	}
	return &thread, nil
}

// routingActionToSpaceType maps a RoutingAction to the preferred SpaceType.
func routingActionToSpaceType(action models.RoutingAction) models.SpaceType {
	switch action {
	case models.RoutingActionSupportTicket:
		return models.SpaceTypeSupport
	case models.RoutingActionGeneral:
		return models.SpaceTypeGeneral
	default: // RoutingActionSalesLead and unknown values
		return models.SpaceTypeCRM
	}
}

// createLeadThread creates a new thread in the space matching routingAction,
// falling back to any available space when the preferred type has no board.
func (m *ThreadMatcher) createLeadThread(ctx context.Context, orgID string, parsed *ParsedEmail, routingAction models.RoutingAction) (*models.Thread, error) {
	preferred := routingActionToSpaceType(routingAction)

	// Try preferred space type first, then any space as fallback.
	var space models.Space
	err := m.db.WithContext(ctx).
		Where("org_id = ? AND type = ? AND deleted_at IS NULL", orgID, preferred).
		First(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = m.db.WithContext(ctx).
			Where("org_id = ? AND deleted_at IS NULL", orgID).
			First(&space).Error
	}
	if err != nil {
		return nil, fmt.Errorf("finding space for org %s: %w", orgID, err)
	}

	var board models.Board
	if err := m.db.WithContext(ctx).
		Where("space_id = ? AND deleted_at IS NULL", space.ID).
		First(&board).Error; err != nil {
		return nil, fmt.Errorf("finding board for space %s: %w", space.ID, err)
	}

	title := parsed.Subject
	if title == "" {
		title = fmt.Sprintf("Inbound email from %s", parsed.From)
	}

	// Build thread metadata with email-specific fields.
	meta := map[string]any{
		"source":        "inbound_email",
		"contact_email": parsed.From,
		"email_address": parsed.From,
		"channel_type":  "email",
	}
	if parsed.MessageID != "" {
		meta["message_ids"] = []string{parsed.MessageID}
		meta["message_id"] = parsed.MessageID
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling thread metadata: %w", err)
	}

	slugID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating slug: %w", err)
	}

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    title,
		Body:     parsed.Body,
		Slug:     fmt.Sprintf("lead-%s", slugID.String()[:8]),
		Metadata: string(metaBytes),
		AuthorID: "system",
	}
	if err := m.db.WithContext(ctx).Create(thread).Error; err != nil {
		return nil, fmt.Errorf("creating lead thread: %w", err)
	}
	return thread, nil
}

// AppendMessageID adds a message ID to the thread's metadata message_ids array.
func (m *ThreadMatcher) AppendMessageID(ctx context.Context, thread *models.Thread, messageID string) error {
	if messageID == "" {
		return nil
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(thread.Metadata), &meta); err != nil {
		meta = make(map[string]any)
	}

	// Get existing message_ids array.
	var messageIDs []string
	if existing, ok := meta["message_ids"]; ok {
		if arr, ok := existing.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					messageIDs = append(messageIDs, s)
				}
			}
		}
	}

	// Check for duplicate.
	for _, id := range messageIDs {
		if id == messageID {
			return nil
		}
	}

	messageIDs = append(messageIDs, messageID)
	meta["message_ids"] = messageIDs

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	thread.Metadata = string(metaBytes)
	if err := m.db.WithContext(ctx).Model(thread).Update("metadata", thread.Metadata).Error; err != nil {
		return fmt.Errorf("updating thread metadata: %w", err)
	}
	return nil
}
