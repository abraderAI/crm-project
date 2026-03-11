package models

// Role represents a user's role within an entity (org, space, board).
type Role string

const (
	RoleViewer      Role = "viewer"
	RoleCommenter   Role = "commenter"
	RoleContributor Role = "contributor"
	RoleModerator   Role = "moderator"
	RoleAdmin       Role = "admin"
	RoleOwner       Role = "owner"
)

// RoleHierarchy returns the role hierarchy ordered from lowest to highest.
func RoleHierarchy() []Role {
	return []Role{
		RoleViewer,
		RoleCommenter,
		RoleContributor,
		RoleModerator,
		RoleAdmin,
		RoleOwner,
	}
}

// IsValid checks if the role is a recognized value.
func (r Role) IsValid() bool {
	for _, v := range RoleHierarchy() {
		if r == v {
			return true
		}
	}
	return false
}

// Level returns the numeric level for role comparison. Higher = more permissions.
func (r Role) Level() int {
	for i, v := range RoleHierarchy() {
		if r == v {
			return i
		}
	}
	return -1
}

// OrgMembership links a user to an organization with a specific role.
type OrgMembership struct {
	BaseModel
	OrgID  string `gorm:"type:text;not null;uniqueIndex:idx_org_member" json:"org_id"`
	UserID string `gorm:"type:text;not null;uniqueIndex:idx_org_member" json:"user_id"`
	Role   Role   `gorm:"type:text;not null;default:'viewer'" json:"role"`

	// Associations.
	Org Org `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
}

// SpaceMembership links a user to a space with a specific role.
type SpaceMembership struct {
	BaseModel
	SpaceID string `gorm:"type:text;not null;uniqueIndex:idx_space_member" json:"space_id"`
	UserID  string `gorm:"type:text;not null;uniqueIndex:idx_space_member" json:"user_id"`
	Role    Role   `gorm:"type:text;not null;default:'viewer'" json:"role"`

	// Associations.
	Space Space `gorm:"foreignKey:SpaceID;constraint:OnDelete:CASCADE" json:"-"`
}

// BoardMembership links a user to a board with a specific role.
type BoardMembership struct {
	BaseModel
	BoardID string `gorm:"type:text;not null;uniqueIndex:idx_board_member" json:"board_id"`
	UserID  string `gorm:"type:text;not null;uniqueIndex:idx_board_member" json:"user_id"`
	Role    Role   `gorm:"type:text;not null;default:'viewer'" json:"role"`

	// Associations.
	Board Board `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"-"`
}
