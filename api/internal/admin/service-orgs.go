package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Org Admin Management ---

// OrgListParams holds filter/pagination for admin org listing.
type OrgListParams struct {
	pagination.Params
	Slug           string
	Name           string
	BillingTier    string
	PaymentStatus  string
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	MemberCountMin *int
	MemberCountMax *int
}

// OrgDetail contains an org with aggregate counts for admin view.
type OrgDetail struct {
	models.Org
	MemberCount int64 `json:"member_count"`
	SpaceCount  int64 `json:"space_count"`
	BoardCount  int64 `json:"board_count"`
	ThreadCount int64 `json:"thread_count"`
}

// ListOrgs returns a filtered, paginated list of orgs for admin.
func (s *Service) ListOrgs(ctx context.Context, params OrgListParams) ([]models.Org, *pagination.PageInfo, error) {
	var orgs []models.Org
	query := s.db.WithContext(ctx).Order("id ASC")

	if params.Slug != "" {
		query = query.Where("slug LIKE ?", "%"+params.Slug+"%")
	}
	if params.Name != "" {
		query = query.Where("name LIKE ?", "%"+params.Name+"%")
	}
	if params.BillingTier != "" {
		query = query.Where("billing_tier = ?", params.BillingTier)
	}
	if params.PaymentStatus != "" {
		query = query.Where("payment_status = ?", params.PaymentStatus)
	}
	if params.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *params.CreatedAfter)
	}
	if params.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *params.CreatedBefore)
	}

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&orgs).Error; err != nil {
		return nil, nil, fmt.Errorf("listing orgs: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(orgs) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(orgs[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		orgs = orgs[:params.Limit]
	}

	return orgs, pageInfo, nil
}

// GetOrgDetail returns an org with aggregate counts.
func (s *Service) GetOrgDetail(ctx context.Context, orgIDOrSlug string) (*OrgDetail, error) {
	var org models.Org
	query := s.db.WithContext(ctx)
	if _, err := uuid.Parse(orgIDOrSlug); err == nil {
		query = query.Where("id = ?", orgIDOrSlug)
	} else {
		query = query.Where("slug = ?", orgIDOrSlug)
	}
	if err := query.First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting org: %w", err)
	}

	detail := &OrgDetail{Org: org}
	s.db.WithContext(ctx).Model(&models.OrgMembership{}).Where("org_id = ?", org.ID).Count(&detail.MemberCount)
	s.db.WithContext(ctx).Model(&models.Space{}).Where("org_id = ?", org.ID).Count(&detail.SpaceCount)

	// Count boards across all spaces in this org.
	s.db.WithContext(ctx).Model(&models.Board{}).
		Where("space_id IN (?)", s.db.Model(&models.Space{}).Select("id").Where("org_id = ?", org.ID)).
		Count(&detail.BoardCount)

	// Count threads across all boards in this org.
	s.db.WithContext(ctx).Model(&models.Thread{}).
		Where("board_id IN (?)",
			s.db.Model(&models.Board{}).Select("id").
				Where("space_id IN (?)", s.db.Model(&models.Space{}).Select("id").Where("org_id = ?", org.ID))).
		Count(&detail.ThreadCount)

	return detail, nil
}

// SuspendOrg sets suspended_at on an org.
func (s *Service) SuspendOrg(ctx context.Context, orgIDOrSlug, reason, suspendedBy string) error {
	now := time.Now()
	query := s.db.WithContext(ctx).Model(&models.Org{})
	if _, err := uuid.Parse(orgIDOrSlug); err == nil {
		query = query.Where("id = ?", orgIDOrSlug)
	} else {
		query = query.Where("slug = ?", orgIDOrSlug)
	}
	result := query.Updates(map[string]any{
		"suspended_at":   now,
		"suspend_reason": reason,
		"suspended_by":   suspendedBy,
	})
	if result.Error != nil {
		return fmt.Errorf("suspending org: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("org not found")
	}
	return nil
}

// UnsuspendOrg clears suspended_at on an org.
func (s *Service) UnsuspendOrg(ctx context.Context, orgIDOrSlug string) error {
	query := s.db.WithContext(ctx).Model(&models.Org{})
	if _, err := uuid.Parse(orgIDOrSlug); err == nil {
		query = query.Where("id = ?", orgIDOrSlug)
	} else {
		query = query.Where("slug = ?", orgIDOrSlug)
	}
	result := query.Updates(map[string]any{
		"suspended_at":   nil,
		"suspend_reason": "",
		"suspended_by":   "",
	})
	if result.Error != nil {
		return fmt.Errorf("unsuspending org: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("org not found")
	}
	return nil
}

// IsOrgSuspended checks if an org is currently suspended.
func (s *Service) IsOrgSuspended(ctx context.Context, orgIDOrSlug string) (bool, error) {
	var org models.Org
	query := s.db.WithContext(ctx)
	if _, err := uuid.Parse(orgIDOrSlug); err == nil {
		query = query.Where("id = ?", orgIDOrSlug)
	} else {
		query = query.Where("slug = ?", orgIDOrSlug)
	}
	if err := query.First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("checking org suspension: %w", err)
	}
	return org.SuspendedAt != nil, nil
}

// TransferOrgOwnership transfers ownership of an org to a new user.
func (s *Service) TransferOrgOwnership(ctx context.Context, orgIDOrSlug, newOwnerUserID string) error {
	// Resolve org ID.
	var org models.Org
	query := s.db.WithContext(ctx)
	if _, err := uuid.Parse(orgIDOrSlug); err == nil {
		query = query.Where("id = ?", orgIDOrSlug)
	} else {
		query = query.Where("slug = ?", orgIDOrSlug)
	}
	if err := query.First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("org not found")
		}
		return fmt.Errorf("finding org: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Demote current owners to admin.
		if err := tx.Model(&models.OrgMembership{}).
			Where("org_id = ? AND role = ?", org.ID, models.RoleOwner).
			Update("role", models.RoleAdmin).Error; err != nil {
			return fmt.Errorf("demoting current owners: %w", err)
		}

		// Upsert new owner membership.
		var existing models.OrgMembership
		err := tx.Where("org_id = ? AND user_id = ?", org.ID, newOwnerUserID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			m := models.OrgMembership{
				OrgID:  org.ID,
				UserID: newOwnerUserID,
				Role:   models.RoleOwner,
			}
			return tx.Create(&m).Error
		}
		if err != nil {
			return fmt.Errorf("checking membership: %w", err)
		}
		return tx.Model(&existing).Update("role", models.RoleOwner).Error
	})
}
