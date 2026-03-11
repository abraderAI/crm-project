package voice

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// CreateCallRequest is the JSON body for POST /v1/orgs/{org}/calls.
type CreateCallRequest struct {
	CallerID  string               `json:"caller_id"`
	Direction models.CallDirection `json:"direction"`
	Duration  int                  `json:"duration"`
	Status    models.CallStatus    `json:"status"`
	Metadata  string               `json:"metadata,omitempty"`
}

// Handler provides HTTP handlers for voice call operations.
type Handler struct {
	provider VoiceProvider
}

// NewHandler creates a new voice Handler.
func NewHandler(provider VoiceProvider) *Handler {
	return &Handler{provider: provider}
}

// LogCall handles POST /v1/orgs/{org}/calls.
func (h *Handler) LogCall(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if orgID == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var req CreateCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if req.CallerID == "" {
		apierrors.ValidationError(w, "caller_id is required", nil)
		return
	}

	callLog, err := h.provider.LogCall(r.Context(), LogCallInput{
		OrgID:     orgID,
		CallerID:  req.CallerID,
		Direction: req.Direction,
		Duration:  req.Duration,
		Status:    req.Status,
		Metadata:  req.Metadata,
	})
	if err != nil {
		apierrors.InternalError(w, "failed to log call")
		return
	}

	response.Created(w, callLog)
}

// GetTranscript handles GET /v1/orgs/{org}/calls/{call}.
func (h *Handler) GetTranscript(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "call")
	if callID == "" {
		apierrors.BadRequest(w, "call identifier is required")
		return
	}

	transcript, err := h.provider.GetTranscript(r.Context(), callID)
	if err != nil {
		apierrors.NotFound(w, "call not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"call_id":    callID,
		"transcript": transcript,
	})
}

// Escalate handles POST /v1/orgs/{org}/calls/{call}/escalate.
func (h *Handler) Escalate(w http.ResponseWriter, r *http.Request) {
	callID := chi.URLParam(r, "call")
	if callID == "" {
		apierrors.BadRequest(w, "call identifier is required")
		return
	}

	result, err := h.provider.Escalate(r.Context(), callID)
	if err != nil {
		apierrors.NotFound(w, "call not found")
		return
	}

	response.JSON(w, http.StatusOK, result)
}
