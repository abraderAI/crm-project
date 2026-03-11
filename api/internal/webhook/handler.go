package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for webhook operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new webhook handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /v1/orgs/{org}/webhooks.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	sub, err := h.service.Create(r.Context(), orgID, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, sub)
}

// List handles GET /v1/orgs/{org}/webhooks.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params := pagination.Parse(r)

	subs, pageInfo, err := h.service.List(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list webhooks")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     subs,
		PageInfo: pageInfo,
	})
}

// Get handles GET /v1/orgs/{org}/webhooks/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sub, err := h.service.Get(r.Context(), id)
	if err != nil {
		apierrors.InternalError(w, "failed to get webhook")
		return
	}
	if sub == nil {
		apierrors.NotFound(w, "webhook not found")
		return
	}
	response.JSON(w, http.StatusOK, sub)
}

// Delete handles DELETE /v1/orgs/{org}/webhooks/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "webhook not found")
			return
		}
		apierrors.InternalError(w, "failed to delete webhook")
		return
	}
	response.NoContent(w)
}

// ListDeliveries handles GET /v1/orgs/{org}/webhooks/{id}/deliveries.
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	subID := chi.URLParam(r, "id")
	params := pagination.Parse(r)

	deliveries, pageInfo, err := h.service.ListDeliveries(r.Context(), subID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list deliveries")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     deliveries,
		PageInfo: pageInfo,
	})
}

// Replay handles POST /v1/orgs/{org}/webhooks/{id}/deliveries/{deliveryID}/replay.
func (h *Handler) Replay(w http.ResponseWriter, r *http.Request) {
	deliveryID := chi.URLParam(r, "deliveryID")
	delivery, err := h.service.Replay(r.Context(), deliveryID)
	if err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "delivery not found")
			return
		}
		apierrors.InternalError(w, "failed to replay delivery")
		return
	}
	response.JSON(w, http.StatusOK, delivery)
}
