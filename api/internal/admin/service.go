// Package admin provides platform administration endpoints and middleware
// for the DEFT Evolution CRM platform.
package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Service provides platform administration operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new admin service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// --- Platform Admin Management ---

// IsPlatformAdmin checks whether a user is an active platform admin.
func (s *Service) IsPlatformAdmin(ctx context.Context, userID string) (bool, error) {
	var admin models.PlatformAdmin
	err := s.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).First(&admin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("checking platform admin: %w", err)
	}
	return true, nil
}

// ListPlatformAdmins returns all platform admins.
func (s *Service) ListPlatformAdmins(ctx context.Context) ([]models.PlatformAdmin, error) {
	var admins []models.PlatformAdmin
	if err := s.db.WithContext(ctx).Find(&admins).Error; err != nil {
		return nil, fmt.Errorf("listing platform admins: %w", err)
	}
	return admins, nil
}

// AddPlatformAdmin grants platform admin to a user.
func (s *Service) AddPlatformAdmin(ctx context.Context, userID, grantedBy string) (*models.PlatformAdmin, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	admin := &models.PlatformAdmin{
		UserID:    userID,
		GrantedBy: grantedBy,
		IsActive:  true,
	}
	if err := s.db.WithContext(ctx).Create(admin).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			return nil, fmt.Errorf("user is already a platform admin")
		}
		return nil, fmt.Errorf("adding platform admin: %w", err)
	}
	return admin, nil
}

// RemovePlatformAdmin deactivates a platform admin. Returns error if it's the last active admin.
func (s *Service) RemovePlatformAdmin(ctx context.Context, userID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.PlatformAdmin{}).
		Where("is_active = ?", true).Count(&count).Error; err != nil {
		return fmt.Errorf("counting platform admins: %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot remove the last platform admin")
	}

	result := s.db.WithContext(ctx).Delete(&models.PlatformAdmin{}, "user_id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("removing platform admin: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("platform admin not found")
	}
	return nil
}

// BootstrapAdmin seeds the first platform admin from an env var value.
// Idempotent — does nothing if the admin already exists.
func (s *Service) BootstrapAdmin(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	var existing models.PlatformAdmin
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		return nil // Already exists.
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("checking bootstrap admin: %w", err)
	}
	admin := &models.PlatformAdmin{
		UserID:    userID,
		GrantedBy: "bootstrap",
		IsActive:  true,
	}
	return s.db.WithContext(ctx).Create(admin).Error
}

// --- User Shadow Management ---

// SyncUserShadow upserts a user shadow record from JWT claims.
func (s *Service) SyncUserShadow(ctx context.Context, userID, email, displayName string) {
	now := time.Now()
	shadow := models.UserShadow{
		ClerkUserID: userID,
		Email:       email,
		DisplayName: displayName,
		LastSeenAt:  now,
		SyncedAt:    now,
	}
	// Upsert: create or update last_seen_at and synced_at.
	s.db.WithContext(ctx).Where("clerk_user_id = ?", userID).
		Assign(map[string]any{
			"last_seen_at": now,
			"synced_at":    now,
			"email":        email,
			"display_name": displayName,
		}).FirstOrCreate(&shadow)
}

// IsUserBanned checks if a user is banned.
func (s *Service) IsUserBanned(ctx context.Context, userID string) (bool, error) {
	var shadow models.UserShadow
	err := s.db.WithContext(ctx).Where("clerk_user_id = ?", userID).First(&shadow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("checking ban status: %w", err)
	}
	return shadow.IsBanned, nil
}

// UserListParams holds filter/pagination for admin user listing.
type UserListParams struct {
	pagination.Params
	Email      string
	Name       string
	UserID     string
	OrgSlug    string
	IsBanned   *bool
	SeenAfter  *time.Time
	SeenBefore *time.Time
}

// UserDetail contains a user shadow with their org memberships.
type UserDetail struct {
	models.UserShadow
	Memberships []models.OrgMembership `json:"memberships,omitempty"`
}

// ListUsers returns a filtered, paginated list of user shadows.
func (s *Service) ListUsers(ctx context.Context, params UserListParams) ([]models.UserShadow, *pagination.PageInfo, error) {
	var users []models.UserShadow
	query := s.db.WithContext(ctx).Order("clerk_user_id ASC")

	if params.Email != "" {
		query = query.Where("email LIKE ?", "%"+params.Email+"%")
	}
	if params.Name != "" {
		query = query.Where("display_name LIKE ?", "%"+params.Name+"%")
	}
	if params.UserID != "" {
		query = query.Where("clerk_user_id LIKE ?", params.UserID+"%")
	}
	if params.IsBanned != nil {
		query = query.Where("is_banned = ?", *params.IsBanned)
	}
	if params.SeenAfter != nil {
		query = query.Where("last_seen_at >= ?", *params.SeenAfter)
	}
	if params.SeenBefore != nil {
		query = query.Where("last_seen_at <= ?", *params.SeenBefore)
	}
	if params.OrgSlug != "" {
		query = query.Where("clerk_user_id IN (?)",
			s.db.Model(&models.OrgMembership{}).Select("user_id").
				Joins("JOIN orgs ON orgs.id = org_memberships.org_id").
				Where("orgs.slug = ?", params.OrgSlug))
	}

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("clerk_user_id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&users).Error; err != nil {
		return nil, nil, fmt.Errorf("listing users: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(users) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(users[params.Limit-1].ClerkUserID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		users = users[:params.Limit]
	}

	return users, pageInfo, nil
}

// GetUser returns a user with all their org memberships.
func (s *Service) GetUser(ctx context.Context, userID string) (*UserDetail, error) {
	var shadow models.UserShadow
	if err := s.db.WithContext(ctx).Where("clerk_user_id = ?", userID).First(&shadow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}

	var memberships []models.OrgMembership
	_ = s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&memberships).Error

	return &UserDetail{
		UserShadow:  shadow,
		Memberships: memberships,
	}, nil
}

// BanUser sets is_banned=true for a user.
func (s *Service) BanUser(ctx context.Context, userID, reason, bannedBy string) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.UserShadow{}).
		Where("clerk_user_id = ?", userID).
		Updates(map[string]any{
			"is_banned":  true,
			"ban_reason": reason,
			"banned_at":  &now,
			"banned_by":  bannedBy,
		})
	if result.Error != nil {
		return fmt.Errorf("banning user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		// User shadow might not exist yet; create it as banned.
		shadow := models.UserShadow{
			ClerkUserID: userID,
			IsBanned:    true,
			BanReason:   reason,
			BannedAt:    &now,
			BannedBy:    bannedBy,
			SyncedAt:    now,
			LastSeenAt:  now,
		}
		return s.db.WithContext(ctx).Create(&shadow).Error
	}
	return nil
}

// UnbanUser sets is_banned=false for a user.
func (s *Service) UnbanUser(ctx context.Context, userID string) error {
	result := s.db.WithContext(ctx).Model(&models.UserShadow{}).
		Where("clerk_user_id = ?", userID).
		Updates(map[string]any{
			"is_banned":  false,
			"ban_reason": "",
			"banned_at":  nil,
			"banned_by":  "",
		})
	if result.Error != nil {
		return fmt.Errorf("unbanning user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

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
		"suspended_at":   &now,
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

// --- Platform-wide Audit Log ---

// AuditListParams extends pagination with platform-wide audit filters.
type AuditListParams struct {
	pagination.Params
	OrgID      string
	UserID     string
	Action     string
	EntityType string
	IPAddress  string
	After      *time.Time
	Before     *time.Time
}

// ListAuditLogs returns a platform-wide filtered, paginated audit log.
func (s *Service) ListAuditLogs(ctx context.Context, params AuditListParams) ([]models.AuditLog, *pagination.PageInfo, error) {
	var logs []models.AuditLog
	query := s.db.WithContext(ctx).Order("id DESC")

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}
	if params.EntityType != "" {
		query = query.Where("entity_type = ?", params.EntityType)
	}
	if params.IPAddress != "" {
		query = query.Where("ip_address = ?", params.IPAddress)
	}
	if params.After != nil {
		query = query.Where("created_at >= ?", *params.After)
	}
	if params.Before != nil {
		query = query.Where("created_at <= ?", *params.Before)
	}

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&logs).Error; err != nil {
		return nil, nil, fmt.Errorf("listing audit logs: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(logs) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(logs[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		logs = logs[:params.Limit]
	}

	return logs, pageInfo, nil
}
