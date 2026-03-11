package models

// Thread represents a discussion thread within a Board.
type Thread struct {
	BaseModel
	BoardID   string `gorm:"type:text;not null;index" json:"board_id"`
	Title     string `gorm:"type:text;not null" json:"title"`
	Body      string `gorm:"type:text" json:"body,omitempty"`
	Slug      string `gorm:"type:text;not null" json:"slug"`
	Metadata  string `gorm:"type:text;default:'{}'" json:"metadata"`
	AuthorID  string `gorm:"type:text;not null;index" json:"author_id"`
	IsPinned  bool   `gorm:"default:false" json:"is_pinned"`
	IsLocked  bool   `gorm:"default:false" json:"is_locked"`
	VoteScore int    `gorm:"default:0" json:"vote_score"`

	// Generated columns extracted from Metadata JSON for indexing/querying.
	Status     string `gorm:"type:text;->;-:migration" json:"status,omitempty"`
	Priority   string `gorm:"type:text;->;-:migration" json:"priority,omitempty"`
	Stage      string `gorm:"type:text;->;-:migration" json:"stage,omitempty"`
	AssignedTo string `gorm:"type:text;->;-:migration" json:"assigned_to,omitempty"`

	// Associations.
	Board    Board     `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"-"`
	Messages []Message `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"messages,omitempty"`
}
