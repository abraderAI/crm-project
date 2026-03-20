package models

// RoutingAction determines how an inbound email is routed within the CRM.
type RoutingAction string

const (
	// RoutingActionSupportTicket routes the email to the org's support space.
	RoutingActionSupportTicket RoutingAction = "support_ticket"
	// RoutingActionSalesLead routes the email to the org's CRM (sales) space.
	RoutingActionSalesLead RoutingAction = "sales_lead"
	// RoutingActionGeneral routes the email to a general-purpose space.
	RoutingActionGeneral RoutingAction = "general"
)

// ValidRoutingActions returns all valid routing action values.
func ValidRoutingActions() []RoutingAction {
	return []RoutingAction{
		RoutingActionSupportTicket,
		RoutingActionSalesLead,
		RoutingActionGeneral,
	}
}

// IsValid reports whether the routing action is a recognised value.
func (a RoutingAction) IsValid() bool {
	for _, v := range ValidRoutingActions() {
		if a == v {
			return true
		}
	}
	return false
}

// EmailInbox stores the IMAP credentials and routing configuration for a
// single inbound email address. An org may have multiple inboxes
// (e.g. support@, sales@), each with independent routing behaviour.
type EmailInbox struct {
	BaseModel
	// OrgID is the owning organisation.
	OrgID string `gorm:"type:text;not null;index" json:"org_id"`
	// Name is a human-readable label (e.g. "Support", "Sales").
	Name string `gorm:"type:text;not null" json:"name"`
	// EmailAddress is the display address for this inbox (e.g. "support@acme.com").
	EmailAddress string `gorm:"type:text" json:"email_address"`
	// IMAPHost is the IMAP server hostname.
	IMAPHost string `gorm:"type:text;not null" json:"imap_host"`
	// IMAPPort is the IMAP server port (typically 993 for implicit TLS).
	IMAPPort int `gorm:"default:993" json:"imap_port"`
	// Username is the IMAP login username (usually the email address).
	Username string `gorm:"type:text;not null" json:"username"`
	// Password is the IMAP login secret (app password or OAuth token).
	// Always masked as "[REDACTED]" in API responses.
	Password string `gorm:"type:text" json:"password,omitempty"`
	// Mailbox is the IMAP mailbox to monitor (default: "INBOX").
	Mailbox string `gorm:"type:text;default:'INBOX'" json:"mailbox"`
	// RoutingAction determines which space type receives new threads.
	RoutingAction RoutingAction `gorm:"type:text;not null;default:'support_ticket'" json:"routing_action"`
	// Enabled controls whether the IMAP watcher is active for this inbox.
	Enabled bool `gorm:"default:true" json:"enabled"`

	// Associations.
	Org Org `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
}
