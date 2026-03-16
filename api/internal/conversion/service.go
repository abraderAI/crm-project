// Package conversion provides Tier 2→3 conversion flows: self-service upgrade,
// sales-assisted conversion, and platform admin override.
package conversion

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/seed"
)

// Service provides conversion business logic.
type Service struct {
	db *gorm.DB
}

// NewService creates a new conversion Service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// SelfServiceResult holds the result of a self-service upgrade.
type SelfServiceResult struct {
	Org    *models.Org `json:"org"`
	Status string      `json:"status"`
	Tier   int         `json:"tier"`
}

// SelfServiceUpgrade creates an org for a registered user (Tier 2) and promotes
// them to Tier 3. This is the stub billing flow — no real payment is processed.
func (s *Service) SelfServiceUpgrade(ctx context.Context, userID, orgName string) (*SelfServiceResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if orgName == "" {
		return nil, fmt.Errorf("org_name is required")
	}

	// Verify user exists.
	var shadow models.UserShadow
	if err := s.db.WithContext(ctx).Where("clerk_user_id = ?", userID).First(&shadow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	// Check user is not already in a customer org.
	var existingMember models.OrgMembership
	err := s.db.WithContext(ctx).
		Joins("JOIN orgs ON orgs.id = org_memberships.org_id AND orgs.deleted_at IS NULL").
		Where("org_memberships.user_id = ? AND orgs.slug != ? AND orgs.slug != ?",
			userID, seed.SystemOrgSlug, seed.DeftOrgSlug).
		First(&existingMember).Error
	if err == nil {
		return nil, fmt.Errorf("user already belongs to an organization")
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("checking existing membership: %w", err)
	}

	// Create the org and membership in a transaction.
	var org models.Org
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org = models.Org{
			Name:     orgName,
			Slug:     generateSlug(orgName),
			Metadata: "{}",
		}
		if err := tx.Create(&org).Error; err != nil {
			return fmt.Errorf("creating org: %w", err)
		}

		membership := models.OrgMembership{
			OrgID:  org.ID,
			UserID: userID,
			Role:   models.RoleOwner,
		}
		if err := tx.Create(&membership).Error; err != nil {
			return fmt.Errorf("creating membership: %w", err)
		}

		// Update any existing lead to converted.
		tx.Model(&models.Lead{}).
			Where("user_id = ? OR email = ?", userID, shadow.Email).
			Updates(map[string]any{
				"status":  models.LeadStatusConverted,
				"user_id": &userID,
			})

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return &SelfServiceResult{
		Org:    &org,
		Status: "converted",
		Tier:   3,
	}, nil
}

// SalesConvertResult holds the result of a sales-assisted conversion.
type SalesConvertResult struct {
	LeadID string      `json:"lead_id"`
	Org    *models.Org `json:"org"`
	Status string      `json:"status"`
}

// SalesConvert allows a DEFT sales member to convert a lead to Tier 3.
// Creates an org and promotes the lead's user (if linked) to the new org.
func (s *Service) SalesConvert(ctx context.Context, leadID, orgName, actorID string) (*SalesConvertResult, error) {
	if leadID == "" {
		return nil, fmt.Errorf("lead_id is required")
	}
	if orgName == "" {
		return nil, fmt.Errorf("org_name is required")
	}

	// Look up the lead.
	var lead models.Lead
	if err := s.db.WithContext(ctx).First(&lead, "id = ?", leadID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("looking up lead: %w", err)
	}

	if lead.Status == models.LeadStatusConverted {
		return nil, fmt.Errorf("lead is already converted")
	}

	var org models.Org
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org = models.Org{
			Name:     orgName,
			Slug:     generateSlug(orgName),
			Metadata: "{}",
		}
		if err := tx.Create(&org).Error; err != nil {
			return fmt.Errorf("creating org: %w", err)
		}

		// If the lead has a linked user, add them as owner.
		if lead.UserID != nil && *lead.UserID != "" {
			membership := models.OrgMembership{
				OrgID:  org.ID,
				UserID: *lead.UserID,
				Role:   models.RoleOwner,
			}
			if err := tx.Create(&membership).Error; err != nil {
				return fmt.Errorf("creating membership: %w", err)
			}
		}

		// Update lead status.
		if err := tx.Model(&lead).Updates(map[string]any{
			"status": models.LeadStatusConverted,
		}).Error; err != nil {
			return fmt.Errorf("updating lead status: %w", err)
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return &SalesConvertResult{
		LeadID: leadID,
		Org:    &org,
		Status: "converted",
	}, nil
}

// AdminPromoteResult holds the result of an admin-driven promotion.
type AdminPromoteResult struct {
	UserID string      `json:"user_id"`
	Org    *models.Org `json:"org"`
	Status string      `json:"status"`
	Tier   int         `json:"tier"`
}

// AdminPromote allows a platform admin to assign a user to an org (creating it
// if needed) and mark them as a paying customer (Tier 3).
func (s *Service) AdminPromote(ctx context.Context, userID, orgName string) (*AdminPromoteResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if orgName == "" {
		return nil, fmt.Errorf("org_name is required")
	}

	// Verify user exists.
	var shadow models.UserShadow
	if err := s.db.WithContext(ctx).Where("clerk_user_id = ?", userID).First(&shadow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	var org models.Org
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		org = models.Org{
			Name:     orgName,
			Slug:     generateSlug(orgName),
			Metadata: "{}",
		}
		if err := tx.Create(&org).Error; err != nil {
			return fmt.Errorf("creating org: %w", err)
		}

		membership := models.OrgMembership{
			OrgID:  org.ID,
			UserID: userID,
			Role:   models.RoleOwner,
		}
		if err := tx.Create(&membership).Error; err != nil {
			return fmt.Errorf("creating membership: %w", err)
		}

		// Update any existing lead to converted.
		tx.Model(&models.Lead{}).
			Where("user_id = ? OR email = ?", userID, shadow.Email).
			Updates(map[string]any{
				"status":  models.LeadStatusConverted,
				"user_id": &userID,
			})

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return &AdminPromoteResult{
		UserID: userID,
		Org:    &org,
		Status: "promoted",
		Tier:   3,
	}, nil
}

// IsDeftOrgMember checks whether the given user is a member of the DEFT org.
func (s *Service) IsDeftOrgMember(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.OrgMembership{}).
		Joins("JOIN orgs ON orgs.id = org_memberships.org_id AND orgs.deleted_at IS NULL").
		Where("org_memberships.user_id = ? AND orgs.slug = ?", userID, seed.DeftOrgSlug).
		Count(&count).Error
	return count > 0, err
}

// generateSlug creates a URL-safe slug from an org name.
func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z':
			slug += string(c)
		case c >= 'A' && c <= 'Z':
			slug += string(c + 32) // toLower
		case c >= '0' && c <= '9':
			slug += string(c)
		case c == ' ' || c == '-' || c == '_':
			if len(slug) > 0 && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}
	// Trim trailing hyphen.
	if len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}
	if slug == "" {
		slug = "org"
	}
	return slug
}
