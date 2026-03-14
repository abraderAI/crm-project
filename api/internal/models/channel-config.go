package models

import "time"

// ChannelType represents the type of IO channel.
type ChannelType string

const (
	// ChannelTypeEmail is the inbound email (Gmail IMAP) channel.
	ChannelTypeEmail ChannelType = "email"
	// ChannelTypeVoice is the voice call (LiveKit) channel.
	ChannelTypeVoice ChannelType = "voice"
	// ChannelTypeChat is the embeddable web chat channel.
	ChannelTypeChat ChannelType = "chat"
)

// ValidChannelTypes returns all valid channel type values.
func ValidChannelTypes() []ChannelType {
	return []ChannelType{ChannelTypeEmail, ChannelTypeVoice, ChannelTypeChat}
}

// IsValid checks if the channel type is a recognized value.
func (c ChannelType) IsValid() bool {
	for _, v := range ValidChannelTypes() {
		if c == v {
			return true
		}
	}
	return false
}

// ChannelConfig stores per-org configuration for a single IO channel.
// The composite unique index on (OrgID, ChannelType) ensures one config per org per channel.
type ChannelConfig struct {
	BaseModel
	OrgID       string      `gorm:"type:text;not null;uniqueIndex:idx_channel_config_org_type" json:"org_id"`
	ChannelType ChannelType `gorm:"type:text;not null;uniqueIndex:idx_channel_config_org_type" json:"channel_type"`
	// Settings holds channel-specific configuration as a JSONB string.
	// Secrets within Settings are masked in API responses.
	Settings string `gorm:"type:text;default:'{}'" json:"settings"`
	Enabled  bool   `gorm:"default:false" json:"enabled"`

	// Associations.
	Org Org `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
}

// DLQStatus represents the lifecycle status of a dead letter queue event.
type DLQStatus string

const (
	// DLQStatusFailed indicates the event has failed all retry attempts.
	DLQStatusFailed DLQStatus = "failed"
	// DLQStatusRetrying indicates a manual retry has been requested.
	DLQStatusRetrying DLQStatus = "retrying"
	// DLQStatusResolved indicates the event was successfully reprocessed.
	DLQStatusResolved DLQStatus = "resolved"
	// DLQStatusDismissed indicates the event was dismissed without reprocessing.
	DLQStatusDismissed DLQStatus = "dismissed"
)

// ValidDLQStatuses returns all valid DLQ status values.
func ValidDLQStatuses() []DLQStatus {
	return []DLQStatus{DLQStatusFailed, DLQStatusRetrying, DLQStatusResolved, DLQStatusDismissed}
}

// IsValid checks if the DLQ status is a recognized value.
func (s DLQStatus) IsValid() bool {
	for _, v := range ValidDLQStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// DeadLetterEvent stores a failed inbound channel event for later retry or dismissal.
// Events are inserted after exhausting all retry attempts.
type DeadLetterEvent struct {
	BaseModel
	OrgID        string      `gorm:"type:text;not null;index:idx_dlq_org_status" json:"org_id"`
	ChannelType  ChannelType `gorm:"type:text;not null" json:"channel_type"`
	EventPayload string      `gorm:"type:text;default:'{}'" json:"event_payload"`
	ErrorMessage string      `gorm:"type:text" json:"error_message"`
	Attempts     int         `gorm:"default:0" json:"attempts"`
	// LastAttemptAt is the timestamp of the most recent retry attempt.
	LastAttemptAt *time.Time `gorm:"index" json:"last_attempt_at,omitempty"`
	Status        DLQStatus  `gorm:"type:text;not null;default:'failed';index:idx_dlq_org_status" json:"status"`
}
