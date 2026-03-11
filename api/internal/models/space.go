package models

// SpaceType represents the kind of space.
type SpaceType string

const (
	SpaceTypeGeneral       SpaceType = "general"
	SpaceTypeCRM           SpaceType = "crm"
	SpaceTypeSupport       SpaceType = "support"
	SpaceTypeCommunity     SpaceType = "community"
	SpaceTypeKnowledgeBase SpaceType = "knowledge_base"
)

// ValidSpaceTypes returns all valid space type values.
func ValidSpaceTypes() []SpaceType {
	return []SpaceType{
		SpaceTypeGeneral,
		SpaceTypeCRM,
		SpaceTypeSupport,
		SpaceTypeCommunity,
		SpaceTypeKnowledgeBase,
	}
}

// IsValid checks if the space type is a recognized value.
func (s SpaceType) IsValid() bool {
	for _, v := range ValidSpaceTypes() {
		if s == v {
			return true
		}
	}
	return false
}

// Space represents a categorized area within an Org.
type Space struct {
	BaseModel
	OrgID       string    `gorm:"type:text;not null;index" json:"org_id"`
	Name        string    `gorm:"type:text;not null" json:"name"`
	Slug        string    `gorm:"type:text;not null" json:"slug"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Metadata    string    `gorm:"type:text;default:'{}'" json:"metadata"`
	Type        SpaceType `gorm:"type:text;not null;default:'general'" json:"type"`

	// Associations.
	Org    Org     `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
	Boards []Board `gorm:"foreignKey:SpaceID;constraint:OnDelete:CASCADE" json:"boards,omitempty"`
}
