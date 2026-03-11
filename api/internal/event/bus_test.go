package event

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	require.NotNil(t, bus)
	assert.NotNil(t, bus.handlers)
}

func TestBus_Subscribe_And_Publish(t *testing.T) {
	bus := NewBus()
	var received Event
	var mu sync.Mutex
	done := make(chan struct{}, 1)

	bus.Subscribe(OrgCreated, func(evt Event) {
		mu.Lock()
		received = evt
		mu.Unlock()
		done <- struct{}{}
	})

	evt := Event{
		Type:       OrgCreated,
		EntityType: "org",
		EntityID:   "org-123",
		OrgID:      "org-123",
		UserID:     "user-1",
		Payload:    `{"name":"test"}`,
	}
	bus.Publish(evt)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	assert.Equal(t, OrgCreated, received.Type)
	assert.Equal(t, "org-123", received.EntityID)
	mu.Unlock()
}

func TestBus_SubscribeAll(t *testing.T) {
	bus := NewBus()
	var received []Event
	var mu sync.Mutex
	done := make(chan struct{}, 2)

	bus.SubscribeAll(func(evt Event) {
		mu.Lock()
		received = append(received, evt)
		mu.Unlock()
		done <- struct{}{}
	})

	bus.Publish(Event{Type: OrgCreated, EntityID: "1"})
	bus.Publish(Event{Type: ThreadCreated, EntityID: "2"})

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}

	mu.Lock()
	assert.Len(t, received, 2)
	mu.Unlock()
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := NewBus()
	// Should not panic.
	bus.Publish(Event{Type: OrgDeleted, EntityID: "x"})
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()
	var count int
	var mu sync.Mutex
	done := make(chan struct{}, 3)

	for i := 0; i < 3; i++ {
		bus.Subscribe(MessageCreated, func(_ Event) {
			mu.Lock()
			count++
			mu.Unlock()
			done <- struct{}{}
		})
	}

	bus.Publish(Event{Type: MessageCreated, EntityID: "m1"})

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}

	mu.Lock()
	assert.Equal(t, 3, count)
	mu.Unlock()
}

func TestBus_SpecificAndAllSubscribers(t *testing.T) {
	bus := NewBus()
	var specificCount, allCount int
	var mu sync.Mutex
	done := make(chan struct{}, 2)

	bus.Subscribe(SpaceCreated, func(_ Event) {
		mu.Lock()
		specificCount++
		mu.Unlock()
		done <- struct{}{}
	})

	bus.SubscribeAll(func(_ Event) {
		mu.Lock()
		allCount++
		mu.Unlock()
		done <- struct{}{}
	})

	bus.Publish(Event{Type: SpaceCreated, EntityID: "s1"})

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}

	mu.Lock()
	assert.Equal(t, 1, specificCount)
	assert.Equal(t, 1, allCount)
	mu.Unlock()
}

func TestEventTypes(t *testing.T) {
	types := []Type{
		OrgCreated, OrgUpdated, OrgDeleted,
		SpaceCreated, SpaceUpdated, SpaceDeleted,
		BoardCreated, BoardUpdated, BoardDeleted,
		ThreadCreated, ThreadUpdated, ThreadDeleted,
		MessageCreated, MessageUpdated, MessageDeleted,
		UploadCreated, UploadDeleted,
	}
	for _, typ := range types {
		assert.NotEmpty(t, string(typ))
	}
}
