package voice

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
	"gorm.io/gorm"
)

// PhoneHandler provides HTTP handlers for phone number management.
// All endpoints require admin role in the org.
type PhoneHandler struct {
	provider LiveKitProvider
	db       *gorm.DB
}

// NewPhoneHandler creates a new PhoneHandler.
func NewPhoneHandler(provider LiveKitProvider, db *gorm.DB) *PhoneHandler {
	return &PhoneHandler{provider: provider, db: db}
}

// ListNumbers handles GET /v1/orgs/{org}/channels/voice/numbers.
// Returns all provisioned phone numbers for the org's LiveKit account.
func (h *PhoneHandler) ListNumbers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	numbers, err := h.provider.ListPhoneNumbers(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to list phone numbers")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"numbers": numbers})
}

// SearchNumbersRequest is the body for POST .../numbers/search.
type SearchNumbersRequest struct {
	AreaCode string `json:"area_code"`
}

// SearchNumbers handles POST /v1/orgs/{org}/channels/voice/numbers/search.
// Searches for available phone numbers by area code.
func (h *PhoneHandler) SearchNumbers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	var req SearchNumbersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if req.AreaCode == "" {
		apierrors.ValidationError(w, "area_code is required", nil)
		return
	}

	numbers, err := h.provider.SearchPhoneNumbers(r.Context(), req.AreaCode)
	if err != nil {
		apierrors.InternalError(w, "failed to search phone numbers")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"numbers": numbers})
}

// PurchaseNumberRequest is the body for POST .../numbers/purchase.
type PurchaseNumberRequest struct {
	NumberID string `json:"number_id"`
}

// PurchaseNumber handles POST /v1/orgs/{org}/channels/voice/numbers/purchase.
// Purchases and provisions a phone number.
func (h *PhoneHandler) PurchaseNumber(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	var req PurchaseNumberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if req.NumberID == "" {
		apierrors.ValidationError(w, "number_id is required", nil)
		return
	}

	number, err := h.provider.PurchasePhoneNumber(r.Context(), req.NumberID)
	if err != nil {
		apierrors.InternalError(w, "failed to purchase phone number")
		return
	}
	response.Created(w, number)
}

// requireAdmin checks that the authenticated user holds admin or owner role.
func (h *PhoneHandler) requireAdmin(w http.ResponseWriter, r *http.Request, orgID string) bool {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return false
	}

	var count int64
	h.db.Table("org_memberships").
		Where("org_id = ? AND user_id = ? AND role IN ?", orgID, uc.UserID, []string{"admin", "owner"}).
		Count(&count)

	if count == 0 {
		apierrors.Forbidden(w, "org admin or owner role required")
		return false
	}
	return true
}
