package models

// Upload represents a file uploaded and attached to an entity.
type Upload struct {
	BaseModel
	OrgID       string `gorm:"type:text;not null;index" json:"org_id"`
	EntityType  string `gorm:"type:text;not null" json:"entity_type"`
	EntityID    string `gorm:"type:text;not null" json:"entity_id"`
	Filename    string `gorm:"type:text;not null" json:"filename"`
	ContentType string `gorm:"type:text;not null" json:"content_type"`
	Size        int64  `gorm:"not null" json:"size"`
	StoragePath string `gorm:"type:text;not null" json:"storage_path"`
	UploaderID  string `gorm:"type:text;not null;index" json:"uploader_id"`

	// Associations.
	Org Org `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
}
