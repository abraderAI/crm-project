package voice

import (
	"encoding/json"
	"io"
	"net/http"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// WebhookHandler handles inbound LiveKit webhook HTTP requests.
type WebhookHandler struct {
	service   *Service
	authToken string // Expected Authorization token for webhook validation.
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(service *Service, authToken string) *WebhookHandler {
	return &WebhookHandler{service: service, authToken: authToken}
}

// HandleWebhook handles POST /v1/webhooks/livekit.
// Validates the authorization header and processes the webhook event.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Validate auth token if configured.
	if h.authToken != "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != h.authToken && authHeader != "Bearer "+h.authToken {
			apierrors.Unauthorized(w, "invalid webhook authorization")
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit.
	if err != nil {
		apierrors.BadRequest(w, "failed to read request body")
		return
	}

	var evt WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		apierrors.BadRequest(w, "invalid webhook payload")
		return
	}

	if err := h.service.HandleWebhookEvent(r.Context(), evt); err != nil {
		apierrors.InternalError(w, "webhook processing failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
