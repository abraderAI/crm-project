package models

// Board represents a topic board within a Space.
type Board struct {
	BaseModel
	SpaceID     string `gorm:"type:text;not null;index" json:"space_id"`
	Name        string `gorm:"type:text;not null" json:"name"`
	Slug        string `gorm:"type:text;not null" json:"slug"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Metadata    string `gorm:"type:text;default:'{}'" json:"metadata"`
	IsLocked    bool   `gorm:"default:false" json:"is_locked"`

	// Associations.
	Space   Space    `gorm:"foreignKey:SpaceID;constraint:OnDelete:CASCADE" json:"-"`
	Threads []Thread `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"threads,omitempty"`
}
