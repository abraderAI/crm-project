package models

// TicketCounter tracks the last-issued ticket number per org and thread type.
// It is used to atomically assign monotonically increasing, human-readable ticket
// numbers to support threads. The primary key is (OrgID, ThreadType) so each org
// maintains its own sequence.
//
// Use "_system" as OrgID for tickets that are not scoped to a customer org
// (e.g. from tier-2 users who have no org).
type TicketCounter struct {
	// OrgID is the owning organisation, or "_system" for cross-org tickets.
	OrgID string `gorm:"type:text;primaryKey" json:"org_id"`
	// ThreadType identifies the kind of thread being numbered (e.g. "support").
	ThreadType string `gorm:"type:text;primaryKey" json:"thread_type"`
	// LastNumber is the most recently issued ticket number for this bucket.
	LastNumber int64 `gorm:"default:0" json:"last_number"`
}
