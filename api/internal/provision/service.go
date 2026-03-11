// Package provision provides automated customer provisioning when a CRM
// lead reaches the closed_won stage.
package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/billing"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service provides automated provisioning for closed_won leads.
type Service struct {
	db              *gorm.DB
	billingProvider billing.BillingProvider
	eventBus        *event.Bus
}

// NewService creates a new provisioning service.
func NewService(db *gorm.DB, billingProvider billing.BillingProvider, eventBus *event.Bus) *Service {
	return &Service{db: db, billingProvider: billingProvider, eventBus: eventBus}
}

// ProvisionResult holds the outcome of a provisioning operation.
type ProvisionResult struct {
	CustomerOrgID   string   `json:"customer_org_id"`
	CustomerOrgSlug string   `json:"customer_org_slug"`
	SpacesCreated   []string `json:"spaces_created"`
	BoardsCreated   []string `json:"boards_created"`
	BillingCustomer string   `json:"billing_customer_id,omitempty"`
	CRMThreadID     string   `json:"crm_thread_id"`
	Message         string   `json:"message"`
}

// ProvisionInput holds the data needed to trigger provisioning.
type ProvisionInput struct {
	CompanyName  string `json:"company_name"`
	ContactEmail string `json:"contact_email"`
}

// ProvisionCustomer creates a new customer org with default structure from a closed_won thread.
func (s *Service) ProvisionCustomer(ctx context.Context, threadID, userID string, input ProvisionInput) (*ProvisionResult, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}

	// Look up the thread.
	var thread models.Thread
	if err := s.db.WithContext(ctx).First(&thread, "id = ?", threadID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("thread not found")
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}

	// Extract metadata.
	meta := parseMetadata(thread.Metadata)

	// Verify stage is closed_won.
	stage, _ := meta["stage"].(string)
	if stage != "closed_won" {
		return nil, fmt.Errorf("thread must be in closed_won stage to provision (current: %s)", stage)
	}

	// Check if already provisioned.
	if _, ok := meta["customer_org_id"]; ok {
		return nil, fmt.Errorf("customer already provisioned for this thread")
	}

	// Resolve company name from input or metadata.
	companyName := input.CompanyName
	if companyName == "" {
		if cn, ok := meta["company"].(string); ok && cn != "" {
			companyName = cn
		} else {
			companyName = thread.Title
		}
	}

	contactEmail := input.ContactEmail
	if contactEmail == "" {
		if ce, ok := meta["contact_email"].(string); ok {
			contactEmail = ce
		}
	}

	// Execute provisioning in a transaction.
	var result *ProvisionResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = s.executeProvisioning(ctx, tx, thread, companyName, contactEmail, userID)
		return err
	})

	if txErr != nil {
		return nil, fmt.Errorf("provisioning failed: %w", txErr)
	}

	// Publish event.
	if s.eventBus != nil {
		payload, _ := json.Marshal(result)
		s.eventBus.Publish(event.Event{
			Type:       event.CustomerProvisioned,
			EntityType: "thread",
			EntityID:   thread.ID,
			UserID:     userID,
			Payload:    string(payload),
		})
	}

	return result, nil
}

// executeProvisioning performs the actual provisioning within a transaction.
func (s *Service) executeProvisioning(ctx context.Context, tx *gorm.DB, thread models.Thread, companyName, contactEmail, userID string) (*ProvisionResult, error) {
	result := &ProvisionResult{CRMThreadID: thread.ID}

	// 1. Create customer org.
	orgSlug := slug.Generate(companyName)
	customerOrg := &models.Org{
		Name:        companyName,
		Slug:        orgSlug,
		Description: fmt.Sprintf("Customer org provisioned from lead: %s", thread.Title),
		Metadata:    `{"provisioned_from":"crm","source_thread_id":"` + thread.ID + `"}`,
	}
	if err := tx.WithContext(ctx).Create(customerOrg).Error; err != nil {
		return nil, fmt.Errorf("creating customer org: %w", err)
	}
	result.CustomerOrgID = customerOrg.ID
	result.CustomerOrgSlug = customerOrg.Slug

	// 2. Create default spaces.
	defaultSpaces := []struct {
		name   string
		spType models.SpaceType
		boards []string
	}{
		{"Support", models.SpaceTypeSupport, []string{"General Support", "Bug Reports"}},
		{"Community", models.SpaceTypeCommunity, []string{"Discussions", "Feature Requests"}},
		{"Knowledge Base", models.SpaceTypeKnowledgeBase, []string{"Getting Started", "API Documentation"}},
	}

	for _, sp := range defaultSpaces {
		space := &models.Space{
			OrgID:       customerOrg.ID,
			Name:        sp.name,
			Slug:        slug.Generate(sp.name),
			Description: fmt.Sprintf("Default %s space", sp.name),
			Metadata:    "{}",
			Type:        sp.spType,
		}
		if err := tx.WithContext(ctx).Create(space).Error; err != nil {
			return nil, fmt.Errorf("creating space %s: %w", sp.name, err)
		}
		result.SpacesCreated = append(result.SpacesCreated, space.ID)

		// 3. Create default boards for each space.
		for _, boardName := range sp.boards {
			board := &models.Board{
				SpaceID:     space.ID,
				Name:        boardName,
				Slug:        slug.Generate(boardName),
				Description: fmt.Sprintf("Default board: %s", boardName),
				Metadata:    "{}",
			}
			if err := tx.WithContext(ctx).Create(board).Error; err != nil {
				return nil, fmt.Errorf("creating board %s: %w", boardName, err)
			}
			result.BoardsCreated = append(result.BoardsCreated, board.ID)
		}
	}

	// 4. Create FlexPoint customer via billing provider.
	if s.billingProvider != nil && contactEmail != "" {
		customer, err := s.billingProvider.CreateCustomer(ctx, billing.CreateCustomerInput{
			OrgID: customerOrg.ID,
			Name:  companyName,
			Email: contactEmail,
		})
		if err == nil && customer != nil {
			result.BillingCustomer = customer.ExternalID
			// Update customer org metadata with billing info.
			billingMeta := map[string]any{
				"billing_customer_id": customer.ExternalID,
				"billing_tier":        "free",
				"payment_status":      "pending",
			}
			updateJSON, _ := json.Marshal(billingMeta)
			merged, _ := metadata.DeepMerge(customerOrg.Metadata, string(updateJSON))
			customerOrg.Metadata = merged
			_ = tx.WithContext(ctx).Save(customerOrg).Error
		}
	}

	// 5. Update CRM thread metadata with customer_org_id.
	threadUpdates := map[string]any{
		"customer_org_id":   customerOrg.ID,
		"customer_org_slug": customerOrg.Slug,
		"provisioned_at":    time.Now().UTC().Format(time.RFC3339),
		"provisioned_by":    userID,
	}
	updateJSON, _ := json.Marshal(threadUpdates)
	merged, _ := metadata.DeepMerge(thread.Metadata, string(updateJSON))
	thread.Metadata = merged
	if err := tx.WithContext(ctx).Save(&thread).Error; err != nil {
		return nil, fmt.Errorf("updating thread metadata: %w", err)
	}

	// 6. Post confirmation message to CRM thread.
	confirmMsg := &models.Message{
		ThreadID: thread.ID,
		Body:     fmt.Sprintf("✅ Customer org **%s** has been provisioned with default spaces and boards.", companyName),
		AuthorID: userID,
		Metadata: `{"type":"system","action":"provision_complete"}`,
		Type:     models.MessageTypeSystem,
	}
	if err := tx.WithContext(ctx).Create(confirmMsg).Error; err != nil {
		return nil, fmt.Errorf("creating confirmation message: %w", err)
	}

	result.Message = fmt.Sprintf("Customer org '%s' provisioned successfully", companyName)
	return result, nil
}

// HandleStageChanged is an event handler that auto-provisions on closed_won.
func (s *Service) HandleStageChanged(evt event.Event) {
	if evt.EntityType != "thread" || evt.EntityID == "" {
		return
	}

	// Parse the event payload to check the new stage.
	var result struct {
		NewStage string `json:"new_stage"`
	}
	if err := json.Unmarshal([]byte(evt.Payload), &result); err != nil {
		return
	}

	if result.NewStage != "closed_won" {
		return
	}

	// Auto-provision with empty input (will use metadata defaults).
	_, _ = s.ProvisionCustomer(context.Background(), evt.EntityID, evt.UserID, ProvisionInput{})
}

// parseMetadata parses metadata JSON into a map.
func parseMetadata(metadataJSON string) map[string]any {
	meta := make(map[string]any)
	if metadataJSON != "" && metadataJSON != "{}" {
		_ = json.Unmarshal([]byte(metadataJSON), &meta)
	}
	return meta
}
