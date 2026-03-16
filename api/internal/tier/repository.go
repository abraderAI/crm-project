package tier

import (
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/seed"
)

// Repository provides database queries for tier resolution.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new tier Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// IsPlatformAdmin checks if the user has an active platform admin record.
func (r *Repository) IsPlatformAdmin(userID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.PlatformAdmin{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Count(&count).Error
	return count > 0, err
}

// deftMembershipResult holds the result of a DEFT org membership query.
type deftMembershipResult struct {
	OrgID     string
	Role      models.Role
	SpaceSlug string
}

// GetDeftOrgMembership checks if the user is a member of the DEFT org and returns
// their role and department space slug (if any).
func (r *Repository) GetDeftOrgMembership(userID string) (*deftMembershipResult, error) {
	// First check org membership.
	var orgMember models.OrgMembership
	err := r.db.
		Joins("JOIN orgs ON orgs.id = org_memberships.org_id AND orgs.deleted_at IS NULL").
		Where("org_memberships.user_id = ? AND orgs.slug = ?", userID, seed.DeftOrgSlug).
		First(&orgMember).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := &deftMembershipResult{
		OrgID: orgMember.OrgID,
		Role:  orgMember.Role,
	}

	// Check department space memberships to determine sub-type.
	var spaceMember models.SpaceMembership
	err = r.db.
		Joins("JOIN spaces ON spaces.id = space_memberships.space_id AND spaces.deleted_at IS NULL").
		Where("space_memberships.user_id = ? AND spaces.org_id = ?", userID, orgMember.OrgID).
		First(&spaceMember).Error
	if err == nil {
		// Look up the space slug.
		var space models.Space
		if err := r.db.Select("slug").First(&space, "id = ?", spaceMember.SpaceID).Error; err == nil {
			result.SpaceSlug = space.Slug
		}
	}

	return result, nil
}

// customerOrgResult holds the result of a customer org membership query.
type customerOrgResult struct {
	OrgID string
	Role  models.Role
}

// GetCustomerOrgMembership checks if the user is a member of any paying customer org.
// Returns the first matching org membership (admin/owner roles sorted first).
func (r *Repository) GetCustomerOrgMembership(userID string) (*customerOrgResult, error) {
	var orgMember models.OrgMembership
	err := r.db.
		Joins("JOIN orgs ON orgs.id = org_memberships.org_id AND orgs.deleted_at IS NULL").
		Where("org_memberships.user_id = ? AND orgs.slug != ? AND orgs.slug != ?",
			userID, seed.SystemOrgSlug, seed.DeftOrgSlug).
		Order("CASE WHEN org_memberships.role = 'owner' THEN 0 WHEN org_memberships.role = 'admin' THEN 1 ELSE 2 END").
		First(&orgMember).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &customerOrgResult{
		OrgID: orgMember.OrgID,
		Role:  orgMember.Role,
	}, nil
}

// UserExists checks if a user shadow record exists for the given user ID.
func (r *Repository) UserExists(userID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserShadow{}).
		Where("clerk_user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}

// HomePreferences retrieves the user's saved home layout preferences.
// Returns nil if no preferences are saved.
func (r *Repository) HomePreferences(userID string) (*models.UserHomePreferences, error) {
	var prefs models.UserHomePreferences
	err := r.db.Where("user_id = ?", userID).First(&prefs).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

// SaveHomePreferences upserts the user's home layout preferences.
func (r *Repository) SaveHomePreferences(prefs *models.UserHomePreferences) error {
	return r.db.Save(prefs).Error
}
