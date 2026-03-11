package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	ws "github.com/coder/websocket"
)

const (
	// PingInterval is how often the server pings the client.
	PingInterval = 30 * time.Second
	// PongTimeout is the deadline for receiving a pong.
	PongTimeout = 60 * time.Second
	// WriteTimeout is the deadline for writing a message.
	WriteTimeout = 10 * time.Second
	// SendBufferSize is the outgoing message buffer.
	SendBufferSize = 256
	// MaxMessageSize is the maximum incoming message size (64KB).
	MaxMessageSize = 64 * 1024
)

// ClientMessage is the JSON envelope for messages received from clients.
type ClientMessage struct {
	Action  string `json:"action"`  // "subscribe", "unsubscribe", "typing"
	Channel string `json:"channel"` // e.g. "board:xxx", "thread:xxx"
}

// Client represents a single WebSocket connection.
type Client struct {
	conn   *ws.Conn
	hub    *Hub
	userID string
	send   chan []byte
	logger *slog.Logger
	done   chan struct{}
}

// NewClient creates a new WebSocket client.
func NewClient(conn *ws.Conn, hub *Hub, userID string, logger *slog.Logger) *Client {
	return &Client{
		conn:   conn,
		hub:    hub,
		userID: userID,
		send:   make(chan []byte, SendBufferSize),
		logger: logger,
		done:   make(chan struct{}),
	}
}

// UserID returns the authenticated user ID for this client.
func (c *Client) UserID() string {
	return c.userID
}

// Send queues a message for sending to the client. Non-blocking.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
		// Buffer full; skip message.
	}
}

// Run starts the read and write pumps. Blocks until the connection closes.
func (c *Client) Run(ctx context.Context) {
	c.hub.Register(c)
	defer func() {
		c.hub.Unregister(c)
		close(c.done)
		_ = c.conn.Close(ws.StatusNormalClosure, "connection closed")
	}()

	// Set read limit.
	c.conn.SetReadLimit(MaxMessageSize)

	go c.writePump(ctx)
	c.readPump(ctx)
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump(ctx context.Context) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			// Connection closed or error.
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Debug("invalid client message", slog.String("error", err.Error()))
			continue
		}

		c.handleMessage(msg)
	}
}

// writePump writes messages to the WebSocket connection and handles ping/pong.
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, WriteTimeout)
			err := c.conn.Write(writeCtx, ws.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, PongTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		case <-c.done:
			return
		}
	}
}

// handleMessage processes incoming client messages (subscribe, unsubscribe, typing).
func (c *Client) handleMessage(msg ClientMessage) {
	switch msg.Action {
	case "subscribe":
		if msg.Channel == "" {
			return
		}
		c.hub.Subscribe(c, msg.Channel)
		c.sendAck("subscribed", msg.Channel)
	case "unsubscribe":
		if msg.Channel == "" {
			return
		}
		c.hub.Unsubscribe(c, msg.Channel)
		c.sendAck("unsubscribed", msg.Channel)
	case "typing":
		if msg.Channel == "" {
			return
		}
		// Broadcast typing indicator to the channel.
		c.hub.Broadcast(msg.Channel, BroadcastMessage{
			Type:    "typing",
			Channel: msg.Channel,
			Payload: map[string]string{"user_id": c.userID},
		})
	default:
		c.logger.Debug("unknown client action", slog.String("action", msg.Action))
	}
}

// sendAck sends an acknowledgment message to the client.
func (c *Client) sendAck(action, channel string) {
	ack := BroadcastMessage{
		Type:    action,
		Channel: channel,
	}
	data, err := json.Marshal(ack)
	if err != nil {
		return
	}
	c.Send(data)
}
