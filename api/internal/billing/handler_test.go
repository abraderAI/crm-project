package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// newTestRouter creates a Chi router with billing routes for handler testing.
func newTestRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/v1/webhooks/billing", handler.HandleWebhook)
	r.Route("/v1/orgs/{org}/billing", func(bl chi.Router) {
		bl.Get("/", handler.GetBillingStatus)
		bl.Post("/customers", handler.CreateCustomer)
		bl.Post("/invoices", handler.CreateInvoice)
	})
	return r
}

// --- Webhook Handler Tests ---

func TestHandler_Webhook_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "WH Handler", "wh-handler", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s"}`, org.ID)
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result WebhookResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.True(t, result.Processed)
	assert.Equal(t, "payment.succeeded", result.EventType)
}

func TestHandler_Webhook_WithValidSignature(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Sig Handler", "sig-handler", "{}")
	secret := "handler-test-secret"
	svc := NewService(NewFlexPointProvider(secret), db)
	handler := NewHandler(svc, secret)
	router := newTestRouter(handler)

	body := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s"}`, org.ID)
	sig := ComputeWebhookSignature([]byte(body), secret)

	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", sig)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Webhook_MissingSignature(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "required-secret")
	router := newTestRouter(handler)

	body := `{"event_type":"payment.succeeded","org_id":"org-1"}`
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
}

func TestHandler_Webhook_InvalidSignature(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "my-secret")
	router := newTestRouter(handler)

	body := `{"event_type":"payment.succeeded","org_id":"org-1"}`
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "invalid-sig")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Webhook_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Webhook_MissingEventType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"org_id":"org-1"}`
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(w.Body).Decode(&problem))
	assert.Equal(t, 400, problem.Status)
}

func TestHandler_Webhook_UnknownEvent(t *testing.T) {
	db := setupTestDB(t)
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
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"event_type":"custom.unknown","org_id":"org-1"}`
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result WebhookResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.False(t, result.Processed)
}

// --- GetBillingStatus Handler Tests ---

func TestHandler_GetBillingStatus_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Status Handler", "status-handler", `{"billing_tier":"pro","payment_status":"active"}`)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	req := httptest.NewRequest("GET", "/v1/orgs/"+org.ID+"/billing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var status BillingStatus
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.Equal(t, org.ID, status.OrgID)
}

func TestHandler_GetBillingStatus_BySlug(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Slug Status", "slug-status", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	req := httptest.NewRequest("GET", "/v1/orgs/"+org.Slug+"/billing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetBillingStatus_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	req := httptest.NewRequest("GET", "/v1/orgs/nonexistent/billing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
}

// --- CreateCustomer Handler Tests ---

func TestHandler_CreateCustomer_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Cust Handler", "cust-handler", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"name":"Customer Corp","email":"billing@example.com"}`
	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var customer Customer
	require.NoError(t, json.NewDecoder(w.Body).Decode(&customer))
	assert.Equal(t, "Customer Corp", customer.Name)
}

func TestHandler_CreateCustomer_MissingName(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "NoName Org", "noname-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateCustomer_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "BadJSON Org", "badjson-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/customers", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateCustomer_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"name":"Test"}`
	req := httptest.NewRequest("POST", "/v1/orgs/nonexistent/billing/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- CreateInvoice Handler Tests ---

func TestHandler_CreateInvoice_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Inv Handler", "inv-handler", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"customer_id":"cust-1","amount":99.99,"currency":"USD","description":"Monthly"}`
	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/invoices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var invoice Invoice
	require.NoError(t, json.NewDecoder(w.Body).Decode(&invoice))
	assert.Equal(t, 99.99, invoice.Amount)
}

func TestHandler_CreateInvoice_MissingCustomerID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "NoCust Org", "nocust-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"amount":10,"currency":"USD"}`
	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/invoices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateInvoice_ZeroAmount(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Zero Org", "zero-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"customer_id":"c1","amount":0,"currency":"USD"}`
	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/invoices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateInvoice_MissingCurrency(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "NoCur Org", "nocur-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"customer_id":"c1","amount":10}`
	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/invoices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateInvoice_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "BadInv Org", "badinv-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/billing/invoices", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateInvoice_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := `{"customer_id":"c1","amount":10,"currency":"USD"}`
	req := httptest.NewRequest("POST", "/v1/orgs/nonexistent/billing/invoices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Webhook + Metadata Integration Tests ---

func TestHandler_Webhook_UpdatesOrgMetadata(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Meta WH Org", "meta-wh-org", "{}")
	svc := NewService(&mockProvider{}, db)
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	// Send payment.succeeded webhook.
	body := fmt.Sprintf(`{"event_type":"payment.succeeded","org_id":"%s"}`, org.ID)
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify billing status reflects the update.
	req = httptest.NewRequest("GET", "/v1/orgs/"+org.ID+"/billing", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var status BillingStatus
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.Equal(t, org.ID, status.OrgID)
}

func TestHandler_Webhook_SubscriptionTierUpdate(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "Tier WH Org", "tier-wh-org", "{}")
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
	handler := NewHandler(svc, "")
	router := newTestRouter(handler)

	body := fmt.Sprintf(`{"event_type":"subscription.created","org_id":"%s","data":"{\"billing_tier\":\"enterprise\"}"}`, org.ID)
	req := httptest.NewRequest("POST", "/v1/webhooks/billing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
