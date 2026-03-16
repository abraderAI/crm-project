package tier

import (
	"fmt"
	"strings"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Service provides tier resolution and home preference management.
type Service struct {
	repo *Repository
}

// NewService creates a new tier Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ResolveTier determines a user's tier based on the priority chain:
// 1. Platform admin flag → Tier 6
// 2. DEFT org member → Tier 4
// 3. Customer org admin/owner → Tier 5
// 4. Customer org member → Tier 3
// 5. Registered user (has UserShadow) → Tier 2
// 6. Anonymous → Tier 1
func (s *Service) ResolveTier(userID string) (*TierResult, error) {
	// Empty/anonymous user → Tier 1.
	if userID == "" {
		return &TierResult{Tier: TierAnonymous}, nil
	}

	// Step 1: Check platform admin.
	isAdmin, err := s.repo.IsPlatformAdmin(userID)
	if err != nil {
		return nil, fmt.Errorf("checking platform admin: %w", err)
	}
	if isAdmin {
		return &TierResult{Tier: TierPlatformAdmin}, nil
	}

	// Step 2: Check DEFT org membership.
	deftMember, err := s.repo.GetDeftOrgMembership(userID)
	if err != nil {
		return nil, fmt.Errorf("checking deft membership: %w", err)
	}
	if deftMember != nil {
		result := &TierResult{
			Tier:  TierDeftEmployee,
			OrgID: deftMember.OrgID,
		}
		result.SubType, result.DeftDepartment = resolveDeftDepartment(deftMember.SpaceSlug)
		return result, nil
	}

	// Step 3 & 4: Check customer org membership.
	custMember, err := s.repo.GetCustomerOrgMembership(userID)
	if err != nil {
		return nil, fmt.Errorf("checking customer membership: %w", err)
	}
	if custMember != nil {
		if custMember.Role == models.RoleAdmin || custMember.Role == models.RoleOwner {
			subType := SubTypeNone
			if custMember.Role == models.RoleOwner {
				subType = SubTypeOrgOwner
			}
			return &TierResult{
				Tier:    TierCustomerAdmin,
				OrgID:   custMember.OrgID,
				SubType: subType,
			}, nil
		}
		return &TierResult{
			Tier:  TierCustomer,
			OrgID: custMember.OrgID,
		}, nil
	}

	// Step 5: Check if registered user.
	exists, err := s.repo.UserExists(userID)
	if err != nil {
		return nil, fmt.Errorf("checking user existence: %w", err)
	}
	if exists {
		return &TierResult{Tier: TierRegistered}, nil
	}

	// Step 6: Anonymous.
	return &TierResult{Tier: TierAnonymous}, nil
}

// resolveDeftDepartment maps a space slug to a department sub-type.
func resolveDeftDepartment(spaceSlug string) (SubType, string) {
	switch {
	case strings.HasSuffix(spaceSlug, "-sales"):
		return SubTypeDeftSales, "sales"
	case strings.HasSuffix(spaceSlug, "-support"):
		return SubTypeDeftSupport, "support"
	case strings.HasSuffix(spaceSlug, "-finance"):
		return SubTypeDeftFinance, "finance"
	default:
		return SubTypeNone, ""
	}
}

// GetHomePreferences retrieves a user's saved home layout preferences.
func (s *Service) GetHomePreferences(userID string) (*models.UserHomePreferences, error) {
	return s.repo.HomePreferences(userID)
}

// SaveHomePreferences validates and stores home layout preferences.
func (s *Service) SaveHomePreferences(prefs *models.UserHomePreferences) error {
	if prefs.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if prefs.Layout == "" {
		return fmt.Errorf("layout is required")
	}
	if !Tier(prefs.Tier).IsValid() {
		return fmt.Errorf("invalid tier: %d", prefs.Tier)
	}
	return s.repo.SaveHomePreferences(prefs)
}
