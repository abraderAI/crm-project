// Package websocket provides WebSocket hub, client management, and real-time broadcasting.
package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Hub manages WebSocket channels and connected clients.
type Hub struct {
	mu       sync.RWMutex
	channels map[string]map[*Client]bool // channel name → set of clients
	clients  map[*Client]bool
	logger   *slog.Logger
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		channels: make(map[string]map[*Client]bool),
		clients:  make(map[*Client]bool),
		logger:   logger,
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

// Unregister removes a client from the hub and all its channels.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client)

	for ch, members := range h.channels {
		delete(members, client)
		if len(members) == 0 {
			delete(h.channels, ch)
		}
	}
}

// Subscribe adds a client to a named channel.
func (h *Hub) Subscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true
}

// Unsubscribe removes a client from a named channel.
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if members, ok := h.channels[channel]; ok {
		delete(members, client)
		if len(members) == 0 {
			delete(h.channels, channel)
		}
	}
}

// BroadcastMessage is the JSON envelope sent to WebSocket clients.
type BroadcastMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Payload any    `json:"payload"`
}

// Broadcast sends a message to all clients subscribed to the given channel.
func (h *Hub) Broadcast(channel string, msg BroadcastMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal broadcast message", slog.String("error", err.Error()))
		return
	}

	h.mu.RLock()
	members := h.channels[channel]
	// Copy to avoid holding lock during send.
	targets := make([]*Client, 0, len(members))
	for c := range members {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		c.Send(data)
	}
}

// ClientCount returns the total number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ChannelCount returns the total number of active channels.
func (h *Hub) ChannelCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels)
}

// ChannelClientCount returns the number of clients in a specific channel.
func (h *Hub) ChannelClientCount(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels[channel])
}
