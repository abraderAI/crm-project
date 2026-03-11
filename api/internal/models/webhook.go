package models

import "time"

// WebhookSubscription defines a webhook endpoint subscribed to events.
type WebhookSubscription struct {
	BaseModel
	OrgID       string `gorm:"type:text;not null;index" json:"org_id"`
	ScopeType   string `gorm:"type:text;not null" json:"scope_type"`
	ScopeID     string `gorm:"type:text;not null" json:"scope_id"`
	URL         string `gorm:"type:text;not null" json:"url"`
	Secret      string `gorm:"type:text;not null" json:"-"`
	EventFilter string `gorm:"type:text;default:'[]'" json:"event_filter"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`

	// Associations.
	Org        Org               `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE" json:"-"`
	Deliveries []WebhookDelivery `gorm:"foreignKey:SubscriptionID;constraint:OnDelete:CASCADE" json:"deliveries,omitempty"`
}

// WebhookDelivery records each delivery attempt for a webhook event.
type WebhookDelivery struct {
	BaseModel
	SubscriptionID string     `gorm:"type:text;not null;index" json:"subscription_id"`
	EventType      string     `gorm:"type:text;not null" json:"event_type"`
	Payload        string     `gorm:"type:text;not null" json:"payload"`
	StatusCode     int        `json:"status_code"`
	ResponseBody   string     `gorm:"type:text" json:"response_body,omitempty"`
	Attempts       int        `gorm:"default:0" json:"attempts"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`

	// Associations.
	Subscription WebhookSubscription `gorm:"foreignKey:SubscriptionID;constraint:OnDelete:CASCADE" json:"-"`
}
