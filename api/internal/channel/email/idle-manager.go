package email

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/abraderAI/crm-project/api/internal/channel"
)

const (
	// IDLEBaseDelay is the initial reconnect backoff delay.
	IDLEBaseDelay = 2 * time.Second
	// IDLEMaxDelay is the maximum reconnect backoff delay.
	IDLEMaxDelay = 5 * time.Minute
	// IDLEJitterFraction controls random jitter on backoff (±25%).
	IDLEJitterFraction = 0.25
)

// IDLEManagerConfig holds configuration for the IDLE manager.
type IDLEManagerConfig struct {
	OrgID       string
	EmailConfig channel.EmailConfig
	Provider    IMAPProvider
	OnMessage   func(uid uint32)
	Logger      *slog.Logger
}

// IDLEManager manages an IMAP IDLE connection for a single org.
// It runs a goroutine that maintains IDLE on the configured mailbox,
// reconnects with exponential backoff on disconnect, and supports
// graceful shutdown.
type IDLEManager struct {
	mu     sync.Mutex
	config IDLEManagerConfig
	state  ConnectionState
	cancel context.CancelFunc
	done   chan struct{}
	logger *slog.Logger

	// reconnectAttempts tracks consecutive reconnection failures.
	reconnectAttempts int
	lastError         error
	lastConnectedAt   time.Time

	// sleepFn is injectable for tests.
	sleepFn func(d time.Duration)
}

// NewIDLEManager creates a new IDLE manager for the given org.
func NewIDLEManager(cfg IDLEManagerConfig) *IDLEManager {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &IDLEManager{
		config:  cfg,
		state:   ConnectionStateDisconnected,
		done:    make(chan struct{}),
		logger:  logger,
		sleepFn: time.Sleep,
	}
}

// newIDLEManagerForTest creates an IDLE manager with a custom sleep function for testing.
func newIDLEManagerForTest(cfg IDLEManagerConfig, sleepFn func(time.Duration)) *IDLEManager {
	mgr := NewIDLEManager(cfg)
	mgr.sleepFn = sleepFn
	return mgr
}

// Start begins the IDLE monitoring loop in a background goroutine.
// Returns immediately. Call Stop() to shut down.
func (m *IDLEManager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancel = cancel
	m.mu.Unlock()
	go m.run(ctx)
}

// Stop gracefully shuts down the IDLE manager.
func (m *IDLEManager) Stop() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	<-m.done
}

// State returns the current connection state.
func (m *IDLEManager) State() ConnectionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// LastError returns the most recent error, if any.
func (m *IDLEManager) LastError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastError
}

// ReconnectAttempts returns the number of consecutive reconnection failures.
func (m *IDLEManager) ReconnectAttempts() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reconnectAttempts
}

// IsHealthy returns true if the connection is in IDLE or connected state.
func (m *IDLEManager) IsHealthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state == ConnectionStateIDLE || m.state == ConnectionStateConnected
}

// run is the main IDLE loop.
func (m *IDLEManager) run(ctx context.Context) {
	defer close(m.done)
	defer func() {
		_ = m.config.Provider.Close()
		m.setState(ConnectionStateDisconnected)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Connect.
		if err := m.config.Provider.Connect(m.config.EmailConfig); err != nil {
			m.setError(err)
			m.logger.Error("IDLE connect failed",
				"org_id", m.config.OrgID,
				"attempt", m.reconnectAttempts,
				"error", err,
			)
			if m.waitBackoff(ctx) {
				return
			}
			continue
		}

		m.setState(ConnectionStateConnected)
		m.mu.Lock()
		m.reconnectAttempts = 0
		m.lastConnectedAt = time.Now()
		m.mu.Unlock()

		// Start IDLE.
		mailbox := m.config.EmailConfig.Mailbox
		if mailbox == "" {
			mailbox = "INBOX"
		}

		m.setState(ConnectionStateIDLE)
		err := m.config.Provider.StartIDLE(mailbox, m.config.OnMessage)
		if err != nil {
			m.setError(err)
			m.logger.Warn("IDLE disconnected",
				"org_id", m.config.OrgID,
				"error", err,
			)
			_ = m.config.Provider.Close()

			select {
			case <-ctx.Done():
				return
			default:
			}

			if m.waitBackoff(ctx) {
				return
			}
			continue
		}

		// IDLE returned without error — usually means the connection was closed cleanly.
		_ = m.config.Provider.Close()

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// waitBackoff sleeps for the computed backoff duration. Returns true if context was cancelled.
func (m *IDLEManager) waitBackoff(ctx context.Context) bool {
	m.mu.Lock()
	m.reconnectAttempts++
	attempt := m.reconnectAttempts
	m.mu.Unlock()

	delay := computeIDLEBackoff(attempt)
	m.logger.Info("IDLE backoff",
		"org_id", m.config.OrgID,
		"delay", delay,
		"attempt", attempt,
	)

	// Use sleepFn in a goroutine so it's cancellable via context.
	done := make(chan struct{})
	go func() {
		m.sleepFn(delay)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return true
	case <-done:
		return false
	}
}

func (m *IDLEManager) setState(state ConnectionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

func (m *IDLEManager) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = err
	m.state = ConnectionStateError
}

// computeIDLEBackoff calculates exponential backoff with jitter for reconnection attempts.
func computeIDLEBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return IDLEBaseDelay
	}
	delay := float64(IDLEBaseDelay) * math.Pow(2, float64(attempt-1))
	if delay > float64(IDLEMaxDelay) {
		delay = float64(IDLEMaxDelay)
	}
	// Apply ±JitterFraction random jitter.
	jitter := delay * IDLEJitterFraction * (rand.Float64()*2 - 1) //nolint:gosec
	result := time.Duration(delay + jitter)
	if result < 0 {
		result = IDLEBaseDelay
	}
	return result
}

// HealthReport returns a summary of the IDLE manager's health status.
func (m *IDLEManager) HealthReport() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	report := map[string]any{
		"org_id":             m.config.OrgID,
		"state":              string(m.state),
		"reconnect_attempts": m.reconnectAttempts,
	}
	if m.lastError != nil {
		report["last_error"] = m.lastError.Error()
	}
	if !m.lastConnectedAt.IsZero() {
		report["last_connected_at"] = m.lastConnectedAt.Format(time.RFC3339)
	}
	return report
}

// IDLEManagerRegistry manages IDLE managers for multiple orgs.
type IDLEManagerRegistry struct {
	mu       sync.RWMutex
	managers map[string]*IDLEManager // keyed by org ID
}

// NewIDLEManagerRegistry creates a new registry.
func NewIDLEManagerRegistry() *IDLEManagerRegistry {
	return &IDLEManagerRegistry{
		managers: make(map[string]*IDLEManager),
	}
}

// Register adds and starts an IDLE manager for the given org.
// If one already exists for the org, it is stopped and replaced.
func (r *IDLEManagerRegistry) Register(orgID string, mgr *IDLEManager) {
	r.mu.Lock()
	if existing, ok := r.managers[orgID]; ok {
		existing.Stop()
	}
	r.managers[orgID] = mgr
	r.mu.Unlock()
	mgr.Start()
}

// Deregister stops and removes the IDLE manager for the given org.
func (r *IDLEManagerRegistry) Deregister(orgID string) {
	r.mu.Lock()
	mgr, ok := r.managers[orgID]
	if ok {
		delete(r.managers, orgID)
	}
	r.mu.Unlock()
	if ok {
		mgr.Stop()
	}
}

// StopAll stops all IDLE managers.
func (r *IDLEManagerRegistry) StopAll() {
	r.mu.Lock()
	managers := make([]*IDLEManager, 0, len(r.managers))
	for _, mgr := range r.managers {
		managers = append(managers, mgr)
	}
	r.managers = make(map[string]*IDLEManager)
	r.mu.Unlock()
	for _, mgr := range managers {
		mgr.Stop()
	}
}

// HealthReports returns health summaries for all registered managers.
func (r *IDLEManagerRegistry) HealthReports() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reports := make([]map[string]any, 0, len(r.managers))
	for _, mgr := range r.managers {
		reports = append(reports, mgr.HealthReport())
	}
	return reports
}

// Size returns the number of registered managers.
func (r *IDLEManagerRegistry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.managers)
}

// Get returns the IDLE manager for the given org, or nil if not found.
func (r *IDLEManagerRegistry) Get(orgID string) *IDLEManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.managers[orgID]
}

// IsHealthy returns true if the org's manager exists and is healthy.
func (r *IDLEManagerRegistry) IsHealthy(orgID string) bool {
	r.mu.RLock()
	mgr, ok := r.managers[orgID]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	return mgr.IsHealthy()
}

// FormatError returns a user-friendly error message for IDLE health issues.
func FormatError(orgID string, err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("email IDLE error for org %s: %v", orgID, err)
}
