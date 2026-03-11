package websocket

import (
	"fmt"
	"log/slog"

	"github.com/abraderAI/crm-project/api/internal/eventbus"
)

// Broadcaster subscribes to the event bus and broadcasts events to WebSocket channels.
type Broadcaster struct {
	hub    *Hub
	bus    *eventbus.Bus
	unsub  func()
	logger *slog.Logger
	done   chan struct{}
}

// NewBroadcaster creates a new event broadcaster.
func NewBroadcaster(hub *Hub, bus *eventbus.Bus, logger *slog.Logger) *Broadcaster {
	return &Broadcaster{
		hub:    hub,
		bus:    bus,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start begins listening for events and broadcasting to WebSocket channels.
func (b *Broadcaster) Start() {
	events, unsub := b.bus.Subscribe("", 256)
	b.unsub = unsub

	go func() {
		defer close(b.done)
		for event := range events {
			b.handleEvent(event)
		}
	}()
}

// Stop stops the broadcaster.
func (b *Broadcaster) Stop() {
	if b.unsub != nil {
		b.unsub()
	}
	<-b.done
}

// handleEvent routes an event to the appropriate WebSocket channel(s).
func (b *Broadcaster) handleEvent(event eventbus.Event) {
	switch event.Type {
	case "message.created":
		// Broadcast to thread and board channels.
		if threadID, ok := extractPayloadField(event.Payload, "thread_id"); ok {
			b.hub.Broadcast(fmt.Sprintf("thread:%s", threadID), BroadcastMessage{
				Type:    event.Type,
				Channel: fmt.Sprintf("thread:%s", threadID),
				Payload: event.Payload,
			})
		}
		if boardID, ok := extractPayloadField(event.Payload, "board_id"); ok {
			b.hub.Broadcast(fmt.Sprintf("board:%s", boardID), BroadcastMessage{
				Type:    event.Type,
				Channel: fmt.Sprintf("board:%s", boardID),
				Payload: event.Payload,
			})
		}
	case "thread.updated", "thread.created":
		if boardID, ok := extractPayloadField(event.Payload, "board_id"); ok {
			b.hub.Broadcast(fmt.Sprintf("board:%s", boardID), BroadcastMessage{
				Type:    event.Type,
				Channel: fmt.Sprintf("board:%s", boardID),
				Payload: event.Payload,
			})
		}
		// Also broadcast to the thread channel itself.
		b.hub.Broadcast(fmt.Sprintf("thread:%s", event.EntityID), BroadcastMessage{
			Type:    event.Type,
			Channel: fmt.Sprintf("thread:%s", event.EntityID),
			Payload: event.Payload,
		})
	case "typing":
		// Typing is handled directly by the client; no bus routing needed.
		// But if published on the bus, broadcast to the specified channel.
		if channel, ok := extractPayloadField(event.Payload, "channel"); ok {
			b.hub.Broadcast(channel, BroadcastMessage{
				Type:    "typing",
				Channel: channel,
				Payload: event.Payload,
			})
		}
	default:
		// For other events, broadcast to entity-specific channel.
		if event.EntityType != "" && event.EntityID != "" {
			channel := fmt.Sprintf("%s:%s", event.EntityType, event.EntityID)
			b.hub.Broadcast(channel, BroadcastMessage{
				Type:    event.Type,
				Channel: channel,
				Payload: event.Payload,
			})
		}
	}
}

// extractPayloadField extracts a string field from an event payload.
func extractPayloadField(payload any, field string) (string, bool) {
	if m, ok := payload.(map[string]any); ok {
		if v, exists := m[field]; exists {
			if s, ok := v.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}
