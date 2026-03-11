package websocket

import (
	"log/slog"
	"net/http"

	ws "github.com/coder/websocket"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// Handler provides the WebSocket upgrade HTTP handler.
type Handler struct {
	hub       *Hub
	validator *auth.JWTValidator
	logger    *slog.Logger
	origins   []string
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub, validator *auth.JWTValidator, logger *slog.Logger, origins []string) *Handler {
	return &Handler{
		hub:       hub,
		validator: validator,
		logger:    logger,
		origins:   origins,
	}
}

// Upgrade handles the WebSocket upgrade request at GET /v1/ws.
// Authentication is via query param ?token= (JWT).
func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	// Authenticate via query parameter.
	token := r.URL.Query().Get("token")
	if token == "" {
		apierrors.Unauthorized(w, "token query parameter is required for WebSocket auth")
		return
	}

	claims, err := h.validator.Validate(token)
	if err != nil {
		apierrors.Unauthorized(w, "invalid or expired token")
		return
	}

	// Accept the WebSocket connection.
	opts := &ws.AcceptOptions{
		InsecureSkipVerify: len(h.origins) == 0,
		OriginPatterns:     h.origins,
	}

	conn, err := ws.Accept(w, r, opts)
	if err != nil {
		h.logger.Error("websocket accept failed", slog.String("error", err.Error()))
		return
	}

	client := NewClient(conn, h.hub, claims.Subject, h.logger)
	client.Run(r.Context())
}
