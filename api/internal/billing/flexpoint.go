package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FlexPointProvider implements BillingProvider for FlexPoint billing.
type FlexPointProvider struct {
	webhookSecret string
}

// NewFlexPointProvider creates a new FlexPoint billing provider.
func NewFlexPointProvider(webhookSecret string) *FlexPointProvider {
	return &FlexPointProvider{webhookSecret: webhookSecret}
}

// CreateCustomer creates a billing customer via FlexPoint.
func (f *FlexPointProvider) CreateCustomer(_ context.Context, input CreateCustomerInput) (*Customer, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("customer name is required")
	}
	if input.OrgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating customer id: %w", err)
	}

	return &Customer{
		ID:         id.String(),
		OrgID:      input.OrgID,
		ExternalID: fmt.Sprintf("fp_cust_%s", id.String()[:8]),
		Name:       input.Name,
		Email:      input.Email,
	}, nil
}

// CreateInvoice creates an invoice via FlexPoint.
func (f *FlexPointProvider) CreateInvoice(_ context.Context, input CreateInvoiceInput) (*Invoice, error) {
	if input.CustomerID == "" {
		return nil, fmt.Errorf("customer_id is required")
	}
	if input.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if input.Currency == "" {
		return nil, fmt.Errorf("currency is required")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating invoice id: %w", err)
	}

	return &Invoice{
		ID:          id.String(),
		CustomerID:  input.CustomerID,
		Amount:      input.Amount,
		Currency:    input.Currency,
		Status:      StatusPending,
		Description: input.Description,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// GetPaymentStatus retrieves the payment status for a customer from FlexPoint.
func (f *FlexPointProvider) GetPaymentStatus(_ context.Context, customerID string) (*PaymentStatus, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customer_id is required")
	}

	// In a real implementation, this would call FlexPoint API.
	// For now, return a default status.
	return &PaymentStatus{
		CustomerID:  customerID,
		Status:      StatusActive,
		BillingTier: TierFree,
	}, nil
}

// HandleWebhook processes and validates an incoming FlexPoint webhook event.
func (f *FlexPointProvider) HandleWebhook(_ context.Context, payload WebhookPayload) (*WebhookResult, error) {
	if f.webhookSecret != "" && payload.Signature != "" {
		if !f.verifySignature(payload.RawBody, payload.Signature) {
			return nil, fmt.Errorf("invalid webhook signature")
		}
	}

	if payload.EventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}

	action := mapEventToAction(payload.EventType)
	if action == "" {
		return &WebhookResult{
			Processed: false,
			EventType: payload.EventType,
			OrgID:     payload.OrgID,
			Action:    "ignored",
		}, nil
	}

	return &WebhookResult{
		Processed: true,
		EventType: payload.EventType,
		OrgID:     payload.OrgID,
		Action:    action,
	}, nil
}

// verifySignature checks the HMAC-SHA256 signature of a webhook payload.
func (f *FlexPointProvider) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(f.webhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyWebhookSignature is exported for testing and external use.
func VerifyWebhookSignature(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ComputeWebhookSignature computes the HMAC-SHA256 signature for a payload.
func ComputeWebhookSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// mapEventToAction maps a FlexPoint event type to an internal action.
func mapEventToAction(eventType string) string {
	switch eventType {
	case EventPaymentSucceeded:
		return "update_payment_status_active"
	case EventPaymentFailed:
		return "update_payment_status_past_due"
	case EventSubscriptionCreated:
		return "update_billing_tier"
	case EventSubscriptionCanceled:
		return "cancel_subscription"
	case EventInvoicePaid:
		return "mark_invoice_paid"
	case EventInvoiceOverdue:
		return "mark_invoice_overdue"
	case EventCustomerCreated:
		return "link_customer"
	default:
		return ""
	}
}

// MapEventToMetadata returns the metadata updates for a given webhook event.
func MapEventToMetadata(result *WebhookResult, payload WebhookPayload) map[string]any {
	updates := make(map[string]any)

	switch payload.EventType {
	case EventPaymentSucceeded:
		updates["payment_status"] = StatusActive
		updates["last_payment_at"] = time.Now().UTC().Format(time.RFC3339)
	case EventPaymentFailed:
		updates["payment_status"] = StatusPastDue
	case EventSubscriptionCreated:
		tier := extractTierFromData(payload.Data)
		if tier != "" {
			updates["billing_tier"] = tier
		}
		updates["payment_status"] = StatusActive
	case EventSubscriptionCanceled:
		updates["payment_status"] = StatusCanceled
		updates["billing_tier"] = TierFree
	case EventInvoicePaid:
		updates["payment_status"] = StatusActive
	case EventInvoiceOverdue:
		updates["payment_status"] = StatusPastDue
	case EventCustomerCreated:
		updates["billing_customer_id"] = payload.CustomerID
	}

	return updates
}

// extractTierFromData extracts the billing tier from webhook event data JSON.
func extractTierFromData(data string) string {
	if data == "" {
		return ""
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return ""
	}
	if tier, ok := parsed["billing_tier"].(string); ok {
		return tier
	}
	if tier, ok := parsed["tier"].(string); ok {
		return tier
	}
	return ""
}
