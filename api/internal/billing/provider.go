// Package billing provides the billing module with a provider-abstracted
// interface for billing operations (FlexPoint → Stripe swappable).
package billing

import (
	"context"
	"time"
)

// BillingProvider defines the interface for billing operations.
// All external billing integrations MUST implement this interface.
type BillingProvider interface {
	// CreateCustomer creates a billing customer for an org.
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (*Customer, error)
	// CreateInvoice creates an invoice for a customer.
	CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*Invoice, error)
	// GetPaymentStatus retrieves the payment status for a customer.
	GetPaymentStatus(ctx context.Context, customerID string) (*PaymentStatus, error)
	// HandleWebhook processes an incoming billing webhook event.
	HandleWebhook(ctx context.Context, payload WebhookPayload) (*WebhookResult, error)
}

// CreateCustomerInput holds data needed to create a billing customer.
type CreateCustomerInput struct {
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Customer represents a billing customer record.
type Customer struct {
	ID         string `json:"id"`
	OrgID      string `json:"org_id"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

// CreateInvoiceInput holds data needed to create an invoice.
type CreateInvoiceInput struct {
	CustomerID  string  `json:"customer_id"`
	OrgID       string  `json:"org_id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

// Invoice represents a billing invoice.
type Invoice struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// PaymentStatus represents the current payment status for a customer.
type PaymentStatus struct {
	CustomerID    string `json:"customer_id"`
	Status        string `json:"status"`
	BillingTier   string `json:"billing_tier"`
	LastPaymentAt string `json:"last_payment_at,omitempty"`
}

// WebhookPayload represents an incoming billing webhook event.
type WebhookPayload struct {
	EventType  string `json:"event_type"`
	CustomerID string `json:"customer_id"`
	OrgID      string `json:"org_id"`
	Data       string `json:"data"`
	Signature  string `json:"-"`
	RawBody    []byte `json:"-"`
}

// WebhookResult represents the outcome of processing a webhook event.
type WebhookResult struct {
	Processed bool   `json:"processed"`
	EventType string `json:"event_type"`
	OrgID     string `json:"org_id"`
	Action    string `json:"action"`
}

// Billing event type constants.
const (
	EventPaymentSucceeded     = "payment.succeeded"
	EventPaymentFailed        = "payment.failed"
	EventSubscriptionCreated  = "subscription.created"
	EventSubscriptionCanceled = "subscription.canceled"
	EventInvoicePaid          = "invoice.paid"
	EventInvoiceOverdue       = "invoice.overdue"
	EventCustomerCreated      = "customer.created"
)

// Payment status constants.
const (
	StatusActive   = "active"
	StatusPastDue  = "past_due"
	StatusCanceled = "canceled"
	StatusPending  = "pending"
)

// Billing tier constants.
const (
	TierFree       = "free"
	TierStarter    = "starter"
	TierPro        = "pro"
	TierEnterprise = "enterprise"
)
