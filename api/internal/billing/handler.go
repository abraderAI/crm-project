package billing

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for billing operations.
type Handler struct {
	service       *Service
	webhookSecret string
}

// NewHandler creates a new billing handler.
func NewHandler(service *Service, webhookSecret string) *Handler {
	return &Handler{service: service, webhookSecret: webhookSecret}
}

// HandleWebhook handles POST /v1/webhooks/billing.
// This endpoint is public (no JWT required) but verified via HMAC signature.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		apierrors.BadRequest(w, "failed to read request body")
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Verify HMAC signature if secret is configured.
	signature := r.Header.Get("X-Webhook-Signature")
	if h.webhookSecret != "" {
		if signature == "" {
			apierrors.Unauthorized(w, "missing webhook signature")
			return
		}
		if !VerifyWebhookSignature(body, signature, h.webhookSecret) {
			apierrors.Unauthorized(w, "invalid webhook signature")
			return
		}
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		apierrors.BadRequest(w, "invalid webhook payload")
		return
	}
	payload.Signature = signature
	payload.RawBody = body

	if payload.EventType == "" {
		apierrors.ValidationError(w, "event_type is required", nil)
		return
	}

	result, err := h.service.ProcessWebhook(r.Context(), payload)
	if err != nil {
		apierrors.InternalError(w, "failed to process webhook")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetBillingStatus handles GET /v1/orgs/{org}/billing.
func (h *Handler) GetBillingStatus(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	status, err := h.service.GetBillingStatus(r.Context(), orgIDOrSlug)
	if err != nil {
		if err.Error() == "org not found" {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to get billing status")
		return
	}

	response.JSON(w, http.StatusOK, status)
}

// CreateCustomer handles POST /v1/orgs/{org}/billing/customers.
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var input CreateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if input.Name == "" {
		apierrors.ValidationError(w, "name is required", nil)
		return
	}

	customer, err := h.service.CreateCustomer(r.Context(), orgIDOrSlug, input)
	if err != nil {
		if err.Error() == "org not found" {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to create customer")
		return
	}

	response.Created(w, customer)
}

// CreateInvoice handles POST /v1/orgs/{org}/billing/invoices.
func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var input CreateInvoiceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if input.CustomerID == "" {
		apierrors.ValidationError(w, "customer_id is required", nil)
		return
	}
	if input.Amount <= 0 {
		apierrors.ValidationError(w, "amount must be positive", nil)
		return
	}
	if input.Currency == "" {
		apierrors.ValidationError(w, "currency is required", nil)
		return
	}

	invoice, err := h.service.CreateInvoice(r.Context(), orgIDOrSlug, input)
	if err != nil {
		if err.Error() == "org not found" {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to create invoice")
		return
	}

	response.Created(w, invoice)
}
