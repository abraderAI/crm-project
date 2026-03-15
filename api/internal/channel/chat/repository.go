package chat

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Repository handles database operations for chat sessions and visitors.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new chat Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateSession inserts a new ChatSession.
func (r *Repository) CreateSession(ctx context.Context, session *ChatSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("creating chat session: %w", err)
	}
	return nil
}

// FindSession retrieves a ChatSession by its ID.
// Returns nil, nil when no record exists.
func (r *Repository) FindSession(ctx context.Context, id string) (*ChatSession, error) {
	var session ChatSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding chat session: %w", err)
	}
	return &session, nil
}

// UpdateSession saves changes to an existing ChatSession.
func (r *Repository) UpdateSession(ctx context.Context, session *ChatSession) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("updating chat session: %w", err)
	}
	return nil
}

// FindOrCreateVisitor finds an existing visitor by org+fingerprint or creates a new one.
func (r *Repository) FindOrCreateVisitor(ctx context.Context, orgID, fingerprintHash string) (*ChatVisitor, bool, error) {
	var visitor ChatVisitor
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND fingerprint_hash = ?", orgID, fingerprintHash).
		First(&visitor).Error
	if err == nil {
		return &visitor, false, nil // Existing visitor.
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("finding visitor: %w", err)
	}

	// Create new visitor.
	visitor = ChatVisitor{
		OrgID:           orgID,
		FingerprintHash: fingerprintHash,
	}
	if err := r.db.WithContext(ctx).Create(&visitor).Error; err != nil {
		return nil, false, fmt.Errorf("creating visitor: %w", err)
	}
	return &visitor, true, nil
}

// UpdateVisitor saves changes to an existing ChatVisitor.
func (r *Repository) UpdateVisitor(ctx context.Context, visitor *ChatVisitor) error {
	if err := r.db.WithContext(ctx).Save(visitor).Error; err != nil {
		return fmt.Errorf("updating visitor: %w", err)
	}
	return nil
}

// FindVisitor retrieves a ChatVisitor by ID.
func (r *Repository) FindVisitor(ctx context.Context, id string) (*ChatVisitor, error) {
	var visitor ChatVisitor
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&visitor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding visitor: %w", err)
	}
	return &visitor, nil
}

// FindChannelConfigByEmbedKey retrieves an enabled chat channel config by embed key.
// Returns nil, nil when no matching config exists.
func (r *Repository) FindChannelConfigByEmbedKey(ctx context.Context, embedKey string) (*models.ChannelConfig, error) {
	var cfg models.ChannelConfig
	err := r.db.WithContext(ctx).
		Where("channel_type = ? AND enabled = ? AND settings LIKE ?",
			models.ChannelTypeChat, true, fmt.Sprintf("%%\"embed_key\":\"%s\"%%", embedKey)).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding channel config by embed key: %w", err)
	}
	return &cfg, nil
}

// FindThreadByVisitor looks for the most recent thread associated with a visitor
// in the given org, scoped through board→space→org joins.
func (r *Repository) FindThreadByVisitor(ctx context.Context, orgID, visitorID string) (*models.Thread, error) {
	// Look up the visitor to get last_thread_id.
	var visitor ChatVisitor
	err := r.db.WithContext(ctx).Where("id = ? AND org_id = ?", visitorID, orgID).First(&visitor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding visitor for thread lookup: %w", err)
	}
	if visitor.LastThreadID == "" {
		return nil, nil
	}

	var thread models.Thread
	err = r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", visitor.LastThreadID).First(&thread).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread for visitor: %w", err)
	}
	return &thread, nil
}

// CreateThread creates a new CRM thread.
func (r *Repository) CreateThread(ctx context.Context, thread *models.Thread) error {
	if err := r.db.WithContext(ctx).Create(thread).Error; err != nil {
		return fmt.Errorf("creating thread: %w", err)
	}
	return nil
}

// CreateMessage creates a new message on a thread.
func (r *Repository) CreateMessage(ctx context.Context, msg *models.Message) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("creating message: %w", err)
	}
	return nil
}

// ListThreadMessages retrieves all messages for a thread, ordered by creation time.
func (r *Repository) ListThreadMessages(ctx context.Context, threadID string) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Where("thread_id = ?", threadID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("listing thread messages: %w", err)
	}
	return messages, nil
}

// UpdateThreadMetadata updates the metadata field of a thread.
func (r *Repository) UpdateThreadMetadata(ctx context.Context, threadID, metadata string) error {
	err := r.db.WithContext(ctx).
		Model(&models.Thread{}).
		Where("id = ?", threadID).
		Update("metadata", metadata).Error
	if err != nil {
		return fmt.Errorf("updating thread metadata: %w", err)
	}
	return nil
}

// FindFirstBoardInOrg finds the first board in the org's CRM space (or any space).
func (r *Repository) FindFirstBoardInOrg(ctx context.Context, orgID string) (*models.Board, error) {
	var board models.Board
	// Prefer a CRM space.
	err := r.db.WithContext(ctx).
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.org_id = ? AND spaces.type = ? AND boards.deleted_at IS NULL",
			orgID, models.SpaceTypeCRM).
		First(&board).Error
	if err == nil {
		return &board, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("finding CRM board: %w", err)
	}

	// Fall back to any space.
	err = r.db.WithContext(ctx).
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.org_id = ? AND boards.deleted_at IS NULL", orgID).
		First(&board).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding any board: %w", err)
	}
	return &board, nil
}
