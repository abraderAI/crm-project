package eventbus

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	bus := New()
	require.NotNil(t, bus)
	assert.False(t, bus.closed)
	assert.Empty(t, bus.subscribers)
}

func TestSubscribe_ReceivesEvents(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch, _ := bus.Subscribe("", 10)
	bus.Publish(Event{Type: "test", EntityID: "1"})

	select {
	case e := <-ch:
		assert.Equal(t, "test", e.Type)
		assert.Equal(t, "1", e.EntityID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSubscribe_FilteredEvents(t *testing.T) {
	bus := New()
	defer bus.Close()

	chAll, _ := bus.Subscribe("", 10)
	chFiltered, _ := bus.Subscribe("message.created", 10)

	bus.Publish(Event{Type: "message.created"})
	bus.Publish(Event{Type: "thread.updated"})

	// All subscriber gets both.
	e1 := <-chAll
	assert.Equal(t, "message.created", e1.Type)
	e2 := <-chAll
	assert.Equal(t, "thread.updated", e2.Type)

	// Filtered subscriber only gets matching.
	e3 := <-chFiltered
	assert.Equal(t, "message.created", e3.Type)

	// Should be empty.
	select {
	case <-chFiltered:
		t.Fatal("should not receive non-matching event")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch, unsub := bus.Subscribe("", 10)
	bus.Publish(Event{Type: "before_unsub"})
	<-ch

	unsub()
	bus.Publish(Event{Type: "after_unsub"})

	// Channel should be closed.
	_, ok := <-ch
	assert.False(t, ok)
}

func TestClose(t *testing.T) {
	bus := New()

	ch1, _ := bus.Subscribe("", 10)
	ch2, _ := bus.Subscribe("test", 10)

	bus.Close()

	// All channels should be closed.
	_, ok1 := <-ch1
	assert.False(t, ok1)
	_, ok2 := <-ch2
	assert.False(t, ok2)

	// Publishing after close should not panic.
	bus.Publish(Event{Type: "after_close"})

	// Double close should not panic.
	bus.Close()
}

func TestPublish_SetsTimestamp(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch, _ := bus.Subscribe("", 10)
	before := time.Now()
	bus.Publish(Event{Type: "test"})
	after := time.Now()

	e := <-ch
	assert.False(t, e.Timestamp.IsZero())
	assert.True(t, e.Timestamp.After(before) || e.Timestamp.Equal(before))
	assert.True(t, e.Timestamp.Before(after) || e.Timestamp.Equal(after))
}

func TestPublish_PreservesExistingTimestamp(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch, _ := bus.Subscribe("", 10)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	bus.Publish(Event{Type: "test", Timestamp: ts})

	e := <-ch
	assert.Equal(t, ts, e.Timestamp)
}

func TestPublish_DropsForSlowSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()

	// Buffer size of 1.
	ch, _ := bus.Subscribe("", 1)

	// Fill the buffer.
	bus.Publish(Event{Type: "first"})
	// This should be dropped (non-blocking).
	bus.Publish(Event{Type: "second"})

	e := <-ch
	assert.Equal(t, "first", e.Type)
}

func TestConcurrentPublish(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch, _ := bus.Subscribe("", 1000)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bus.Publish(Event{Type: "concurrent", EntityID: "test"})
		}(i)
	}

	wg.Wait()

	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:
	assert.Equal(t, 100, count)
}

func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, unsub := bus.Subscribe("", 10)
			time.Sleep(time.Millisecond)
			unsub()
		}()
	}
	wg.Wait()
}

func TestSubscribe_DefaultBufferSize(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch, _ := bus.Subscribe("", 0)
	assert.NotNil(t, ch)
}

func TestSubscribe_MultipleSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()

	ch1, _ := bus.Subscribe("", 10)
	ch2, _ := bus.Subscribe("", 10)
	ch3, _ := bus.Subscribe("", 10)

	bus.Publish(Event{Type: "broadcast"})

	assert.Equal(t, "broadcast", (<-ch1).Type)
	assert.Equal(t, "broadcast", (<-ch2).Type)
	assert.Equal(t, "broadcast", (<-ch3).Type)
}

func TestEvent_Fields(t *testing.T) {
	e := Event{
		Type:       "message.created",
		EntityType: "message",
		EntityID:   "msg-123",
		Payload:    map[string]any{"body": "hello"},
		UserID:     "user-1",
		Timestamp:  time.Now(),
	}
	assert.Equal(t, "message.created", e.Type)
	assert.Equal(t, "message", e.EntityType)
	assert.Equal(t, "msg-123", e.EntityID)
	assert.Equal(t, "user-1", e.UserID)
	assert.NotNil(t, e.Payload)
}

func FuzzPublish(f *testing.F) {
	f.Add("test.event", "entity", "id123", "user1")
	f.Add("", "", "", "")
	f.Add("message.created", "message", "msg-abc", "user-xyz")
	f.Add("a.b.c.d.e.f", "type", "id", "uid")

	f.Fuzz(func(t *testing.T, eventType, entityType, entityID, userID string) {
		bus := New()
		defer bus.Close()

		ch, _ := bus.Subscribe("", 10)
		bus.Publish(Event{
			Type:       eventType,
			EntityType: entityType,
			EntityID:   entityID,
			UserID:     userID,
		})

		select {
		case e := <-ch:
			assert.Equal(t, eventType, e.Type)
		case <-time.After(100 * time.Millisecond):
			// Dropped due to timing; acceptable.
		}
	})
}
