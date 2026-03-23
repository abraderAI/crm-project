// Package globalspace provides HTTP handlers for the /global-spaces API endpoints.
// These endpoints give authenticated users access to platform-wide spaces such as
// global-support and global-forum without needing to know the underlying org/board IDs.
package globalspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for global space threads.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new global space Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// FindDefaultBoard returns the first non-deleted board in the global space with the
// given slug. Returns nil, nil when no board exists.
func (r *Repository) FindDefaultBoard(ctx context.Context, spaceSlug string) (*models.Board, error) {
	var board models.Board
	err := r.db.WithContext(ctx).
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.slug = ? AND boards.deleted_at IS NULL", spaceSlug).
		First(&board).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding default board for %s: %w", spaceSlug, err)
	}
	return &board, nil
}

// ListParams holds pagination and scoping options for listing global space threads.
type ListParams struct {
	pagination.Params
	// AuthorID, when non-empty, restricts results to threads authored by this user.
	AuthorID string
	// OrgID, when non-empty, restricts results to threads belonging to this org.
	OrgID string
	// VisibleOrgIDs enforces org-tier visibility: only threads with org_id IN this
	// set are returned. Applied when the caller is an org member (ScopeOrg).
	VisibleOrgIDs []string
	// VisibleUserID enforces owner-tier visibility: only threads authored by or
	// assigned to this user are returned. Applied when the caller has no org (ScopeOwner).
	VisibleUserID string
}

// ListThreads returns a paginated list of threads in boardID, filtered by ListParams.
func (r *Repository) ListThreads(ctx context.Context, boardID string, params ListParams) ([]models.Thread, *pagination.PageInfo, error) {
	var threads []models.Thread
	query := r.db.WithContext(ctx).Where("board_id = ?", boardID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}
	if params.AuthorID != "" {
		query = query.Where("author_id = ?", params.AuthorID)
	}
	if params.OrgID != "" {
		query = query.Where("org_id = ?", params.OrgID)
	}

	// Visibility enforcement filters.
	if len(params.VisibleOrgIDs) > 0 {
		query = query.Where("org_id IN ?", params.VisibleOrgIDs)
	}
	if params.VisibleUserID != "" {
		query = query.Where("(author_id = ? OR assigned_to = ?)", params.VisibleUserID, params.VisibleUserID)
	}

	if err := query.Limit(params.Limit + 1).Find(&threads).Error; err != nil {
		return nil, nil, fmt.Errorf("listing global space threads: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(threads) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(threads[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		threads = threads[:params.Limit]
	}
	return threads, pageInfo, nil
}

// CreateThread inserts a new thread record.
func (r *Repository) CreateThread(ctx context.Context, thread *models.Thread) error {
	if err := r.db.WithContext(ctx).Create(thread).Error; err != nil {
		return fmt.Errorf("creating global space thread: %w", err)
	}
	return nil
}

// CreateThreadWithInitialEntry creates a thread and optionally creates an
// initial message entry in the same transaction.
func (r *Repository) CreateThreadWithInitialEntry(ctx context.Context, thread *models.Thread, initialEntry *models.Message) error {
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(thread).Error; err != nil {
			return fmt.Errorf("creating global space thread: %w", err)
		}
		if initialEntry != nil {
			initialEntry.ThreadID = thread.ID
			if err := tx.Create(initialEntry).Error; err != nil {
				return fmt.Errorf("creating initial thread entry: %w", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// SlugExistsInBoard reports whether a thread slug already exists in boardID.
func (r *Repository) SlugExistsInBoard(ctx context.Context, boardID, threadSlug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Thread{}).
		Where("board_id = ? AND slug = ?", boardID, threadSlug).Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug: %w", err)
	}
	return count > 0, nil
}

// FindThreadBySlug returns the thread matching slug in boardID, or nil if not found.
func (r *Repository) FindThreadBySlug(ctx context.Context, boardID, threadSlug string) (*models.Thread, error) {
	var t models.Thread
	err := r.db.WithContext(ctx).
		Where("board_id = ? AND slug = ? AND deleted_at IS NULL", boardID, threadSlug).
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread by slug: %w", err)
	}
	return &t, nil
}

// UpdateThread persists changes to an existing thread record.
func (r *Repository) UpdateThread(ctx context.Context, t *models.Thread) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		return fmt.Errorf("updating global space thread: %w", err)
	}
	return nil
}

// GetUserShadowsByIDs returns a map of Clerk user ID → UserShadow for the given IDs.
// Missing entries are silently omitted.
func (r *Repository) GetUserShadowsByIDs(ctx context.Context, ids []string) (map[string]models.UserShadow, error) {
	if len(ids) == 0 {
		return map[string]models.UserShadow{}, nil
	}
	var shadows []models.UserShadow
	if err := r.db.WithContext(ctx).
		Where("clerk_user_id IN ?", ids).
		Find(&shadows).Error; err != nil {
		return nil, fmt.Errorf("fetching user shadows: %w", err)
	}
	result := make(map[string]models.UserShadow, len(shadows))
	for _, s := range shadows {
		result[s.ClerkUserID] = s
	}
	return result, nil
}

// CreateRevision stores a revision record for audit tracking.
func (r *Repository) CreateRevision(ctx context.Context, rev *models.Revision) error {
	if err := r.db.WithContext(ctx).Create(rev).Error; err != nil {
		return fmt.Errorf("creating revision: %w", err)
	}
	return nil
}

// ListMessages returns paginated messages for a thread.
func (r *Repository) ListMessages(ctx context.Context, threadID string, params pagination.Params) ([]models.Message, *pagination.PageInfo, error) {
	var messages []models.Message
	query := r.db.WithContext(ctx).Where("thread_id = ? AND deleted_at IS NULL", threadID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&messages).Error; err != nil {
		return nil, nil, fmt.Errorf("listing messages: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(messages) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(messages[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		messages = messages[:params.Limit]
	}
	return messages, pageInfo, nil
}

// CreateMessage inserts a new message record.
func (r *Repository) CreateMessage(ctx context.Context, msg *models.Message) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("creating message: %w", err)
	}
	return nil
}

// ListUploadsByThread returns all non-deleted uploads attached to the given thread ID.
func (r *Repository) ListUploadsByThread(ctx context.Context, threadID string) ([]models.Upload, error) {
	var uploads []models.Upload
	if err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND deleted_at IS NULL", "thread", threadID).
		Order("created_at ASC").
		Find(&uploads).Error; err != nil {
		return nil, fmt.Errorf("listing thread uploads: %w", err)
	}
	return uploads, nil
}

// FindSystemOrgID returns the ID of the _system org used for platform-level entity scoping.
// Returns an error if the org does not exist.
func (r *Repository) FindSystemOrgID(ctx context.Context) (string, error) {
	var org models.Org
	if err := r.db.WithContext(ctx).Select("id").Where("slug = ?", "_system").First(&org).Error; err != nil {
		return "", fmt.Errorf("finding system org: %w", err)
	}
	return org.ID, nil
}

// FindUserOrgIDs returns the IDs of all orgs the user is an active member of.
// An empty slice is returned when the user has no org memberships.
func (r *Repository) FindUserOrgIDs(ctx context.Context, userID string) ([]string, error) {
	var orgIDs []string
	err := r.db.WithContext(ctx).
		Model(&models.OrgMembership{}).
		Select("org_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Pluck("org_id", &orgIDs).Error
	if err != nil {
		return nil, fmt.Errorf("finding user org IDs: %w", err)
	}
	return orgIDs, nil
}

// IsDeftOrAdmin returns true when the user is a platform admin or has active
// membership in the DEFT org (slug = "deft"). This determines top-tier
// visibility for support ticket scoping.
func (r *Repository) IsDeftOrAdmin(ctx context.Context, userID string) (bool, error) {
	// Check platform admin first.
	var adminCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.PlatformAdmin{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Count(&adminCount).Error; err != nil {
		return false, fmt.Errorf("checking platform admin: %w", err)
	}
	if adminCount > 0 {
		return true, nil
	}

	// Check org membership in the DEFT org.
	var memberCount int64
	err := r.db.WithContext(ctx).
		Model(&models.OrgMembership{}).
		Joins("JOIN orgs ON orgs.id = org_memberships.org_id").
		Where("org_memberships.user_id = ? AND orgs.slug = ? AND org_memberships.deleted_at IS NULL",
			userID, "deft").
		Count(&memberCount).Error
	if err != nil {
		return false, fmt.Errorf("checking deft membership: %w", err)
	}
	return memberCount > 0, nil
}

// GetOrgNamesByIDs returns a map of org ID → org name for the given IDs.
// Missing entries are silently omitted.
func (r *Repository) GetOrgNamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	var orgs []models.Org
	if err := r.db.WithContext(ctx).
		Select("id, name").
		Where("id IN ?", ids).
		Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("fetching org names: %w", err)
	}
	result := make(map[string]string, len(orgs))
	for _, o := range orgs {
		result[o.ID] = o.Name
	}
	return result, nil
}
