package voice

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// BridgeHandler provides internal API endpoints for the agent sidecar.
// Authenticated via X-Internal-Key header (not Clerk JWT).
type BridgeHandler struct {
	service     *Service
	internalKey string
}

// NewBridgeHandler creates a new BridgeHandler.
func NewBridgeHandler(service *Service, internalKey string) *BridgeHandler {
	return &BridgeHandler{service: service, internalKey: internalKey}
}

// LookupContact handles GET /v1/internal/contacts/lookup?email=&phone=.
// Returns threads associated with the given email or phone number.
func (h *BridgeHandler) LookupContact(w http.ResponseWriter, r *http.Request) {
	if !h.checkInternalKey(w, r) {
		return
	}

	email := r.URL.Query().Get("email")
	phone := r.URL.Query().Get("phone")
	if email == "" && phone == "" {
		apierrors.BadRequest(w, "email or phone query parameter required")
		return
	}

	results, err := h.service.LookupContact(r.Context(), email, phone)
	if err != nil {
		apierrors.InternalError(w, "contact lookup failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"contacts": results})
}

// GetThreadSummary handles GET /v1/internal/threads/{id}/summary.
// Returns a brief summary of the thread for the agent sidecar.
func (h *BridgeHandler) GetThreadSummary(w http.ResponseWriter, r *http.Request) {
	if !h.checkInternalKey(w, r) {
		return
	}

	threadID := chi.URLParam(r, "id")
	if threadID == "" {
		apierrors.BadRequest(w, "thread id is required")
		return
	}

	summary, err := h.service.GetThreadSummary(r.Context(), threadID)
	if err != nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.JSON(w, http.StatusOK, summary)
}

// checkInternalKey validates the X-Internal-Key header.
func (h *BridgeHandler) checkInternalKey(w http.ResponseWriter, r *http.Request) bool {
	key := r.Header.Get("X-Internal-Key")
	if h.internalKey == "" {
		// No key configured; allow all requests (dev mode).
		return true
	}
	if key != h.internalKey {
		apierrors.Unauthorized(w, "invalid internal API key")
		return false
	}
	return true
}
