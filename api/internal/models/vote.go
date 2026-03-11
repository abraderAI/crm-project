package models

// Vote represents a user's vote on a thread.
// Unique constraint on (ThreadID, UserID) ensures one vote per user per thread.
type Vote struct {
	BaseModel
	ThreadID string `gorm:"type:text;not null;uniqueIndex:idx_vote_unique" json:"thread_id"`
	UserID   string `gorm:"type:text;not null;uniqueIndex:idx_vote_unique" json:"user_id"`
	Weight   int    `gorm:"default:1" json:"weight"`

	// Associations.
	Thread Thread `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"-"`
}
