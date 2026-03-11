package models

// FlagStatus represents the current state of a moderation flag.
type FlagStatus string

const (
	FlagStatusOpen      FlagStatus = "open"
	FlagStatusResolved  FlagStatus = "resolved"
	FlagStatusDismissed FlagStatus = "dismissed"
)

// IsValid checks if the flag status is a recognized value.
func (f FlagStatus) IsValid() bool {
	switch f {
	case FlagStatusOpen, FlagStatusResolved, FlagStatusDismissed:
		return true
	}
	return false
}

// Flag represents a user-submitted moderation flag on a thread.
type Flag struct {
	BaseModel
	ThreadID   string     `gorm:"type:text;not null;index" json:"thread_id"`
	UserID     string     `gorm:"type:text;not null;index" json:"user_id"`
	Reason     string     `gorm:"type:text;not null" json:"reason"`
	Status     FlagStatus `gorm:"type:text;not null;default:'open'" json:"status"`
	ResolvedBy string     `gorm:"type:text" json:"resolved_by,omitempty"`

	// Associations.
	Thread Thread `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"-"`
}
