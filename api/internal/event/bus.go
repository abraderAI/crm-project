// Package event provides an in-process pub/sub event bus for domain events.
package event

import (
	"sync"
)

// Type identifies the kind of event.
type Type string

const (
	// Entity lifecycle events.
	OrgCreated     Type = "org.created"
	OrgUpdated     Type = "org.updated"
	OrgDeleted     Type = "org.deleted"
	SpaceCreated   Type = "space.created"
	SpaceUpdated   Type = "space.updated"
	SpaceDeleted   Type = "space.deleted"
	BoardCreated   Type = "board.created"
	BoardUpdated   Type = "board.updated"
	BoardDeleted   Type = "board.deleted"
	ThreadCreated  Type = "thread.created"
	ThreadUpdated  Type = "thread.updated"
	ThreadDeleted  Type = "thread.deleted"
	MessageCreated Type = "message.created"
	MessageUpdated Type = "message.updated"
	MessageDeleted Type = "message.deleted"
	UploadCreated  Type = "upload.created"
	UploadDeleted  Type = "upload.deleted"
)

// Event represents a domain event published to the bus.
type Event struct {
	Type       Type   `json:"type"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	OrgID      string `json:"org_id"`
	UserID     string `json:"user_id"`
	Payload    string `json:"payload"` // JSON-encoded entity state.
}

// Handler is a callback invoked when an event is published.
type Handler func(Event)

// Bus is a simple in-process event bus supporting typed publish/subscribe.
type Bus struct {
	mu       sync.RWMutex
	handlers map[Type][]Handler
	allSubs  []Handler // wildcard subscribers
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[Type][]Handler),
	}
}

// Subscribe registers a handler for a specific event type.
func (b *Bus) Subscribe(eventType Type, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// SubscribeAll registers a handler that receives all events.
func (b *Bus) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.allSubs = append(b.allSubs, handler)
}

// Publish sends an event to all matching subscribers asynchronously.
func (b *Bus) Publish(evt Event) {
	b.mu.RLock()
	specific := make([]Handler, len(b.handlers[evt.Type]))
	copy(specific, b.handlers[evt.Type])
	all := make([]Handler, len(b.allSubs))
	copy(all, b.allSubs)
	b.mu.RUnlock()

	for _, h := range specific {
		go h(evt)
	}
	for _, h := range all {
		go h(evt)
	}
}
