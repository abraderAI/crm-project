package models

// Org represents a top-level organization in the platform hierarchy.
type Org struct {
	BaseModel
	Name        string `gorm:"type:text;not null" json:"name"`
	Slug        string `gorm:"type:text;uniqueIndex;not null" json:"slug"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Metadata    string `gorm:"type:text;default:'{}'" json:"metadata"`

	// Generated columns extracted from Metadata JSON for indexing/querying.
	BillingTier   string `gorm:"type:text;->;-:migration" json:"billing_tier,omitempty"`
	PaymentStatus string `gorm:"type:text;->;-:migration" json:"payment_status,omitempty"`

	// Associations.
	Spaces []Space `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"spaces,omitempty"`
}
