package models

// ThreadType represents the kind of content a thread contains.
type ThreadType string

const (
	ThreadTypeWiki          ThreadType = "wiki"
	ThreadTypeDocumentation ThreadType = "documentation"
	ThreadTypeTutorial      ThreadType = "tutorial"
	ThreadTypeForum         ThreadType = "forum"
	ThreadTypeSupport       ThreadType = "support"
	ThreadTypeLead          ThreadType = "lead"
)

// ValidThreadTypes returns all valid thread type values.
func ValidThreadTypes() []ThreadType {
	return []ThreadType{
		ThreadTypeWiki,
		ThreadTypeDocumentation,
		ThreadTypeTutorial,
		ThreadTypeForum,
		ThreadTypeSupport,
		ThreadTypeLead,
	}
}

// IsValid checks if the thread type is a recognized value.
func (t ThreadType) IsValid() bool {
	for _, v := range ValidThreadTypes() {
		if t == v {
			return true
		}
	}
	return false
}

// ThreadVisibility controls who can see a thread.
type ThreadVisibility string

const (
	ThreadVisibilityPublic   ThreadVisibility = "public"
	ThreadVisibilityOrgOnly  ThreadVisibility = "org-only"
	ThreadVisibilityDeftOnly ThreadVisibility = "deft-only"
)

// ValidThreadVisibilities returns all valid visibility values.
func ValidThreadVisibilities() []ThreadVisibility {
	return []ThreadVisibility{
		ThreadVisibilityPublic,
		ThreadVisibilityOrgOnly,
		ThreadVisibilityDeftOnly,
	}
}

// IsValid checks if the thread visibility is a recognized value.
func (v ThreadVisibility) IsValid() bool {
	for _, val := range ValidThreadVisibilities() {
		if v == val {
			return true
		}
	}
	return false
}

// Thread represents a discussion thread within a Board.
type Thread struct {
	BaseModel
	BoardID    string           `gorm:"type:text;not null;index" json:"board_id"`
	Title      string           `gorm:"type:text;not null" json:"title"`
	Body       string           `gorm:"type:text" json:"body,omitempty"`
	Slug       string           `gorm:"type:text;not null" json:"slug"`
	Metadata   string           `gorm:"type:text;default:'{}'" json:"metadata"`
	AuthorID   string           `gorm:"type:text;not null;index" json:"author_id"`
	IsPinned   bool             `gorm:"default:false" json:"is_pinned"`
	IsLocked   bool             `gorm:"default:false" json:"is_locked"`
	IsHidden   bool             `gorm:"default:false" json:"is_hidden"`
	VoteScore  int              `gorm:"default:0" json:"vote_score"`
	ThreadType ThreadType       `gorm:"type:text;not null;default:'forum';index" json:"thread_type"`
	Visibility ThreadVisibility `gorm:"type:text;not null;default:'org-only';index" json:"visibility"`
	OrgID      *string          `gorm:"type:text;index" json:"org_id,omitempty"`

	// Generated columns extracted from Metadata JSON for indexing/querying.
	Status     string `gorm:"type:text;->;-:migration" json:"status,omitempty"`
	Priority   string `gorm:"type:text;->;-:migration" json:"priority,omitempty"`
	Stage      string `gorm:"type:text;->;-:migration" json:"stage,omitempty"`
	AssignedTo string `gorm:"type:text;->;-:migration" json:"assigned_to,omitempty"`

	// ContactEmail stores the email of the person this ticket is for, when created
	// on behalf of an unregistered user. It is cleared once the user registers
	// and claims the ticket.
	ContactEmail string `gorm:"type:text;index" json:"contact_email,omitempty"`

	// TicketNumber is a sequential, human-readable identifier assigned when a
	// support thread is created. Zero means not yet assigned (non-support threads).
	TicketNumber int64 `gorm:"default:0;index" json:"ticket_number,omitempty"`

	// Associations.
	Board    Board     `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"-"`
	Messages []Message `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"messages,omitempty"`
}
