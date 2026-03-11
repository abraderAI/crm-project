package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/billing"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// --- Phase 8 Live API Tests ---

// TestLive_Phase8_WebhookPaymentSucceeded simulates a FlexPoint payment.succeeded
// webhook POST, then verifies org metadata is updated with payment_status.
func TestLive_Phase8_WebhookPaymentSucceeded(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create an org.
	org := &models.Org{Name: "Billing Org", Slug: "billing-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// POST webhook with payment.succeeded event.
	body := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s","customer_id":"cust-1"}`, org.ID)
	req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var result billing.WebhookResult
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.True(t, result.Processed)
	assert.Equal(t, "payment.succeeded", result.EventType)

	// Verify org metadata updated.
	var updated models.Org
	require.NoError(t, env.DB.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"payment_status":"active"`)
}

// TestLive_Phase8_WebhookUpdatesBillingTier simulates a subscription.created
// webhook and verifies the org's billing_tier is updated in metadata.
func TestLive_Phase8_WebhookUpdatesBillingTier(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Tier Org", Slug: "tier-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	body := fmt.Sprintf(`{"event_type":"subscription.created","org_id":"%s","data":"{\"billing_tier\":\"pro\"}"}`, org.ID)
	req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify org metadata has billing_tier.
	var updated models.Org
	require.NoError(t, env.DB.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"billing_tier":"pro"`)
	assert.Contains(t, updated.Metadata, `"payment_status":"active"`)
}

// TestLive_Phase8_BillingStatusViaOrgGet verifies billing metadata is visible
// in GET /v1/orgs/{org} response after a webhook updates it.
func TestLive_Phase8_BillingStatusViaOrgGet(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Get Org", Slug: "get-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// Send payment.succeeded webhook.
	whBody := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s"}`, org.ID)
	whReq, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(whBody))
	require.NoError(t, err)
	whReq.Header.Set("Content-Type", "application/json")
	whResp, err := http.DefaultClient.Do(whReq)
	require.NoError(t, err)
	_ = whResp.Body.Close()
	assert.Equal(t, http.StatusOK, whResp.StatusCode)

	// GET the org and verify billing metadata is visible.
	getResp := authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+org.ID, "")
	defer func() { _ = getResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	orgData := decodeJSON(t, getResp)
	metadata := orgData["metadata"].(string)
	assert.Contains(t, metadata, "payment_status")
	assert.Contains(t, metadata, "active")
}

// TestLive_Phase8_BillingStatusEndpoint verifies GET /v1/orgs/{org}/billing
// returns billing status for an org with billing metadata.
func TestLive_Phase8_BillingStatusEndpoint(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{
		Name:     "Status Org",
		Slug:     "status-org",
		Metadata: `{"billing_tier":"pro","payment_status":"active","billing_customer_id":"cust-x"}`,
	}
	require.NoError(t, env.DB.Create(org).Error)

	resp := authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+org.ID+"/billing", "")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var status billing.BillingStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
	assert.Equal(t, org.ID, status.OrgID)
	assert.Equal(t, "cust-x", status.CustomerID)
}

// TestLive_Phase8_BillingStatusDefault verifies default billing status for an org
// without any billing metadata.
func TestLive_Phase8_BillingStatusDefault(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Default Org", Slug: "default-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	resp := authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+org.ID+"/billing", "")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var status billing.BillingStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
	assert.Equal(t, org.ID, status.OrgID)
	assert.Equal(t, "free", status.BillingTier)
	assert.Equal(t, "pending", status.PaymentStatus)
}

// TestLive_Phase8_BillingStatusNotFound verifies 404 for nonexistent org.
func TestLive_Phase8_BillingStatusNotFound(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "GET", env.BaseURL+"/v1/orgs/nonexistent-org-id/billing", "")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase8_CreateCustomer verifies POST /v1/orgs/{org}/billing/customers.
func TestLive_Phase8_CreateCustomer(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Cust Org", Slug: "cust-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	body := `{"name":"Acme Corp","email":"billing@acme.com"}`
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+org.ID+"/billing/customers", body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var customer billing.Customer
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&customer))
	assert.Equal(t, "Acme Corp", customer.Name)
	assert.Equal(t, org.ID, customer.OrgID)
	assert.Contains(t, customer.ExternalID, "fp_cust_")

	// Verify org metadata was updated with customer ID.
	var updated models.Org
	require.NoError(t, env.DB.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, "billing_customer_id")
}

// TestLive_Phase8_CreateCustomerValidation verifies validation errors.
func TestLive_Phase8_CreateCustomerValidation(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Val Org", Slug: "val-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// Missing name.
	body := `{"email":"test@example.com"}`
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+org.ID+"/billing/customers", body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase8_CreateInvoice verifies POST /v1/orgs/{org}/billing/invoices.
func TestLive_Phase8_CreateInvoice(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Inv Org", Slug: "inv-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	body := `{"customer_id":"cust-1","amount":99.99,"currency":"USD","description":"Monthly Pro"}`
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+org.ID+"/billing/invoices", body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var invoice billing.Invoice
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&invoice))
	assert.Equal(t, 99.99, invoice.Amount)
	assert.Equal(t, "USD", invoice.Currency)
	assert.Equal(t, "pending", invoice.Status)
}

// TestLive_Phase8_CreateInvoiceValidation verifies invoice validation errors.
func TestLive_Phase8_CreateInvoiceValidation(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "InvVal Org", Slug: "invval-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	tests := []struct {
		name string
		body string
	}{
		{"missing customer_id", `{"amount":10,"currency":"USD"}`},
		{"zero amount", `{"customer_id":"c1","amount":0,"currency":"USD"}`},
		{"missing currency", `{"customer_id":"c1","amount":10}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+org.ID+"/billing/invoices", tt.body)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
		})
	}
}

// TestLive_Phase8_WebhookInvalidPayload verifies invalid webhook payloads
// return proper RFC 7807 error responses.
func TestLive_Phase8_WebhookInvalidPayload(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"invalid json", "not-json", http.StatusBadRequest},
		{"missing event_type", `{"org_id":"org-1"}`, http.StatusBadRequest},
		{"empty body", `{}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

			var problem apierrors.ProblemDetail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
			assert.Equal(t, tt.status, problem.Status)
		})
	}
}

// TestLive_Phase8_WebhookPaymentFailed simulates payment failure and verifies
// metadata is updated to past_due.
func TestLive_Phase8_WebhookPaymentFailed(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Fail Org", Slug: "fail-org", Metadata: `{"payment_status":"active"}`}
	require.NoError(t, env.DB.Create(org).Error)

	body := fmt.Sprintf(`{"event_type":"payment.failed","org_id":"%s"}`, org.ID)
	req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updated models.Org
	require.NoError(t, env.DB.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"payment_status":"past_due"`)
}

// TestLive_Phase8_WebhookSubscriptionCanceled verifies subscription cancellation
// resets billing tier to free and sets status to canceled.
func TestLive_Phase8_WebhookSubscriptionCanceled(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{
		Name:     "Cancel Org",
		Slug:     "cancel-org",
		Metadata: `{"billing_tier":"enterprise","payment_status":"active"}`,
	}
	require.NoError(t, env.DB.Create(org).Error)

	body := fmt.Sprintf(`{"event_type":"subscription.canceled","org_id":"%s"}`, org.ID)
	req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updated models.Org
	require.NoError(t, env.DB.First(&updated, "id = ?", org.ID).Error)
	assert.Contains(t, updated.Metadata, `"billing_tier":"free"`)
	assert.Contains(t, updated.Metadata, `"payment_status":"canceled"`)
}

// TestLive_Phase8_FullBillingLifecycle exercises the complete billing flow:
// create customer → create invoice → webhook payment → verify status.
func TestLive_Phase8_FullBillingLifecycle(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// 1. Create org.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Lifecycle Org"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	orgData := decodeJSON(t, resp)
	orgID := orgData["id"].(string)

	// 2. Create billing customer.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/billing/customers",
		`{"name":"Lifecycle Customer","email":"lifecycle@test.com"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	custData := decodeJSON(t, resp)
	assert.NotEmpty(t, custData["external_id"])

	// 3. Create invoice.
	custID := custData["id"].(string)
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/billing/invoices",
		fmt.Sprintf(`{"customer_id":"%s","amount":199.99,"currency":"USD","description":"Enterprise Plan"}`, custID))
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// 4. Simulate payment succeeded webhook.
	whBody := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s","customer_id":"%s"}`, orgID, custID)
	whReq, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(whBody))
	require.NoError(t, err)
	whReq.Header.Set("Content-Type", "application/json")
	whResp, err := http.DefaultClient.Do(whReq)
	require.NoError(t, err)
	defer func() { _ = whResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, whResp.StatusCode)

	// 5. Verify billing status via GET.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/billing", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var status billing.BillingStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
	assert.Equal(t, orgID, status.OrgID)

	// 6. Verify org metadata has billing info via GET /v1/orgs/{org}.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	orgResult := decodeJSON(t, resp)
	metadata := orgResult["metadata"].(string)
	assert.Contains(t, metadata, "payment_status")
	assert.Contains(t, metadata, "billing_customer_id")
}

// TestLive_Phase8_WebhookResponseHeaders verifies webhook response includes
// proper headers (X-Request-ID, Content-Type).
func TestLive_Phase8_WebhookResponseHeaders(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Header Org", Slug: "header-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	body := fmt.Sprintf(`{"event_type":"customer.created","org_id":"%s","customer_id":"c1"}`, org.ID)
	req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

// TestLive_Phase8_BillingAuthRequired verifies billing status endpoint
// requires authentication.
func TestLive_Phase8_BillingAuthRequired(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Auth Billing", Slug: "auth-billing", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// GET without auth should fail.
	resp, err := http.Get(env.BaseURL + "/v1/orgs/" + org.ID + "/billing")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase8_WebhookIsPublic verifies webhook endpoint does NOT
// require JWT authentication (it's HMAC-verified separately).
func TestLive_Phase8_WebhookIsPublic(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Public WH", Slug: "public-wh", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// POST without auth should work (webhook is public, HMAC-verified).
	body := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s"}`, org.ID)
	req, err := http.NewRequest("POST", env.BaseURL+"/v1/webhooks/billing", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should succeed (no webhook secret configured in test server).
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
