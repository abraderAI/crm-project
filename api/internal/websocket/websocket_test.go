package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	ws "github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/eventbus"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- Hub tests ---

func TestHub_NewHub(t *testing.T) {
	hub := NewHub(testLogger())
	require.NotNil(t, hub)
	assert.Equal(t, 0, hub.ClientCount())
	assert.Equal(t, 0, hub.ChannelCount())
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub(testLogger())
	client := &Client{userID: "user1", send: make(chan []byte, 10)}

	hub.Register(client)
	assert.Equal(t, 1, hub.ClientCount())

	hub.Unregister(client)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_SubscribeUnsubscribe(t *testing.T) {
	hub := NewHub(testLogger())
	client := &Client{userID: "user1", send: make(chan []byte, 10)}

	hub.Register(client)
	hub.Subscribe(client, "board:123")
	assert.Equal(t, 1, hub.ChannelCount())
	assert.Equal(t, 1, hub.ChannelClientCount("board:123"))

	hub.Unsubscribe(client, "board:123")
	assert.Equal(t, 0, hub.ChannelCount())
	assert.Equal(t, 0, hub.ChannelClientCount("board:123"))
}

func TestHub_UnregisterCleansChannels(t *testing.T) {
	hub := NewHub(testLogger())
	client := &Client{userID: "user1", send: make(chan []byte, 10)}

	hub.Register(client)
	hub.Subscribe(client, "board:1")
	hub.Subscribe(client, "thread:2")
	assert.Equal(t, 2, hub.ChannelCount())

	hub.Unregister(client)
	assert.Equal(t, 0, hub.ChannelCount())
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub(testLogger())
	c1 := &Client{userID: "user1", send: make(chan []byte, 10)}
	c2 := &Client{userID: "user2", send: make(chan []byte, 10)}
	c3 := &Client{userID: "user3", send: make(chan []byte, 10)}

	hub.Register(c1)
	hub.Register(c2)
	hub.Register(c3)

	hub.Subscribe(c1, "board:1")
	hub.Subscribe(c2, "board:1")
	// c3 not subscribed to board:1.

	hub.Broadcast("board:1", BroadcastMessage{
		Type:    "message.created",
		Channel: "board:1",
		Payload: map[string]string{"body": "hello"},
	})

	// c1 and c2 should receive.
	select {
	case msg := <-c1.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "message.created", bm.Type)
	case <-time.After(time.Second):
		t.Fatal("c1 timeout")
	}

	select {
	case msg := <-c2.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "message.created", bm.Type)
	case <-time.After(time.Second):
		t.Fatal("c2 timeout")
	}

	// c3 should NOT receive.
	select {
	case <-c3.send:
		t.Fatal("c3 should not receive")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

func TestHub_BroadcastToEmptyChannel(t *testing.T) {
	hub := NewHub(testLogger())
	// Should not panic.
	hub.Broadcast("nonexistent", BroadcastMessage{Type: "test"})
}

func TestHub_MultipleChannels(t *testing.T) {
	hub := NewHub(testLogger())
	c := &Client{userID: "user1", send: make(chan []byte, 10)}

	hub.Register(c)
	hub.Subscribe(c, "board:1")
	hub.Subscribe(c, "thread:1")
	assert.Equal(t, 2, hub.ChannelCount())

	hub.Broadcast("board:1", BroadcastMessage{Type: "board_event"})
	hub.Broadcast("thread:1", BroadcastMessage{Type: "thread_event"})

	msg1 := <-c.send
	var bm1 BroadcastMessage
	require.NoError(t, json.Unmarshal(msg1, &bm1))
	assert.Equal(t, "board_event", bm1.Type)

	msg2 := <-c.send
	var bm2 BroadcastMessage
	require.NoError(t, json.Unmarshal(msg2, &bm2))
	assert.Equal(t, "thread_event", bm2.Type)
}

func TestHub_ChannelClientCount_Empty(t *testing.T) {
	hub := NewHub(testLogger())
	assert.Equal(t, 0, hub.ChannelClientCount("nonexistent"))
}

// --- Broadcaster tests ---

func TestBroadcaster_MessageCreated(t *testing.T) {
	bus := eventbus.New()
	hub := NewHub(testLogger())
	bc := NewBroadcaster(hub, bus, testLogger())
	bc.Start()
	defer bc.Stop()

	c := &Client{userID: "user1", send: make(chan []byte, 10)}
	hub.Register(c)
	hub.Subscribe(c, "thread:t1")

	bus.Publish(eventbus.Event{
		Type:       "message.created",
		EntityType: "message",
		EntityID:   "m1",
		Payload: map[string]any{
			"thread_id": "t1",
			"board_id":  "b1",
		},
	})

	select {
	case msg := <-c.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "message.created", bm.Type)
		assert.Equal(t, "thread:t1", bm.Channel)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcaster_ThreadUpdated(t *testing.T) {
	bus := eventbus.New()
	hub := NewHub(testLogger())
	bc := NewBroadcaster(hub, bus, testLogger())
	bc.Start()
	defer bc.Stop()

	c := &Client{userID: "user1", send: make(chan []byte, 10)}
	hub.Register(c)
	hub.Subscribe(c, "board:b1")

	bus.Publish(eventbus.Event{
		Type:       "thread.updated",
		EntityType: "thread",
		EntityID:   "t1",
		Payload: map[string]any{
			"board_id": "b1",
		},
	})

	select {
	case msg := <-c.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "thread.updated", bm.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBroadcaster_GenericEvent(t *testing.T) {
	bus := eventbus.New()
	hub := NewHub(testLogger())
	bc := NewBroadcaster(hub, bus, testLogger())
	bc.Start()
	defer bc.Stop()

	c := &Client{userID: "user1", send: make(chan []byte, 10)}
	hub.Register(c)
	hub.Subscribe(c, "org:o1")

	bus.Publish(eventbus.Event{
		Type:       "org.updated",
		EntityType: "org",
		EntityID:   "o1",
		Payload:    map[string]any{},
	})

	select {
	case msg := <-c.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "org.updated", bm.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBroadcaster_StopStopsListening(t *testing.T) {
	bus := eventbus.New()
	hub := NewHub(testLogger())
	bc := NewBroadcaster(hub, bus, testLogger())
	bc.Start()
	bc.Stop()

	// After stop, publishing should not cause issues.
	bus.Publish(eventbus.Event{Type: "test"})
}

// --- extractPayloadField tests ---

func TestExtractPayloadField(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		field   string
		want    string
		ok      bool
	}{
		{"valid", map[string]any{"id": "123"}, "id", "123", true},
		{"missing", map[string]any{"id": "123"}, "other", "", false},
		{"nil payload", nil, "id", "", false},
		{"non-map", "string", "id", "", false},
		{"non-string value", map[string]any{"id": 123}, "id", "", false},
		{"empty map", map[string]any{}, "id", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractPayloadField(tt.payload, tt.field)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

// --- Client message tests ---

func TestClientMessage_JSON(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		action string
		ch     string
	}{
		{"subscribe", `{"action":"subscribe","channel":"board:1"}`, "subscribe", "board:1"},
		{"unsubscribe", `{"action":"unsubscribe","channel":"thread:2"}`, "unsubscribe", "thread:2"},
		{"typing", `{"action":"typing","channel":"thread:3"}`, "typing", "thread:3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg ClientMessage
			require.NoError(t, json.Unmarshal([]byte(tt.json), &msg))
			assert.Equal(t, tt.action, msg.Action)
			assert.Equal(t, tt.ch, msg.Channel)
		})
	}
}

func TestBroadcastMessage_JSON(t *testing.T) {
	bm := BroadcastMessage{
		Type:    "message.created",
		Channel: "thread:t1",
		Payload: map[string]string{"body": "hello"},
	}
	data, err := json.Marshal(bm)
	require.NoError(t, err)

	var decoded BroadcastMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "message.created", decoded.Type)
	assert.Equal(t, "thread:t1", decoded.Channel)
}

// --- Client unit tests ---

func TestNewClient(t *testing.T) {
	hub := NewHub(testLogger())
	c := NewClient(nil, hub, "test-user", testLogger())
	require.NotNil(t, c)
	assert.Equal(t, "test-user", c.UserID())
	assert.NotNil(t, c.send)
}

func TestClient_Send_BufferFull(t *testing.T) {
	c := &Client{
		userID: "user1",
		send:   make(chan []byte, 1), // Tiny buffer.
	}
	c.Send([]byte("msg1")) // Should succeed.
	c.Send([]byte("msg2")) // Buffer full, should be dropped (non-blocking).
	assert.Len(t, c.send, 1)
}

func TestClient_HandleMessage_Subscribe(t *testing.T) {
	hub := NewHub(testLogger())
	c := &Client{
		hub:    hub,
		userID: "user1",
		send:   make(chan []byte, 10),
		logger: testLogger(),
	}
	hub.Register(c)

	c.handleMessage(ClientMessage{Action: "subscribe", Channel: "board:1"})
	assert.Equal(t, 1, hub.ChannelClientCount("board:1"))

	// Check ack.
	select {
	case msg := <-c.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "subscribed", bm.Type)
		assert.Equal(t, "board:1", bm.Channel)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestClient_HandleMessage_Unsubscribe(t *testing.T) {
	hub := NewHub(testLogger())
	c := &Client{
		hub:    hub,
		userID: "user1",
		send:   make(chan []byte, 10),
		logger: testLogger(),
	}
	hub.Register(c)
	hub.Subscribe(c, "board:1")

	c.handleMessage(ClientMessage{Action: "unsubscribe", Channel: "board:1"})
	assert.Equal(t, 0, hub.ChannelClientCount("board:1"))

	select {
	case msg := <-c.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "unsubscribed", bm.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestClient_HandleMessage_Typing(t *testing.T) {
	hub := NewHub(testLogger())
	c1 := &Client{
		hub:    hub,
		userID: "user1",
		send:   make(chan []byte, 10),
		logger: testLogger(),
	}
	c2 := &Client{
		hub:    hub,
		userID: "user2",
		send:   make(chan []byte, 10),
		logger: testLogger(),
	}
	hub.Register(c1)
	hub.Register(c2)
	hub.Subscribe(c1, "thread:1")
	hub.Subscribe(c2, "thread:1")

	c1.handleMessage(ClientMessage{Action: "typing", Channel: "thread:1"})

	// Both should receive the typing indicator.
	for _, c := range []*Client{c1, c2} {
		select {
		case msg := <-c.send:
			var bm BroadcastMessage
			require.NoError(t, json.Unmarshal(msg, &bm))
			assert.Equal(t, "typing", bm.Type)
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestClient_HandleMessage_EmptyChannel(t *testing.T) {
	hub := NewHub(testLogger())
	c := &Client{
		hub:    hub,
		userID: "user1",
		send:   make(chan []byte, 10),
		logger: testLogger(),
	}
	hub.Register(c)

	// All actions with empty channel should be no-ops.
	c.handleMessage(ClientMessage{Action: "subscribe", Channel: ""})
	c.handleMessage(ClientMessage{Action: "unsubscribe", Channel: ""})
	c.handleMessage(ClientMessage{Action: "typing", Channel: ""})
	assert.Equal(t, 0, hub.ChannelCount())
}

func TestClient_HandleMessage_Unknown(t *testing.T) {
	hub := NewHub(testLogger())
	c := &Client{
		hub:    hub,
		userID: "user1",
		send:   make(chan []byte, 10),
		logger: testLogger(),
	}
	// Should not panic.
	c.handleMessage(ClientMessage{Action: "unknown", Channel: "ch"})
}

func TestClient_SendAck(t *testing.T) {
	c := &Client{
		userID: "user1",
		send:   make(chan []byte, 10),
	}
	c.sendAck("subscribed", "board:1")

	select {
	case msg := <-c.send:
		var bm BroadcastMessage
		require.NoError(t, json.Unmarshal(msg, &bm))
		assert.Equal(t, "subscribed", bm.Type)
		assert.Equal(t, "board:1", bm.Channel)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// --- Handler tests ---

// mockJWTValidator creates a validator that always succeeds or always fails.
func testWSServer(t *testing.T, hub *Hub, validator *auth.JWTValidator) *httptest.Server {
	t.Helper()
	handler := NewHandler(hub, validator, testLogger(), []string{"*"})
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.Upgrade)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestHandler_Upgrade_NoToken(t *testing.T) {
	hub := NewHub(testLogger())
	validator := auth.NewJWTValidator("http://localhost")
	srv := testWSServer(t, hub, validator)

	resp, err := http.Get(srv.URL + "/ws")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandler_Upgrade_InvalidToken(t *testing.T) {
	hub := NewHub(testLogger())
	validator := auth.NewJWTValidator("http://localhost")
	srv := testWSServer(t, hub, validator)

	resp, err := http.Get(srv.URL + "/ws?token=bad.jwt.token")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandler_NewHandler(t *testing.T) {
	hub := NewHub(testLogger())
	validator := auth.NewJWTValidator("http://localhost")
	h := NewHandler(hub, validator, testLogger(), []string{"http://localhost:3000"})
	require.NotNil(t, h)
	assert.NotNil(t, h.hub)
	assert.NotNil(t, h.validator)
}

// --- Full WebSocket integration test ---

func TestClient_FullLifecycle(t *testing.T) {
	hub := NewHub(testLogger())

	// Create a simple WS echo/subscribe test server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := ws.Accept(w, r, &ws.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		client := NewClient(conn, hub, "test-user", testLogger())
		client.Run(r.Context())
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := ws.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer func() { _ = conn.Close(ws.StatusNormalClosure, "done") }()

	// Send subscribe.
	err = conn.Write(ctx, ws.MessageText, []byte(`{"action":"subscribe","channel":"board:1"}`))
	require.NoError(t, err)

	// Read ack.
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var ack BroadcastMessage
	require.NoError(t, json.Unmarshal(data, &ack))
	assert.Equal(t, "subscribed", ack.Type)
	assert.Equal(t, "board:1", ack.Channel)

	// Send unsubscribe.
	err = conn.Write(ctx, ws.MessageText, []byte(`{"action":"unsubscribe","channel":"board:1"}`))
	require.NoError(t, err)

	_, data, err = conn.Read(ctx)
	require.NoError(t, err)
	var unsubAck BroadcastMessage
	require.NoError(t, json.Unmarshal(data, &unsubAck))
	assert.Equal(t, "unsubscribed", unsubAck.Type)
}

// --- Fuzz tests ---

func FuzzClientMessage(f *testing.F) {
	f.Add(`{"action":"subscribe","channel":"board:1"}`)
	f.Add(`{"action":"unsubscribe","channel":"thread:2"}`)
	f.Add(`{"action":"typing","channel":"thread:3"}`)
	f.Add(`{}`)
	f.Add(`{"action":""}`)
	f.Add(`invalid json`)
	f.Add(`{"action":"subscribe","channel":""}`)
	f.Add(`{"action":"unknown","channel":"test"}`)

	f.Fuzz(func(t *testing.T, input string) {
		var msg ClientMessage
		_ = json.Unmarshal([]byte(input), &msg)
		// Should not panic regardless of input.
	})
}

func FuzzBroadcastMessage(f *testing.F) {
	f.Add("message.created", "board:1", `{"body":"hello"}`)
	f.Add("thread.updated", "thread:2", `{}`)
	f.Add("typing", "thread:3", `{"user_id":"u1"}`)
	f.Add("", "", "")
	f.Add("notification", "user:u1", `{"title":"test","body":"body"}`)

	f.Fuzz(func(t *testing.T, msgType, channel, payloadJSON string) {
		var payload any
		_ = json.Unmarshal([]byte(payloadJSON), &payload)

		bm := BroadcastMessage{
			Type:    msgType,
			Channel: channel,
			Payload: payload,
		}
		data, err := json.Marshal(bm)
		if err == nil {
			var decoded BroadcastMessage
			_ = json.Unmarshal(data, &decoded)
		}
	})
}
