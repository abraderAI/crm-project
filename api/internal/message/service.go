package message

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Service provides business logic for Message operations.
type Service struct {
	repo *Repository
}

// NewService creates a new Message service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the data needed to create a Message.
type CreateInput struct {
	Body     string             `json:"body"`
	Metadata string             `json:"metadata"`
	Type     models.MessageType `json:"type"`
}

// Create validates input and creates a new Message.
// Returns an error if the parent thread is locked.
func (s *Service) Create(ctx context.Context, threadID, authorID string, threadLocked bool, input CreateInput) (*models.Message, error) {
	if threadLocked {
		return nil, fmt.Errorf("thread is locked")
	}
	if input.Body == "" {
		return nil, fmt.Errorf("body is required")
	}
	if input.Type == "" {
		input.Type = models.MessageTypeComment
	}
	if !input.Type.IsValid() {
		return nil, fmt.Errorf("invalid message type: %s", input.Type)
	}
	if input.Metadata == "" {
		input.Metadata = "{}"
	}

	msg := &models.Message{
		ThreadID: threadID,
		Body:     input.Body,
		AuthorID: authorID,
		Metadata: input.Metadata,
		Type:     input.Type,
	}
	if err := s.repo.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// Get retrieves a Message by ID within a thread.
func (s *Service) Get(ctx context.Context, threadID, id string) (*models.Message, error) {
	return s.repo.FindByID(ctx, threadID, id)
}

// List returns a paginated list of Messages within a thread.
func (s *Service) List(ctx context.Context, threadID string, params pagination.Params) ([]models.Message, *pagination.PageInfo, error) {
	return s.repo.List(ctx, threadID, params)
}

// UpdateInput holds partial update data for a Message.
type UpdateInput struct {
	Body *string `json:"body"`
}

// Update applies partial updates to a Message (author-only, creates revision).
func (s *Service) Update(ctx context.Context, threadID, msgID, editorID string, input UpdateInput) (*models.Message, error) {
	msg, err := s.repo.FindByID(ctx, threadID, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}

	// Author-only update check.
	if msg.AuthorID != editorID {
		return nil, fmt.Errorf("only the author can update this message")
	}

	// Create revision before update.
	prevContent := map[string]string{
		"body": msg.Body,
	}
	prevJSON, _ := json.Marshal(prevContent)

	if input.Body != nil {
		msg.Body = *input.Body
	}

	if err := s.repo.Update(ctx, msg); err != nil {
		return nil, err
	}

	// Save revision.
	rev := &models.Revision{
		EntityType:      "message",
		EntityID:        msg.ID,
		PreviousContent: string(prevJSON),
		EditorID:        editorID,
	}
	_ = s.repo.CreateRevision(ctx, rev)

	return msg, nil
}

// Delete soft-deletes a Message.
func (s *Service) Delete(ctx context.Context, threadID, msgID string) error {
	msg, err := s.repo.FindByID(ctx, threadID, msgID)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("not found")
	}
	return s.repo.Delete(ctx, msg.ID)
}
