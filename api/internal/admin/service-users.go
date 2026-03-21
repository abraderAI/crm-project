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
// When email or displayName is empty, the existing DB value is preserved (not overwritten).
// If both are empty and a Clerk client is configured, it attempts a one-time fetch from
// the Clerk Backend API so that users are never displayed by their raw Clerk ID.
func (s *Service) SyncUserShadow(ctx context.Context, userID, email, displayName string) {
	// If the JWT didn't include identity claims, try the Clerk Backend API once —
	// but only when the user shadow doesn't already have an email stored.
	if email == "" && displayName == "" && s.clerkClient != nil {
		var existing models.UserShadow
		findErr := s.db.WithContext(ctx).
			Select("email", "display_name").
			Where("clerk_user_id = ?", userID).
			First(&existing).Error
		// Call Clerk when the record is absent (findErr != nil) or has no email yet.
		if findErr != nil || existing.Email == "" {
			if user, fetchErr := s.clerkClient.GetUser(ctx, userID); fetchErr == nil {
				email = user.Email
				displayName = user.DisplayName
			}
		}
	}

	now := time.Now()
	shadow := models.UserShadow{
		ClerkUserID: userID,
		Email:       email,
		DisplayName: displayName,
		LastSeenAt:  now,
		SyncedAt:    now,
	}
	// Always update last_seen_at and synced_at; only overwrite identity fields when non-empty.
	assignFields := map[string]any{
		"last_seen_at": now,
		"synced_at":    now,
	}
	if email != "" {
		assignFields["email"] = email
	}
	if displayName != "" {
		assignFields["display_name"] = displayName
	}
	s.db.WithContext(ctx).Where("clerk_user_id = ?", userID).
		Assign(assignFields).FirstOrCreate(&shadow)
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

// OrgMembershipEnriched is an org membership with resolved org name and slug.
type OrgMembershipEnriched struct {
	models.OrgMembership
	OrgName string `json:"org_name"`
	OrgSlug string `json:"org_slug"`
}

// UserDetail contains a user shadow with their enriched org memberships.
type UserDetail struct {
	models.UserShadow
	Memberships []OrgMembershipEnriched `json:"memberships,omitempty"`
}

// UserShadowWithOrg extends UserShadow with primary org info for list views.
type UserShadowWithOrg struct {
	models.UserShadow
	PrimaryOrgName string `json:"primary_org_name,omitempty"`
	PrimaryOrgSlug string `json:"primary_org_slug,omitempty"`
}

// ListUsers returns a filtered, paginated list of user shadows with primary org info.
func (s *Service) ListUsers(ctx context.Context, params UserListParams) ([]UserShadowWithOrg, *pagination.PageInfo, error) {
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

	// Resolve primary org for each user in a single batch query.
	result := make([]UserShadowWithOrg, len(users))
	if len(users) > 0 {
		userIDs := make([]string, len(users))
		for i, u := range users {
			userIDs[i] = u.ClerkUserID
			result[i] = UserShadowWithOrg{UserShadow: u}
		}

		// Fetch all active memberships with org info, then pick one per user in Go.
		type orgRow struct {
			UserID    string
			OrgName   string
			OrgSlug   string
			CreatedAt time.Time
		}
		var orgRows []orgRow
		_ = s.db.WithContext(ctx).Raw(`
			SELECT om.user_id, o.name AS org_name, o.slug AS org_slug, om.created_at
			FROM org_memberships om
			JOIN orgs o ON o.id = om.org_id AND o.deleted_at IS NULL
			WHERE om.user_id IN (?) AND om.deleted_at IS NULL
			ORDER BY om.created_at ASC`, userIDs).Scan(&orgRows).Error

		// Pick first (oldest) membership per user.
		orgMap := make(map[string]orgRow, len(userIDs))
		for _, row := range orgRows {
			if _, exists := orgMap[row.UserID]; !exists {
				orgMap[row.UserID] = row
			}
		}
		for i := range result {
			if row, ok := orgMap[result[i].ClerkUserID]; ok {
				result[i].PrimaryOrgName = row.OrgName
				result[i].PrimaryOrgSlug = row.OrgSlug
			}
		}
	}

	return result, pageInfo, nil
}

// GetUser returns a user with all their enriched org memberships.
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

	// Resolve org names in a single query.
	enriched := make([]OrgMembershipEnriched, len(memberships))
	if len(memberships) > 0 {
		orgIDs := make([]string, len(memberships))
		for i, m := range memberships {
			orgIDs[i] = m.OrgID
			enriched[i] = OrgMembershipEnriched{OrgMembership: m}
		}

		var orgs []models.Org
		_ = s.db.WithContext(ctx).Select("id", "name", "slug").Where("id IN ?", orgIDs).Find(&orgs).Error

		orgMap := make(map[string]models.Org, len(orgs))
		for _, o := range orgs {
			orgMap[o.ID] = o
		}
		for i := range enriched {
			if o, ok := orgMap[enriched[i].OrgID]; ok {
				enriched[i].OrgName = o.Name
				enriched[i].OrgSlug = o.Slug
			}
		}
	}

	return &UserDetail{
		UserShadow:  shadow,
		Memberships: enriched,
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
