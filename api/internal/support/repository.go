// Package support provides support ticket entry management — creating, listing,
// publishing, and controlling visibility of the timeline entries that make up a
// support ticket conversation.
package support

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// SystemOrgID is the org ID used for ticket counters when a ticket is not
// scoped to a customer org (e.g. tier-2 users without an org).
const SystemOrgID = "_system"

// Repository handles data access for support entries and ticket counters.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new support Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// FindTicketBySlug returns the support thread identified by slug from the
// global-support board. Returns nil, nil when no such thread exists.
func (r *Repository) FindTicketBySlug(ctx context.Context, slug string) (*models.Thread, error) {
	var t models.Thread
	err := r.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = threads.board_id").
		Joins("JOIN spaces ON spaces.id = boards.space_id").
		Where("threads.slug = ? AND threads.thread_type = ? AND spaces.slug = ?",
			slug, models.ThreadTypeSupport, "global-support").
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding ticket: %w", err)
	}
	return &t, nil
}

// ListEntries returns all messages belonging to the given thread that pass the
// caller's visibility filter.
//   - DEFT members see every entry.
//   - Non-DEFT callers see: published customer messages, published agent replies,
//     system events (all with IsDeftOnly=false), plus their own draft entries
//     (authorID match) so they can review and send later.
func (r *Repository) ListEntries(ctx context.Context, threadID string, isDeftMember bool, ownerID string) ([]models.Message, error) {
	q := r.db.WithContext(ctx).Where("thread_id = ?", threadID)

	if !isDeftMember {
		// Visible public entries (published, not DEFT-only).
		q = q.Where(
			"(type IN ? AND is_published = ? AND is_deft_only = ?) OR (type = ? AND author_id = ?)",
			[]models.MessageType{
				models.MessageTypeCustomer,
				models.MessageTypeAgentReply,
				models.MessageTypeSystemEvent,
			},
			true, false,
			models.MessageTypeDraft, ownerID,
		)
	}

	var msgs []models.Message
	if err := q.Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, fmt.Errorf("listing entries: %w", err)
	}
	return msgs, nil
}

// CreateEntry inserts a new message entry into the database.
func (r *Repository) CreateEntry(ctx context.Context, msg *models.Message) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("creating entry: %w", err)
	}
	return nil
}

// FindEntry returns a single message by ID. Returns nil, nil when not found.
func (r *Repository) FindEntry(ctx context.Context, id string) (*models.Message, error) {
	var msg models.Message
	err := r.db.WithContext(ctx).First(&msg, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding entry: %w", err)
	}
	return &msg, nil
}

// UpdateEntry saves changes to an existing message entry.
func (r *Repository) UpdateEntry(ctx context.Context, msg *models.Message) error {
	if err := r.db.WithContext(ctx).Save(msg).Error; err != nil {
		return fmt.Errorf("updating entry: %w", err)
	}
	return nil
}

// NextTicketNumber atomically increments and returns the next ticket number for
// the given org and thread type. The increment is performed inside a transaction
// with an upsert so concurrent calls are serialised by SQLite.
func (r *Repository) NextTicketNumber(ctx context.Context, orgID, threadType string) (int64, error) {
	var next int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var counter models.TicketCounter
		result := tx.Where("org_id = ? AND thread_type = ?", orgID, threadType).
			First(&counter)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			counter = models.TicketCounter{
				OrgID:      orgID,
				ThreadType: threadType,
				LastNumber: 1,
			}
			if err := tx.Create(&counter).Error; err != nil {
				return fmt.Errorf("creating counter: %w", err)
			}
			next = 1
			return nil
		}
		if result.Error != nil {
			return fmt.Errorf("reading counter: %w", result.Error)
		}

		counter.LastNumber++
		if err := tx.Save(&counter).Error; err != nil {
			return fmt.Errorf("incrementing counter: %w", err)
		}
		next = counter.LastNumber
		return nil
	})
	if err != nil {
		return 0, err
	}
	return next, nil
}

// AssignTicketNumber sets TicketNumber on the given thread and saves it.
func (r *Repository) AssignTicketNumber(ctx context.Context, t *models.Thread, orgID string) error {
	n, err := r.NextTicketNumber(ctx, orgID, string(models.ThreadTypeSupport))
	if err != nil {
		return fmt.Errorf("assigning ticket number: %w", err)
	}
	t.TicketNumber = n
	if err := r.db.WithContext(ctx).
		Model(t).Update("ticket_number", n).Error; err != nil {
		return fmt.Errorf("saving ticket number: %w", err)
	}
	return nil
}

// UpdateThreadMetadata applies a metadata patch to the given thread using a
// targeted column update (does not touch other fields).
func (r *Repository) UpdateThreadMetadata(ctx context.Context, threadID, metadata string) error {
	if err := r.db.WithContext(ctx).
		Model(&models.Thread{}).
		Where("id = ?", threadID).
		Update("metadata", metadata).Error; err != nil {
		return fmt.Errorf("updating thread metadata: %w", err)
	}
	return nil
}

// DeftMemberInfo holds display info for a DEFT org member.
type DeftMemberInfo struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// ListDeftMembers returns all active members of the DEFT org (slug = "deft")
// enriched with display name and email from user_shadows.
func (r *Repository) ListDeftMembers(ctx context.Context) ([]DeftMemberInfo, error) {
	var results []DeftMemberInfo
	err := r.db.WithContext(ctx).
		Table("org_memberships").
		Select("org_memberships.user_id, COALESCE(user_shadows.display_name, '') AS display_name, COALESCE(user_shadows.email, '') AS email").
		Joins("JOIN orgs ON orgs.id = org_memberships.org_id").
		Joins("LEFT JOIN user_shadows ON user_shadows.clerk_user_id = org_memberships.user_id").
		Where("orgs.slug = ? AND org_memberships.deleted_at IS NULL", "deft").
		Order("display_name ASC").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("listing deft members: %w", err)
	}
	return results, nil
}

// IsDeftMember returns true when the user has active membership in any space
// whose org has the slug "deft", or is a platform admin.
func (r *Repository) IsDeftMember(ctx context.Context, userID string) (bool, error) {
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

	// Check org membership in the DEFT org (slug = "deft").
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
