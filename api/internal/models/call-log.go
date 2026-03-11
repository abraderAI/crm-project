package models

// CallDirection represents the direction of a call.
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// IsValid checks if the call direction is a recognized value.
func (d CallDirection) IsValid() bool {
	return d == CallDirectionInbound || d == CallDirectionOutbound
}

// CallStatus represents the current status of a call.
type CallStatus string

const (
	CallStatusRinging   CallStatus = "ringing"
	CallStatusActive    CallStatus = "active"
	CallStatusCompleted CallStatus = "completed"
	CallStatusFailed    CallStatus = "failed"
	CallStatusEscalated CallStatus = "escalated"
)

// IsValid checks if the call status is a recognized value.
func (s CallStatus) IsValid() bool {
	switch s {
	case CallStatusRinging, CallStatusActive, CallStatusCompleted,
		CallStatusFailed, CallStatusEscalated:
		return true
	}
	return false
}

// CallLog records a voice call and its metadata.
type CallLog struct {
	BaseModel
	OrgID      string        `gorm:"type:text;not null;index" json:"org_id"`
	ThreadID   string        `gorm:"type:text;index" json:"thread_id,omitempty"`
	CallerID   string        `gorm:"type:text;not null" json:"caller_id"`
	Direction  CallDirection `gorm:"type:text;not null;default:'inbound'" json:"direction"`
	Duration   int           `gorm:"default:0" json:"duration"`
	Status     CallStatus    `gorm:"type:text;not null;default:'completed'" json:"status"`
	Transcript string        `gorm:"type:text" json:"transcript,omitempty"`
	Metadata   string        `gorm:"type:text;default:'{}'" json:"metadata"`

	// Associations.
	Org Org `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
}
