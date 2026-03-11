// Package membership provides Membership CRUD at org/space/board levels.
package membership

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Repository handles database operations for memberships.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Membership repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- Org Memberships ---

// AddOrgMember creates an org membership.
func (r *Repository) AddOrgMember(ctx context.Context, m *models.OrgMembership) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("adding org member: %w", err)
	}
	return nil
}

// GetOrgMember retrieves an org membership by org and user ID.
func (r *Repository) GetOrgMember(ctx context.Context, orgID, userID string) (*models.OrgMembership, error) {
	var m models.OrgMembership
	if err := r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting org member: %w", err)
	}
	return &m, nil
}

// ListOrgMembers returns all members of an org.
func (r *Repository) ListOrgMembers(ctx context.Context, orgID string) ([]models.OrgMembership, error) {
	var members []models.OrgMembership
	if err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("listing org members: %w", err)
	}
	return members, nil
}

// UpdateOrgMember updates an org membership role.
func (r *Repository) UpdateOrgMember(ctx context.Context, m *models.OrgMembership) error {
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating org member: %w", err)
	}
	return nil
}

// RemoveOrgMember soft-deletes an org membership.
func (r *Repository) RemoveOrgMember(ctx context.Context, orgID, userID string) error {
	result := r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&models.OrgMembership{})
	if result.Error != nil {
		return fmt.Errorf("removing org member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CountOrgOwners counts the number of owners in an org.
func (r *Repository) CountOrgOwners(ctx context.Context, orgID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.OrgMembership{}).
		Where("org_id = ? AND role = ?", orgID, models.RoleOwner).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting org owners: %w", err)
	}
	return count, nil
}

// --- Space Memberships ---

// AddSpaceMember creates a space membership.
func (r *Repository) AddSpaceMember(ctx context.Context, m *models.SpaceMembership) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("adding space member: %w", err)
	}
	return nil
}

// GetSpaceMember retrieves a space membership.
func (r *Repository) GetSpaceMember(ctx context.Context, spaceID, userID string) (*models.SpaceMembership, error) {
	var m models.SpaceMembership
	if err := r.db.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting space member: %w", err)
	}
	return &m, nil
}

// ListSpaceMembers returns all members of a space.
func (r *Repository) ListSpaceMembers(ctx context.Context, spaceID string) ([]models.SpaceMembership, error) {
	var members []models.SpaceMembership
	if err := r.db.WithContext(ctx).Where("space_id = ?", spaceID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("listing space members: %w", err)
	}
	return members, nil
}

// UpdateSpaceMember updates a space membership role.
func (r *Repository) UpdateSpaceMember(ctx context.Context, m *models.SpaceMembership) error {
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating space member: %w", err)
	}
	return nil
}

// RemoveSpaceMember soft-deletes a space membership.
func (r *Repository) RemoveSpaceMember(ctx context.Context, spaceID, userID string) error {
	result := r.db.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).Delete(&models.SpaceMembership{})
	if result.Error != nil {
		return fmt.Errorf("removing space member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// --- Board Memberships ---

// AddBoardMember creates a board membership.
func (r *Repository) AddBoardMember(ctx context.Context, m *models.BoardMembership) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("adding board member: %w", err)
	}
	return nil
}

// GetBoardMember retrieves a board membership.
func (r *Repository) GetBoardMember(ctx context.Context, boardID, userID string) (*models.BoardMembership, error) {
	var m models.BoardMembership
	if err := r.db.WithContext(ctx).Where("board_id = ? AND user_id = ?", boardID, userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting board member: %w", err)
	}
	return &m, nil
}

// ListBoardMembers returns all members of a board.
func (r *Repository) ListBoardMembers(ctx context.Context, boardID string) ([]models.BoardMembership, error) {
	var members []models.BoardMembership
	if err := r.db.WithContext(ctx).Where("board_id = ?", boardID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("listing board members: %w", err)
	}
	return members, nil
}

// UpdateBoardMember updates a board membership role.
func (r *Repository) UpdateBoardMember(ctx context.Context, m *models.BoardMembership) error {
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating board member: %w", err)
	}
	return nil
}

// RemoveBoardMember soft-deletes a board membership.
func (r *Repository) RemoveBoardMember(ctx context.Context, boardID, userID string) error {
	result := r.db.WithContext(ctx).Where("board_id = ? AND user_id = ?", boardID, userID).Delete(&models.BoardMembership{})
	if result.Error != nil {
		return fmt.Errorf("removing board member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
