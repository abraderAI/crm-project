package models

// MessageType represents the kind of message.
type MessageType string

const (
	MessageTypeNote    MessageType = "note"
	MessageTypeEmail   MessageType = "email"
	MessageTypeCallLog MessageType = "call_log"
	MessageTypeComment MessageType = "comment"
	MessageTypeSystem  MessageType = "system"
)

// ValidMessageTypes returns all valid message type values.
func ValidMessageTypes() []MessageType {
	return []MessageType{
		MessageTypeNote,
		MessageTypeEmail,
		MessageTypeCallLog,
		MessageTypeComment,
		MessageTypeSystem,
	}
}

// IsValid checks if the message type is a recognized value.
func (m MessageType) IsValid() bool {
	for _, v := range ValidMessageTypes() {
		if m == v {
			return true
		}
	}
	return false
}

// Message represents a single message within a Thread.
type Message struct {
	BaseModel
	ThreadID string      `gorm:"type:text;not null;index" json:"thread_id"`
	Body     string      `gorm:"type:text;not null" json:"body"`
	AuthorID string      `gorm:"type:text;not null;index" json:"author_id"`
	Metadata string      `gorm:"type:text;default:'{}'" json:"metadata"`
	Type     MessageType `gorm:"type:text;not null;default:'comment'" json:"type"`

	// Associations.
	Thread Thread `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"-"`
}
