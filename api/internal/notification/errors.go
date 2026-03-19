package notification

import "errors"

// Sentinel errors for notification validation.
var (
	ErrUserIDRequired = errors.New("user_id is required")
	ErrTypeRequired   = errors.New("type is required")
	ErrTitleRequired  = errors.New("title is required")
	ErrNotFound       = errors.New("notification not found")
)

// Notification event types.
const (
	TypeNewMessage           = "new_message"
	TypeMention              = "mention"
	TypeStageChange          = "stage_change"
	TypeAssignment           = "assignment"
	TypeInvite               = "invite"
	TypeDigest               = "digest"
	TypeSupportTicketUpdated = "support_ticket.updated"
)

// Notification channels.
const (
	ChannelInApp = "in_app"
	ChannelEmail = "email"
)
