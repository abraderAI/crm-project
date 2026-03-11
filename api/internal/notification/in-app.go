package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/abraderAI/crm-project/api/internal/models"
	wsHub "github.com/abraderAI/crm-project/api/internal/websocket"
)

// InAppProvider stores notifications in the database and pushes via WebSocket.
type InAppProvider struct {
	repo   *Repository
	hub    *wsHub.Hub
	logger *slog.Logger
}

// NewInAppProvider creates a new in-app notification provider.
func NewInAppProvider(repo *Repository, hub *wsHub.Hub, logger *slog.Logger) *InAppProvider {
	return &InAppProvider{
		repo:   repo,
		hub:    hub,
		logger: logger,
	}
}

// Name returns the provider name.
func (p *InAppProvider) Name() string {
	return ChannelInApp
}

// Send stores the notification in the database and pushes it via WebSocket.
func (p *InAppProvider) Send(ctx context.Context, notif *models.Notification) error {
	// Store in database.
	if err := p.repo.Create(ctx, notif); err != nil {
		return fmt.Errorf("storing notification: %w", err)
	}

	// Push via WebSocket to the user's personal channel.
	p.pushToWS(notif)

	return nil
}

// pushToWS sends a notification to the user's WebSocket channel.
func (p *InAppProvider) pushToWS(notif *models.Notification) {
	channel := fmt.Sprintf("user:%s", notif.UserID)
	data, err := json.Marshal(notif)
	if err != nil {
		p.logger.Error("failed to marshal notification for WS", slog.String("error", err.Error()))
		return
	}

	p.hub.Broadcast(channel, wsHub.BroadcastMessage{
		Type:    "notification",
		Channel: channel,
		Payload: json.RawMessage(data),
	})
}
