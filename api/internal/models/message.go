package models

import "time"

// MessageType represents the kind of message.
type MessageType string

const (
	// Generic message types (used across all thread types).
	MessageTypeNote    MessageType = "note"
	MessageTypeEmail   MessageType = "email"
	MessageTypeCallLog MessageType = "call_log"
	MessageTypeComment MessageType = "comment"
	MessageTypeSystem  MessageType = "system"

	// Support-specific entry types.

	// MessageTypeCustomer is a message or comment from the ticket creator, added
	// manually or ingested from an inbound customer email.
	MessageTypeCustomer MessageType = "customer"

	// MessageTypeAgentReply is a published reply from a DEFT support agent,
	// visible to the customer and immutable once posted.
	MessageTypeAgentReply MessageType = "agent_reply"

	// MessageTypeDraft is an unpublished agent reply draft, invisible to
	// customers until explicitly published (which converts it to agent_reply).
	MessageTypeDraft MessageType = "draft"

	// MessageTypeContext is a DEFT-internal note, never visible to customers
	// outside the DEFT org. May be written by humans or AI.
	MessageTypeContext MessageType = "context"

	// MessageTypeSystemEvent is a system-generated audit entry such as
	// "ticket created", "ticket closed", or "ticket reopened".
	MessageTypeSystemEvent MessageType = "system_event"
)

// ValidMessageTypes returns all valid message type values.
func ValidMessageTypes() []MessageType {
	return []MessageType{
		MessageTypeNote,
		MessageTypeEmail,
		MessageTypeCallLog,
		MessageTypeComment,
		MessageTypeSystem,
		MessageTypeCustomer,
		MessageTypeAgentReply,
		MessageTypeDraft,
		MessageTypeContext,
		MessageTypeSystemEvent,
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

// IsSupportType returns true when the message type is one of the support-specific
// entry types (customer, agent_reply, draft, context, system_event).
func (m MessageType) IsSupportType() bool {
	switch m {
	case MessageTypeCustomer, MessageTypeAgentReply, MessageTypeDraft,
		MessageTypeContext, MessageTypeSystemEvent:
		return true
	}
	return false
}

// IsVisibleToCustomer returns true when entries of this type are shown to
// non-DEFT customers by default (subject to IsDeftOnly override).
func (m MessageType) IsVisibleToCustomer() bool {
	switch m {
	case MessageTypeCustomer, MessageTypeAgentReply, MessageTypeSystemEvent:
		return true
	}
	return false
}

// Message represents a single message or support entry within a Thread.
type Message struct {
	BaseModel
	ThreadID string      `gorm:"type:text;not null;index" json:"thread_id"`
	Body     string      `gorm:"type:text;not null" json:"body"`
	AuthorID string      `gorm:"type:text;not null;index" json:"author_id"`
	Metadata string      `gorm:"type:text;default:'{}'" json:"metadata"`
	Type     MessageType `gorm:"type:text;not null;default:'comment'" json:"type"`

	// Support entry lifecycle fields.

	// IsDeftOnly, when true, instantly hides this entry from any user outside
	// the DEFT org regardless of entry type. Togglable by DEFT agents at any time.
	IsDeftOnly bool `gorm:"default:false" json:"is_deft_only"`

	// IsPublished marks whether an agent_reply or draft has been released to
	// the customer. For non-draft types this is always true.
	IsPublished bool `gorm:"default:false" json:"is_published"`

	// IsImmutable prevents further edits. Set automatically when an entry is
	// published or when the type is customer / system_event.
	IsImmutable bool `gorm:"default:false" json:"is_immutable"`

	// PublishedAt records when a draft was promoted to agent_reply.
	PublishedAt *time.Time `json:"published_at,omitempty"`

	// Associations.
	Thread Thread `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"-"`
}
