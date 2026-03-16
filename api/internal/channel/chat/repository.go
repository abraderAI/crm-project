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

// CreateOrUpdateLead creates a new lead record or updates an existing one identified by visitor ID.
// This ensures duplicate sessions do not create duplicate leads.
func (r *Repository) CreateOrUpdateLead(ctx context.Context, visitorID, email, name, source string) (*models.Lead, error) {
	anonSessionID := visitorID // Use visitor ID as the anon session identifier.
	var lead models.Lead
	err := r.db.WithContext(ctx).Where("anon_session_id = ?", anonSessionID).First(&lead).Error
	if err == nil {
		// Update existing lead.
		if email != "" && lead.Email == "" {
			lead.Email = email
		}
		if name != "" && lead.Name == "" {
			lead.Name = name
		}
		if err := r.db.WithContext(ctx).Save(&lead).Error; err != nil {
			return nil, fmt.Errorf("updating lead: %w", err)
		}
		return &lead, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("finding lead: %w", err)
	}

	// Create new lead.
	lead = models.Lead{
		Email:         email,
		Name:          name,
		Source:        source,
		Status:        models.LeadStatusAnonymous,
		AnonSessionID: &anonSessionID,
		Metadata:      "{}",
	}
	if err := r.db.WithContext(ctx).Create(&lead).Error; err != nil {
		return nil, fmt.Errorf("creating lead: %w", err)
	}
	return &lead, nil
}

// FindLeadByAnonSession retrieves a lead by its anonymous session ID.
// Returns nil, nil when no record exists.
func (r *Repository) FindLeadByAnonSession(ctx context.Context, anonSessionID string) (*models.Lead, error) {
	var lead models.Lead
	err := r.db.WithContext(ctx).Where("anon_session_id = ?", anonSessionID).First(&lead).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding lead by anon session: %w", err)
	}
	return &lead, nil
}

// PromoteAnonymousSession links an anonymous session's lead record to a registered user.
// It sets the user_id and changes the status from anonymous to registered.
func (r *Repository) PromoteAnonymousSession(ctx context.Context, anonSessionID, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Lead{}).
		Where("anon_session_id = ? AND status = ?", anonSessionID, models.LeadStatusAnonymous).
		Updates(map[string]any{
			"user_id": userID,
			"status":  models.LeadStatusRegistered,
		})
	if result.Error != nil {
		return fmt.Errorf("promoting anonymous session: %w", result.Error)
	}
	return nil
}

// FindGlobalSupportBoard retrieves the first board in the global-support space.
// Returns nil, nil when no board exists.
func (r *Repository) FindGlobalSupportBoard(ctx context.Context) (*models.Board, error) {
	var board models.Board
	err := r.db.WithContext(ctx).
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.slug = ? AND boards.deleted_at IS NULL", "global-support").
		First(&board).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding global-support board: %w", err)
	}
	return &board, nil
}

// SearchGlobalDocs performs a text search over threads in the global-docs space.
// Returns up to `limit` matching threads ordered by relevance.
func (r *Repository) SearchGlobalDocs(ctx context.Context, query string, limit int) ([]models.Thread, error) {
	if query == "" || limit <= 0 {
		return nil, nil
	}

	var threads []models.Thread
	likeQuery := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = threads.board_id AND boards.deleted_at IS NULL").
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.slug = ? AND threads.deleted_at IS NULL AND (threads.title LIKE ? OR threads.body LIKE ?)",
			"global-docs", likeQuery, likeQuery).
		Order("threads.created_at DESC").
		Limit(limit).
		Find(&threads).Error
	if err != nil {
		return nil, fmt.Errorf("searching global-docs: %w", err)
	}
	return threads, nil
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
