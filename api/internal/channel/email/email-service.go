package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/channel"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
)

// Service orchestrates inbound email processing: parse, match thread,
// process attachments, build metadata, and normalize to InboundEvent.
type Service struct {
	db                *gorm.DB
	parser            func(msg *mail.Message) (*ParsedEmail, error)
	threadMatcher     *ThreadMatcher
	attachmentHandler *AttachmentHandler
	gateway           *channel.Gateway
}

// NewService creates a new email service.
func NewService(db *gorm.DB, storage upload.StorageProvider, gateway *channel.Gateway) *Service {
	return &Service{
		db:                db,
		parser:            ParseEmail,
		threadMatcher:     NewThreadMatcher(db),
		attachmentHandler: NewAttachmentHandler(db, storage),
		gateway:           gateway,
	}
}

// ProcessResult holds the result of processing an inbound email.
type ProcessResult struct {
	Thread     *models.Thread
	Message    *models.Message
	Uploads    []*models.Upload
	MatchBy    string
	IsNewLead  bool
	ParsedFrom string
}

// ProcessInbound handles a raw email message for an org: parses it,
// matches/creates thread, creates message, processes attachments, updates metadata.
func (s *Service) ProcessInbound(ctx context.Context, orgID string, msg *mail.Message) (*ProcessResult, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}
	if orgID == "" {
		return nil, fmt.Errorf("orgID is required")
	}

	// Parse the email.
	parsed, err := s.parser(msg)
	if err != nil {
		return nil, fmt.Errorf("parsing email: %w", err)
	}

	// Match to thread.
	matchResult, err := s.threadMatcher.Match(ctx, orgID, parsed)
	if err != nil {
		return nil, fmt.Errorf("matching thread: %w", err)
	}

	// Generate event ID.
	eventID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating event ID: %w", err)
	}

	// Build message metadata.
	msgMeta := BuildMessageMetadata(parsed, eventID.String())

	// Create the message on the thread.
	body := parsed.Body
	if body == "" {
		body = "[empty]"
	}
	message := &models.Message{
		ThreadID: matchResult.Thread.ID,
		Body:     body,
		AuthorID: "system",
		Type:     models.MessageTypeEmail,
		Metadata: msgMeta,
	}
	if err := s.db.WithContext(ctx).Create(message).Error; err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}

	// Process attachments.
	var uploads []*models.Upload
	if len(parsed.Attachments) > 0 {
		uploads, err = s.attachmentHandler.ProcessAttachments(ctx, orgID, message.ID, parsed.Attachments)
		if err != nil {
			// Log but don't fail the entire process for attachment errors.
			_ = err
		}
	}

	// Update thread metadata with the new message ID.
	if parsed.MessageID != "" {
		if err := s.threadMatcher.AppendMessageID(ctx, matchResult.Thread, parsed.MessageID); err != nil {
			// Non-fatal: log the error but don't fail the process.
			_ = err
		}
	}

	return &ProcessResult{
		Thread:     matchResult.Thread,
		Message:    message,
		Uploads:    uploads,
		MatchBy:    matchResult.MatchBy,
		IsNewLead:  matchResult.IsNew,
		ParsedFrom: parsed.From,
	}, nil
}

// Normalize implements the channel.Normalizer interface.
// It converts raw email bytes (in RFC 5322 format) for an org into an InboundEvent.
func (s *Service) Normalize(orgID string, raw []byte) (*channel.InboundEvent, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty email data")
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("reading email message: %w", err)
	}

	parsed, err := s.parser(msg)
	if err != nil {
		return nil, fmt.Errorf("parsing email: %w", err)
	}

	// Build attachments list.
	attachments := make([]channel.AttachmentRef, 0, len(parsed.Attachments))
	for _, att := range parsed.Attachments {
		attachments = append(attachments, channel.AttachmentRef{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        int64(len(att.Data)),
		})
	}

	// Build metadata JSON.
	meta := map[string]any{
		"message_id":  parsed.MessageID,
		"in_reply_to": parsed.InReplyTo,
		"references":  parsed.References,
		"from":        parsed.From,
		"to":          parsed.To,
		"cc":          parsed.CC,
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		metaBytes = []byte("{}")
	}

	return &channel.InboundEvent{
		ChannelType:      models.ChannelTypeEmail,
		OrgID:            orgID,
		ExternalID:       parsed.MessageID,
		SenderIdentifier: parsed.From,
		Subject:          parsed.Subject,
		Body:             parsed.Body,
		Metadata:         string(metaBytes),
		Attachments:      attachments,
		ReceivedAt:       time.Now(),
	}, nil
}
