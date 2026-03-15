package chat

import (
	"encoding/json"
	"net/http"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for the chat widget API.
type Handler struct {
	service   *Service
	jwtSecret string
}

// NewHandler creates a new chat Handler.
func NewHandler(service *Service, jwtSecret string) *Handler {
	return &Handler{service: service, jwtSecret: jwtSecret}
}

// CreateSession handles POST /v1/chat/session.
// Accepts an embed key and fingerprint hash, returns a session JWT.
// This endpoint is public (no auth middleware).
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var input CreateSessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	output, err := h.service.CreateSession(r.Context(), input)
	if err != nil {
		if err.Error() == "invalid embed key" {
			apierrors.Unauthorized(w, "invalid embed key")
			return
		}
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, output)
}

// SendMessage handles POST /v1/chat/message.
// Accepts a chat session JWT in the Authorization header and a message body.
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Extract and validate chat session JWT.
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		apierrors.Unauthorized(w, "chat session token is required")
		return
	}

	claims, err := ValidateSessionToken(h.jwtSecret, token)
	if err != nil {
		apierrors.Unauthorized(w, "invalid or expired chat session token")
		return
	}

	var input struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	resp, err := h.service.HandleChatMessage(r.Context(), claims, input.Message)
	if err != nil {
		apierrors.InternalError(w, "failed to process chat message")
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
