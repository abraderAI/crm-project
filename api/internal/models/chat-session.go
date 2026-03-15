package models

import "time"

// ChatSession represents an anonymous chat widget session.
// Sessions are created when a visitor opens the chat widget and
// are authenticated via a short-lived HMAC JWT (24 hours).
type ChatSession struct {
	BaseModel
	// OrgID is the organization this session belongs to.
	OrgID string `gorm:"type:text;not null;index" json:"org_id"`
	// EmbedKey is the public widget key used to create this session.
	EmbedKey string `gorm:"type:text;not null;index" json:"embed_key"`
	// FingerprintHash is the browser fingerprint hash for visitor tracking.
	FingerprintHash string `gorm:"type:text;not null;index" json:"fingerprint_hash"`
	// VisitorID links to the ChatVisitor record for returning visitors.
	VisitorID string `gorm:"type:text;not null;index" json:"visitor_id"`
	// ThreadID is the CRM thread associated with this chat session.
	ThreadID string `gorm:"type:text" json:"thread_id,omitempty"`
	// Escalated is true when the chat has been escalated to a human agent.
	Escalated bool `gorm:"default:false" json:"escalated"`
	// ExpiresAt is when this session's JWT expires.
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}

// ChatVisitor tracks unique visitors across sessions using browser fingerprints.
// When a returning visitor is detected, their previous session and lead data
// can be retrieved for continuity.
type ChatVisitor struct {
	BaseModel
	// OrgID scopes the visitor to an organization.
	OrgID string `gorm:"type:text;not null;uniqueIndex:idx_visitor_org_fp" json:"org_id"`
	// FingerprintHash is the unique browser fingerprint hash.
	FingerprintHash string `gorm:"type:text;not null;uniqueIndex:idx_visitor_org_fp" json:"fingerprint_hash"`
	// ContactEmail is captured when the visitor provides their email.
	ContactEmail string `gorm:"type:text" json:"contact_email,omitempty"`
	// ContactName is captured when the visitor provides their name.
	ContactName string `gorm:"type:text" json:"contact_name,omitempty"`
	// LastSessionID references the most recent session.
	LastSessionID string `gorm:"type:text" json:"last_session_id,omitempty"`
	// LastThreadID references the most recent CRM thread.
	LastThreadID string `gorm:"type:text" json:"last_thread_id,omitempty"`
}
