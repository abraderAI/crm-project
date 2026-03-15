// Package email implements the inbound email channel adapter for the IO Channel Gateway.
// It parses incoming emails via IMAP, matches them to CRM threads, processes
// attachments, and normalizes everything into InboundEvents for the gateway.
package email

import (
	"context"
	"fmt"
	"net/mail"
	"sync"
	"time"

	"github.com/abraderAI/crm-project/api/internal/channel"
)

// IMAPProvider abstracts IMAP server interactions for testability.
type IMAPProvider interface {
	// Connect establishes a connection to the IMAP server using the given config.
	Connect(cfg channel.EmailConfig) error
	// StartIDLE begins IDLE monitoring on the given mailbox, calling handler
	// for each new message UID received.
	StartIDLE(mailbox string, handler func(uid uint32)) error
	// FetchMessage retrieves a single message by UID.
	FetchMessage(ctx context.Context, uid uint32) (*mail.Message, error)
	// Close terminates the IMAP connection.
	Close() error
}

// ConnectionState tracks the state of an IMAP connection.
type ConnectionState string

const (
	// ConnectionStateDisconnected indicates no active connection.
	ConnectionStateDisconnected ConnectionState = "disconnected"
	// ConnectionStateConnected indicates the connection is established.
	ConnectionStateConnected ConnectionState = "connected"
	// ConnectionStateIDLE indicates the connection is in IDLE mode.
	ConnectionStateIDLE ConnectionState = "idle"
	// ConnectionStateError indicates the connection experienced an error.
	ConnectionStateError ConnectionState = "error"
)

// MockIMAPProvider is a test double for IMAPProvider.
type MockIMAPProvider struct {
	mu       sync.Mutex
	Messages map[uint32]*mail.Message
	// ConnectFunc allows overriding Connect behavior in tests.
	ConnectFunc func(cfg channel.EmailConfig) error
	// StartIDLEFunc allows overriding StartIDLE behavior in tests.
	StartIDLEFunc func(mailbox string, handler func(uid uint32)) error
	// FetchFunc allows overriding FetchMessage behavior in tests.
	FetchFunc func(ctx context.Context, uid uint32) (*mail.Message, error)
	// CloseFunc allows overriding Close behavior in tests.
	CloseFunc func() error

	ConnectCalls   int
	StartIDLECalls int
	FetchCalls     int
	CloseCalls     int
	Connected      bool
	LastConfig     channel.EmailConfig
}

// NewMockIMAPProvider creates a new mock IMAP provider with empty messages.
func NewMockIMAPProvider() *MockIMAPProvider {
	return &MockIMAPProvider{
		Messages: make(map[uint32]*mail.Message),
	}
}

// Connect records the call and optionally delegates to ConnectFunc.
func (m *MockIMAPProvider) Connect(cfg channel.EmailConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalls++
	m.LastConfig = cfg
	if m.ConnectFunc != nil {
		err := m.ConnectFunc(cfg)
		if err == nil {
			m.Connected = true
		}
		return err
	}
	m.Connected = true
	return nil
}

// StartIDLE records the call and optionally delegates to StartIDLEFunc.
func (m *MockIMAPProvider) StartIDLE(mailbox string, handler func(uid uint32)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartIDLECalls++
	if m.StartIDLEFunc != nil {
		return m.StartIDLEFunc(mailbox, handler)
	}
	return nil
}

// FetchMessage returns the pre-loaded message for the given UID.
func (m *MockIMAPProvider) FetchMessage(ctx context.Context, uid uint32) (*mail.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FetchCalls++
	if m.FetchFunc != nil {
		return m.FetchFunc(ctx, uid)
	}
	msg, ok := m.Messages[uid]
	if !ok {
		return nil, fmt.Errorf("message UID %d not found", uid)
	}
	return msg, nil
}

// Close records the call and optionally delegates to CloseFunc.
func (m *MockIMAPProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalls++
	m.Connected = false
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// AddMessage adds a message to the mock provider for the given UID.
func (m *MockIMAPProvider) AddMessage(uid uint32, msg *mail.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages[uid] = msg
}

// ConnectionPool manages per-org IMAP connections.
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]IMAPProvider // keyed by org ID
	factory     func() IMAPProvider
}

// NewConnectionPool creates a new connection pool with the given provider factory.
func NewConnectionPool(factory func() IMAPProvider) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]IMAPProvider),
		factory:     factory,
	}
}

// Get returns the IMAP provider for the given org, creating one if needed.
func (p *ConnectionPool) Get(orgID string) IMAPProvider {
	p.mu.RLock()
	if conn, ok := p.connections[orgID]; ok {
		p.mu.RUnlock()
		return conn
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	// Double-check after acquiring write lock.
	if conn, ok := p.connections[orgID]; ok {
		return conn
	}
	conn := p.factory()
	p.connections[orgID] = conn
	return conn
}

// Remove closes and removes the connection for the given org.
func (p *ConnectionPool) Remove(orgID string) error {
	p.mu.Lock()
	conn, ok := p.connections[orgID]
	if ok {
		delete(p.connections, orgID)
	}
	p.mu.Unlock()
	if ok {
		return conn.Close()
	}
	return nil
}

// CloseAll closes all connections in the pool.
func (p *ConnectionPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for orgID, conn := range p.connections {
		_ = conn.Close()
		delete(p.connections, orgID)
	}
}

// Size returns the number of active connections.
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// mockClock is a test helper for controlling time in IDLE manager tests.
type mockClock struct {
	mu  sync.Mutex
	now time.Time
}

func newMockClock(t time.Time) *mockClock {
	return &mockClock{now: t}
}

func (c *mockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
