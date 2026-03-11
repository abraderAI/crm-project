package billing

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
)

// Service provides business logic for billing operations.
type Service struct {
	provider BillingProvider
	db       *gorm.DB
}

// NewService creates a new billing service.
func NewService(provider BillingProvider, db *gorm.DB) *Service {
	return &Service{provider: provider, db: db}
}

// CreateCustomer creates a billing customer for an org.
func (s *Service) CreateCustomer(ctx context.Context, orgIDOrSlug string, input CreateCustomerInput) (*Customer, error) {
	org, err := s.findOrg(ctx, orgIDOrSlug)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, fmt.Errorf("org not found")
	}

	input.OrgID = org.ID
	customer, err := s.provider.CreateCustomer(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("creating customer: %w", err)
	}

	// Update org metadata with billing customer ID.
	metaUpdates := map[string]any{
		"billing_customer_id": customer.ExternalID,
	}
	if err := s.updateOrgMetadata(ctx, org, metaUpdates); err != nil {
		return nil, fmt.Errorf("updating org metadata: %w", err)
	}

	return customer, nil
}

// CreateInvoice creates an invoice for an org's billing customer.
func (s *Service) CreateInvoice(ctx context.Context, orgIDOrSlug string, input CreateInvoiceInput) (*Invoice, error) {
	org, err := s.findOrg(ctx, orgIDOrSlug)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, fmt.Errorf("org not found")
	}

	input.OrgID = org.ID
	invoice, err := s.provider.CreateInvoice(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("creating invoice: %w", err)
	}

	return invoice, nil
}

// GetBillingStatus retrieves the billing status for an org.
func (s *Service) GetBillingStatus(ctx context.Context, orgIDOrSlug string) (*BillingStatus, error) {
	org, err := s.findOrg(ctx, orgIDOrSlug)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, fmt.Errorf("org not found")
	}

	// Extract billing info from org metadata.
	var meta map[string]any
	if err := json.Unmarshal([]byte(org.Metadata), &meta); err != nil {
		meta = make(map[string]any)
	}

	status := &BillingStatus{
		OrgID:         org.ID,
		BillingTier:   stringFromMeta(meta, "billing_tier", TierFree),
		PaymentStatus: stringFromMeta(meta, "payment_status", StatusPending),
		CustomerID:    stringFromMeta(meta, "billing_customer_id", ""),
	}

	// If we have a customer ID, try to get live status from provider.
	if status.CustomerID != "" {
		ps, err := s.provider.GetPaymentStatus(ctx, status.CustomerID)
		if err == nil && ps != nil {
			status.PaymentStatus = ps.Status
			status.BillingTier = ps.BillingTier
		}
	}

	return status, nil
}

// ProcessWebhook handles an incoming billing webhook and updates org metadata.
func (s *Service) ProcessWebhook(ctx context.Context, payload WebhookPayload) (*WebhookResult, error) {
	result, err := s.provider.HandleWebhook(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("processing webhook: %w", err)
	}

	if !result.Processed {
		return result, nil
	}

	// Update org metadata based on webhook event.
	if payload.OrgID != "" {
		org, err := s.findOrg(ctx, payload.OrgID)
		if err != nil {
			return nil, fmt.Errorf("finding org for webhook: %w", err)
		}
		if org != nil {
			metaUpdates := MapEventToMetadata(result, payload)
			if len(metaUpdates) > 0 {
				if err := s.updateOrgMetadata(ctx, org, metaUpdates); err != nil {
					return nil, fmt.Errorf("updating org metadata from webhook: %w", err)
				}
			}
		}
	}

	return result, nil
}

// BillingStatus represents the billing status response for an org.
type BillingStatus struct {
	OrgID         string `json:"org_id"`
	BillingTier   string `json:"billing_tier"`
	PaymentStatus string `json:"payment_status"`
	CustomerID    string `json:"customer_id,omitempty"`
}

// findOrg retrieves an org by ID or slug.
func (s *Service) findOrg(ctx context.Context, idOrSlug string) (*models.Org, error) {
	var org models.Org
	query := s.db.WithContext(ctx)

	// Try as UUID first, fall back to slug.
	result := query.Where("id = ? OR slug = ?", idOrSlug, idOrSlug).First(&org)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("finding org: %w", result.Error)
	}
	return &org, nil
}

// updateOrgMetadata deep-merges updates into the org's existing metadata.
func (s *Service) updateOrgMetadata(ctx context.Context, org *models.Org, updates map[string]any) error {
	updateJSON, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("marshaling metadata updates: %w", err)
	}

	merged, err := metadata.DeepMerge(org.Metadata, string(updateJSON))
	if err != nil {
		return fmt.Errorf("merging metadata: %w", err)
	}

	org.Metadata = merged
	if err := s.db.WithContext(ctx).Save(org).Error; err != nil {
		return fmt.Errorf("saving org: %w", err)
	}
	return nil
}

// stringFromMeta extracts a string value from metadata map with a default.
func stringFromMeta(meta map[string]any, key, defaultVal string) string {
	if val, ok := meta[key].(string); ok && val != "" {
		return val
	}
	return defaultVal
}
