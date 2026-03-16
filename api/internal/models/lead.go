package models

// LeadStatus represents the current state of a lead.
type LeadStatus string

const (
	LeadStatusAnonymous  LeadStatus = "anonymous"
	LeadStatusRegistered LeadStatus = "registered"
	LeadStatusConverted  LeadStatus = "converted"
)

// ValidLeadStatuses returns all valid lead status values.
func ValidLeadStatuses() []LeadStatus {
	return []LeadStatus{
		LeadStatusAnonymous,
		LeadStatusRegistered,
		LeadStatusConverted,
	}
}

// IsValid checks if the lead status is a recognized value.
func (s LeadStatus) IsValid() bool {
	for _, v := range ValidLeadStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// Lead represents a sales lead captured from the chatbot or other sources.
type Lead struct {
	BaseModel
	Email         string     `gorm:"type:text;index" json:"email,omitempty"`
	Name          string     `gorm:"type:text" json:"name,omitempty"`
	Source        string     `gorm:"type:text;not null;default:'chatbot'" json:"source"`
	Status        LeadStatus `gorm:"type:text;not null;default:'anonymous'" json:"status"`
	UserID        *string    `gorm:"type:text;index" json:"user_id,omitempty"`
	AnonSessionID *string    `gorm:"type:text;index:idx_leads_anon_session" json:"anon_session_id,omitempty"`
	Metadata      string     `gorm:"type:text;default:'{}'" json:"metadata"`
}
