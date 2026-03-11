// Package eventbus provides an in-process pub/sub event bus for domain events.
package eventbus

import (
	"sync"
	"time"
)

// Event represents a domain event published on the bus.
type Event struct {
	Type       string    `json:"type"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Payload    any       `json:"payload"`
	UserID     string    `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// Subscriber is a channel that receives events.
type Subscriber struct {
	ch     chan Event
	filter string // optional event type filter ("" = all events)
}

// Bus is an in-process pub/sub event bus.
type Bus struct {
	mu          sync.RWMutex
	subscribers []*Subscriber
	closed      bool
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{}
}

// Subscribe registers a subscriber for events matching the given filter.
// Pass an empty filter to receive all events.
// Returns a channel for reading events and an unsubscribe function.
func (b *Bus) Subscribe(filter string, bufSize int) (<-chan Event, func()) {
	if bufSize < 1 {
		bufSize = 64
	}
	sub := &Subscriber{
		ch:     make(chan Event, bufSize),
		filter: filter,
	}

	b.mu.Lock()
	b.subscribers = append(b.subscribers, sub)
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, s := range b.subscribers {
			if s == sub {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				close(s.ch)
				break
			}
		}
	}

	return sub.ch, unsub
}

// Publish sends an event to all matching subscribers.
// Non-blocking: drops events for slow subscribers.
func (b *Bus) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	for _, sub := range b.subscribers {
		if sub.filter != "" && sub.filter != event.Type {
			continue
		}
		select {
		case sub.ch <- event:
		default:
			// Subscriber is slow; drop event to avoid blocking.
		}
	}
}

// Close shuts down the bus and closes all subscriber channels.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true

	for _, sub := range b.subscribers {
		close(sub.ch)
	}
	b.subscribers = nil
}
