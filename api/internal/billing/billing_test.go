package billing

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// mockProvider implements BillingProvider for testing.
type mockProvider struct {
	createCustomerFn   func(ctx context.Context, input CreateCustomerInput) (*Customer, error)
	createInvoiceFn    func(ctx context.Context, input CreateInvoiceInput) (*Invoice, error)
	getPaymentStatusFn func(ctx context.Context, customerID string) (*PaymentStatus, error)
	handleWebhookFn    func(ctx context.Context, payload WebhookPayload) (*WebhookResult, error)
}

func (m *mockProvider) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*Customer, error) {
	if m.createCustomerFn != nil {
		return m.createCustomerFn(ctx, input)
	}
	return &Customer{
		ID:         "cust-123",
		OrgID:      input.OrgID,
		ExternalID: "fp_cust_test123",
		Name:       input.Name,
		Email:      input.Email,
	}, nil
}

func (m *mockProvider) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*Invoice, error) {
	if m.createInvoiceFn != nil {
		return m.createInvoiceFn(ctx, input)
	}
	return &Invoice{
		ID:          "inv-123",
		CustomerID:  input.CustomerID,
		Amount:      input.Amount,
		Currency:    input.Currency,
		Status:      StatusPending,
		Description: input.Description,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (m *mockProvider) GetPaymentStatus(ctx context.Context, customerID string) (*PaymentStatus, error) {
	if m.getPaymentStatusFn != nil {
		return m.getPaymentStatusFn(ctx, customerID)
	}
	return &PaymentStatus{
		CustomerID:  customerID,
		Status:      StatusActive,
		BillingTier: TierPro,
	}, nil
}

func (m *mockProvider) HandleWebhook(ctx context.Context, payload WebhookPayload) (*WebhookResult, error) {
	if m.handleWebhookFn != nil {
		return m.handleWebhookFn(ctx, payload)
	}
	return &WebhookResult{
		Processed: true,
		EventType: payload.EventType,
		OrgID:     payload.OrgID,
		Action:    "update_payment_status_active",
	}, nil
}

// setupTestDB creates a test database with migrations.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	require.NoError(t, database.Migrate(db))
	return db
}

// createTestOrg creates a test org in the database.
func createTestOrg(t *testing.T, db *gorm.DB, name, slug, meta string) *models.Org {
	t.Helper()
	if meta == "" {
		meta = "{}"
	}
	org := &models.Org{Name: name, Slug: slug, Metadata: meta}
	require.NoError(t, db.Create(org).Error)
	return org
}

// --- FlexPointProvider Tests ---

func TestFlexPointProvider_CreateCustomer_Success(t *testing.T) {
	fp := NewFlexPointProvider("test-secret")
	customer, err := fp.CreateCustomer(context.Background(), CreateCustomerInput{
		OrgID: "org-1",
		Name:  "Test Co",
		Email: "test@example.com",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, customer.ID)
	assert.Equal(t, "org-1", customer.OrgID)
	assert.Equal(t, "Test Co", customer.Name)
	assert.Contains(t, customer.ExternalID, "fp_cust_")
}

func TestFlexPointProvider_CreateCustomer_MissingName(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.CreateCustomer(context.Background(), CreateCustomerInput{OrgID: "org-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer name is required")
}

func TestFlexPointProvider_CreateCustomer_MissingOrgID(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.CreateCustomer(context.Background(), CreateCustomerInput{Name: "Test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org_id is required")
}

func TestFlexPointProvider_CreateInvoice_Success(t *testing.T) {
	fp := NewFlexPointProvider("")
	invoice, err := fp.CreateInvoice(context.Background(), CreateInvoiceInput{
		CustomerID:  "cust-1",
		Amount:      99.99,
		Currency:    "USD",
		Description: "Monthly plan",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, invoice.ID)
	assert.Equal(t, 99.99, invoice.Amount)
	assert.Equal(t, "USD", invoice.Currency)
	assert.Equal(t, StatusPending, invoice.Status)
}

func TestFlexPointProvider_CreateInvoice_MissingCustomerID(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.CreateInvoice(context.Background(), CreateInvoiceInput{Amount: 10, Currency: "USD"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer_id is required")
}

func TestFlexPointProvider_CreateInvoice_ZeroAmount(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.CreateInvoice(context.Background(), CreateInvoiceInput{CustomerID: "c1", Amount: 0, Currency: "USD"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestFlexPointProvider_CreateInvoice_NegativeAmount(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.CreateInvoice(context.Background(), CreateInvoiceInput{CustomerID: "c1", Amount: -5, Currency: "USD"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestFlexPointProvider_CreateInvoice_MissingCurrency(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.CreateInvoice(context.Background(), CreateInvoiceInput{CustomerID: "c1", Amount: 10})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency is required")
}

func TestFlexPointProvider_GetPaymentStatus_Success(t *testing.T) {
	fp := NewFlexPointProvider("")
	status, err := fp.GetPaymentStatus(context.Background(), "cust-1")
	require.NoError(t, err)
	assert.Equal(t, "cust-1", status.CustomerID)
	assert.Equal(t, StatusActive, status.Status)
	assert.Equal(t, TierFree, status.BillingTier)
}

func TestFlexPointProvider_GetPaymentStatus_EmptyCustomerID(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.GetPaymentStatus(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer_id is required")
}

func TestFlexPointProvider_HandleWebhook_PaymentSucceeded(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "update_payment_status_active", result.Action)
}

func TestFlexPointProvider_HandleWebhook_PaymentFailed(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentFailed,
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "update_payment_status_past_due", result.Action)
}

func TestFlexPointProvider_HandleWebhook_SubscriptionCreated(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventSubscriptionCreated,
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "update_billing_tier", result.Action)
}

func TestFlexPointProvider_HandleWebhook_SubscriptionCanceled(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventSubscriptionCanceled,
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "cancel_subscription", result.Action)
}

func TestFlexPointProvider_HandleWebhook_InvoicePaid(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventInvoicePaid,
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "mark_invoice_paid", result.Action)
}

func TestFlexPointProvider_HandleWebhook_InvoiceOverdue(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventInvoiceOverdue,
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "mark_invoice_overdue", result.Action)
}

func TestFlexPointProvider_HandleWebhook_CustomerCreated(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType:  EventCustomerCreated,
		OrgID:      "org-1",
		CustomerID: "cust-1",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
	assert.Equal(t, "link_customer", result.Action)
}

func TestFlexPointProvider_HandleWebhook_UnknownEvent(t *testing.T) {
	fp := NewFlexPointProvider("")
	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: "unknown.event",
		OrgID:     "org-1",
	})
	require.NoError(t, err)
	assert.False(t, result.Processed)
	assert.Equal(t, "ignored", result.Action)
}

func TestFlexPointProvider_HandleWebhook_EmptyEventType(t *testing.T) {
	fp := NewFlexPointProvider("")
	_, err := fp.HandleWebhook(context.Background(), WebhookPayload{OrgID: "org-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event_type is required")
}

func TestFlexPointProvider_HandleWebhook_InvalidSignature(t *testing.T) {
	fp := NewFlexPointProvider("my-secret")
	_, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     "org-1",
		Signature: "bad-signature",
		RawBody:   []byte(`{"event_type":"payment.succeeded"}`),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook signature")
}

func TestFlexPointProvider_HandleWebhook_ValidSignature(t *testing.T) {
	secret := "test-webhook-secret"
	fp := NewFlexPointProvider(secret)
	body := []byte(`{"event_type":"payment.succeeded","org_id":"org-1"}`)
	sig := ComputeWebhookSignature(body, secret)

	result, err := fp.HandleWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     "org-1",
		Signature: sig,
		RawBody:   body,
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
}

// --- Signature Verification Tests ---

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "my-secret"
	body := []byte(`{"test":"data"}`)
	sig := ComputeWebhookSignature(body, secret)
	assert.True(t, VerifyWebhookSignature(body, sig, secret))
}

func TestVerifyWebhookSignature_Invalid(t *testing.T) {
	assert.False(t, VerifyWebhookSignature([]byte("data"), "bad-sig", "secret"))
}

func TestVerifyWebhookSignature_DifferentSecret(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	sig := ComputeWebhookSignature(body, "secret1")
	assert.False(t, VerifyWebhookSignature(body, sig, "secret2"))
}

func TestVerifyWebhookSignature_EmptyBody(t *testing.T) {
	secret := "test"
	sig := ComputeWebhookSignature([]byte{}, secret)
	assert.True(t, VerifyWebhookSignature([]byte{}, sig, secret))
}

func TestComputeWebhookSignature_Deterministic(t *testing.T) {
	body := []byte(`{"data":"test"}`)
	sig1 := ComputeWebhookSignature(body, "secret")
	sig2 := ComputeWebhookSignature(body, "secret")
	assert.Equal(t, sig1, sig2)
}

func TestComputeWebhookSignature_DifferentBodies(t *testing.T) {
	sig1 := ComputeWebhookSignature([]byte("body1"), "secret")
	sig2 := ComputeWebhookSignature([]byte("body2"), "secret")
	assert.NotEqual(t, sig1, sig2)
}

// --- MapEventToMetadata Tests ---

func TestMapEventToMetadata_PaymentSucceeded(t *testing.T) {
	result := &WebhookResult{Processed: true, EventType: EventPaymentSucceeded}
	payload := WebhookPayload{EventType: EventPaymentSucceeded}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, StatusActive, meta["payment_status"])
	assert.NotEmpty(t, meta["last_payment_at"])
}

func TestMapEventToMetadata_PaymentFailed(t *testing.T) {
	result := &WebhookResult{Processed: true, EventType: EventPaymentFailed}
	payload := WebhookPayload{EventType: EventPaymentFailed}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, StatusPastDue, meta["payment_status"])
}

func TestMapEventToMetadata_SubscriptionCreated_WithTier(t *testing.T) {
	result := &WebhookResult{Processed: true, EventType: EventSubscriptionCreated}
	payload := WebhookPayload{
		EventType: EventSubscriptionCreated,
		Data:      `{"billing_tier":"pro"}`,
	}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, TierPro, meta["billing_tier"])
	assert.Equal(t, StatusActive, meta["payment_status"])
}

func TestMapEventToMetadata_SubscriptionCreated_WithTierAltKey(t *testing.T) {
	result := &WebhookResult{Processed: true, EventType: EventSubscriptionCreated}
	payload := WebhookPayload{
		EventType: EventSubscriptionCreated,
		Data:      `{"tier":"enterprise"}`,
	}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, TierEnterprise, meta["billing_tier"])
}

func TestMapEventToMetadata_SubscriptionCreated_NoTier(t *testing.T) {
	result := &WebhookResult{Processed: true, EventType: EventSubscriptionCreated}
	payload := WebhookPayload{EventType: EventSubscriptionCreated, Data: `{}`}
	meta := MapEventToMetadata(result, payload)
	_, hasTier := meta["billing_tier"]
	assert.False(t, hasTier)
	assert.Equal(t, StatusActive, meta["payment_status"])
}

func TestMapEventToMetadata_SubscriptionCanceled(t *testing.T) {
	result := &WebhookResult{Processed: true}
	payload := WebhookPayload{EventType: EventSubscriptionCanceled}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, StatusCanceled, meta["payment_status"])
	assert.Equal(t, TierFree, meta["billing_tier"])
}

func TestMapEventToMetadata_InvoicePaid(t *testing.T) {
	result := &WebhookResult{Processed: true}
	payload := WebhookPayload{EventType: EventInvoicePaid}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, StatusActive, meta["payment_status"])
}

func TestMapEventToMetadata_InvoiceOverdue(t *testing.T) {
	result := &WebhookResult{Processed: true}
	payload := WebhookPayload{EventType: EventInvoiceOverdue}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, StatusPastDue, meta["payment_status"])
}

func TestMapEventToMetadata_CustomerCreated(t *testing.T) {
	result := &WebhookResult{Processed: true}
	payload := WebhookPayload{EventType: EventCustomerCreated, CustomerID: "cust-99"}
	meta := MapEventToMetadata(result, payload)
	assert.Equal(t, "cust-99", meta["billing_customer_id"])
}

func TestMapEventToMetadata_UnknownEvent(t *testing.T) {
	result := &WebhookResult{Processed: false}
	payload := WebhookPayload{EventType: "unknown.event"}
	meta := MapEventToMetadata(result, payload)
	assert.Empty(t, meta)
}

// --- extractTierFromData Tests ---

func TestExtractTierFromData_BillingTierKey(t *testing.T) {
	tier := extractTierFromData(`{"billing_tier":"enterprise"}`)
	assert.Equal(t, "enterprise", tier)
}

func TestExtractTierFromData_TierKey(t *testing.T) {
	tier := extractTierFromData(`{"tier":"starter"}`)
	assert.Equal(t, "starter", tier)
}

func TestExtractTierFromData_Empty(t *testing.T) {
	tier := extractTierFromData("")
	assert.Equal(t, "", tier)
}

func TestExtractTierFromData_InvalidJSON(t *testing.T) {
	tier := extractTierFromData("not-json")
	assert.Equal(t, "", tier)
}

func TestExtractTierFromData_NoTierField(t *testing.T) {
	tier := extractTierFromData(`{"other":"value"}`)
	assert.Equal(t, "", tier)
}

// --- Service Tests ---

func TestService_CreateCustomer_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Billing Org", "billing-org", "{}")
	mock := &mockProvider{}
	svc := NewService(mock, db)

	customer, err := svc.CreateCustomer(context.Background(), org.ID, CreateCustomerInput{
		Name:  "Customer Inc",
		Email: "cust@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "Customer Inc", customer.Name)
	assert.Equal(t, org.ID, customer.OrgID)

	// Verify org metadata was updated.
	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, "billing_customer_id")
}

func TestService_CreateCustomer_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)

	_, err := svc.CreateCustomer(context.Background(), "nonexistent", CreateCustomerInput{Name: "Test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org not found")
}

func TestService_CreateCustomer_BySlug(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Slug Org", "slug-org", "{}")
	svc := NewService(&mockProvider{}, db)

	customer, err := svc.CreateCustomer(context.Background(), org.Slug, CreateCustomerInput{
		Name:  "Customer",
		Email: "c@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, org.ID, customer.OrgID)
}

func TestService_CreateInvoice_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Invoice Org", "invoice-org", "{}")
	svc := NewService(&mockProvider{}, db)

	invoice, err := svc.CreateInvoice(context.Background(), org.ID, CreateInvoiceInput{
		CustomerID:  "cust-1",
		Amount:      49.99,
		Currency:    "USD",
		Description: "Pro plan",
	})
	require.NoError(t, err)
	assert.Equal(t, 49.99, invoice.Amount)
	assert.Equal(t, "USD", invoice.Currency)
}

func TestService_CreateInvoice_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)

	_, err := svc.CreateInvoice(context.Background(), "nonexistent", CreateInvoiceInput{
		CustomerID: "c1", Amount: 10, Currency: "USD",
	})
	assert.Error(t, err)
}

func TestService_GetBillingStatus_Default(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Status Org", "status-org", "{}")
	svc := NewService(&mockProvider{}, db)

	status, err := svc.GetBillingStatus(context.Background(), org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, status.OrgID)
	assert.Equal(t, TierFree, status.BillingTier)
	assert.Equal(t, StatusPending, status.PaymentStatus)
}

func TestService_GetBillingStatus_WithExistingMetadata(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Meta Org", "meta-org",
		`{"billing_tier":"pro","payment_status":"active","billing_customer_id":"cust-x"}`)
	svc := NewService(&mockProvider{}, db)

	status, err := svc.GetBillingStatus(context.Background(), org.ID)
	require.NoError(t, err)
	// Provider returns live status, overriding metadata.
	assert.Equal(t, TierPro, status.BillingTier)
	assert.Equal(t, StatusActive, status.PaymentStatus)
	assert.Equal(t, "cust-x", status.CustomerID)
}

func TestService_GetBillingStatus_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)

	_, err := svc.GetBillingStatus(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestService_ProcessWebhook_PaymentSucceeded(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "WH Org", "wh-org", "{}")
	svc := NewService(&mockProvider{}, db)

	result, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     org.ID,
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)

	// Verify org metadata updated.
	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"payment_status":"active"`)
}

func TestService_ProcessWebhook_PaymentFailed_MetadataUpdate(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "PF Org", "pf-org", `{"payment_status":"active"}`)

	mock := &mockProvider{
		handleWebhookFn: func(_ context.Context, payload WebhookPayload) (*WebhookResult, error) {
			return &WebhookResult{
				Processed: true,
				EventType: payload.EventType,
				OrgID:     payload.OrgID,
				Action:    "update_payment_status_past_due",
			}, nil
		},
	}
	svc := NewService(mock, db)

	result, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentFailed,
		OrgID:     org.ID,
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)

	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"payment_status":"past_due"`)
}

func TestService_ProcessWebhook_SubscriptionCreated_WithTier(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Sub Org", "sub-org", "{}")
	mock := &mockProvider{
		handleWebhookFn: func(_ context.Context, payload WebhookPayload) (*WebhookResult, error) {
			return &WebhookResult{
				Processed: true,
				EventType: payload.EventType,
				OrgID:     payload.OrgID,
				Action:    "update_billing_tier",
			}, nil
		},
	}
	svc := NewService(mock, db)

	_, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: EventSubscriptionCreated,
		OrgID:     org.ID,
		Data:      `{"billing_tier":"enterprise"}`,
	})
	require.NoError(t, err)

	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"billing_tier":"enterprise"`)
	assert.Contains(t, updated.Metadata, `"payment_status":"active"`)
}

func TestService_ProcessWebhook_UnknownEvent_NoUpdate(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "UK Org", "uk-org", `{"existing":"data"}`)
	mock := &mockProvider{
		handleWebhookFn: func(_ context.Context, payload WebhookPayload) (*WebhookResult, error) {
			return &WebhookResult{
				Processed: false,
				EventType: payload.EventType,
				OrgID:     payload.OrgID,
				Action:    "ignored",
			}, nil
		},
	}
	svc := NewService(mock, db)

	result, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: "unknown.event",
		OrgID:     org.ID,
	})
	require.NoError(t, err)
	assert.False(t, result.Processed)

	// Metadata should be unchanged.
	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	assert.Equal(t, `{"existing":"data"}`, updated.Metadata)
}

func TestService_ProcessWebhook_NoOrgID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)

	result, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     "", // no org ID
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
}

func TestService_ProcessWebhook_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)

	// Should not error — just skip the metadata update.
	result, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     "nonexistent-org",
	})
	require.NoError(t, err)
	assert.True(t, result.Processed)
}

func TestService_ProcessWebhook_DeepMerge(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Merge Org", "merge-org",
		`{"billing_tier":"starter","custom":"field"}`)
	svc := NewService(&mockProvider{}, db)

	_, err := svc.ProcessWebhook(context.Background(), WebhookPayload{
		EventType: EventPaymentSucceeded,
		OrgID:     org.ID,
	})
	require.NoError(t, err)

	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated.Metadata), &meta))
	assert.Equal(t, "active", meta["payment_status"])
	assert.Equal(t, "starter", meta["billing_tier"])
	assert.Equal(t, "field", meta["custom"])
}

// --- stringFromMeta Tests ---

func TestStringFromMeta_Exists(t *testing.T) {
	meta := map[string]any{"key": "value"}
	assert.Equal(t, "value", stringFromMeta(meta, "key", "default"))
}

func TestStringFromMeta_Missing(t *testing.T) {
	meta := map[string]any{}
	assert.Equal(t, "default", stringFromMeta(meta, "key", "default"))
}

func TestStringFromMeta_EmptyString(t *testing.T) {
	meta := map[string]any{"key": ""}
	assert.Equal(t, "default", stringFromMeta(meta, "key", "default"))
}

func TestStringFromMeta_NonString(t *testing.T) {
	meta := map[string]any{"key": 123}
	assert.Equal(t, "default", stringFromMeta(meta, "key", "default"))
}

// --- Constants Tests ---

func TestEventConstants(t *testing.T) {
	assert.Equal(t, "payment.succeeded", EventPaymentSucceeded)
	assert.Equal(t, "payment.failed", EventPaymentFailed)
	assert.Equal(t, "subscription.created", EventSubscriptionCreated)
	assert.Equal(t, "subscription.canceled", EventSubscriptionCanceled)
	assert.Equal(t, "invoice.paid", EventInvoicePaid)
	assert.Equal(t, "invoice.overdue", EventInvoiceOverdue)
	assert.Equal(t, "customer.created", EventCustomerCreated)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "active", StatusActive)
	assert.Equal(t, "past_due", StatusPastDue)
	assert.Equal(t, "canceled", StatusCanceled)
	assert.Equal(t, "pending", StatusPending)
}

func TestTierConstants(t *testing.T) {
	assert.Equal(t, "free", TierFree)
	assert.Equal(t, "starter", TierStarter)
	assert.Equal(t, "pro", TierPro)
	assert.Equal(t, "enterprise", TierEnterprise)
}
