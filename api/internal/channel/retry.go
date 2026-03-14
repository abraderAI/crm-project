package channel

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/abraderAI/crm-project/api/internal/models"
)

const (
	// MaxRetries is the maximum number of process attempts before DLQ insertion.
	MaxRetries = 5
	// BaseDelay is the initial backoff delay (2^0 seconds).
	BaseDelay = 1 * time.Second
	// MaxDelay is the cap on backoff delay (2^4 seconds).
	MaxDelay = 16 * time.Second
	// JitterFraction controls the magnitude of random jitter (±25% of the base delay).
	JitterFraction = 0.25
)

// RetryFunc is a function that processes an InboundEvent and may return an error.
type RetryFunc func(ctx context.Context, evt *InboundEvent) error

// RetryEngine processes failed inbound events with exponential backoff and DLQ fallback.
// After MaxRetries failures the event is inserted into the dead letter queue.
type RetryEngine struct {
	repo *Repository
	// sleepFn is injectable for tests; defaults to time.Sleep.
	sleepFn func(d time.Duration)
}

// NewRetryEngine creates a new RetryEngine using the real time.Sleep.
func NewRetryEngine(repo *Repository) *RetryEngine {
	return &RetryEngine{repo: repo, sleepFn: time.Sleep}
}

// newRetryEngineWithSleep creates a RetryEngine with a custom sleep function for testing.
func newRetryEngineWithSleep(repo *Repository, sleepFn func(d time.Duration)) *RetryEngine {
	return &RetryEngine{repo: repo, sleepFn: sleepFn}
}

// ProcessWithRetry attempts to process an InboundEvent using fn, retrying on failure
// with exponential backoff. After MaxRetries failures the event is inserted into the DLQ
// and an error wrapping the last processing error is returned.
func (re *RetryEngine) ProcessWithRetry(ctx context.Context, evt *InboundEvent, fn RetryFunc) error {
	var lastErr error
	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			delay := computeBackoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				re.sleepFn(delay)
			}
		}
		if err := fn(ctx, evt); err != nil {
			lastErr = err
			continue
		}
		return nil // Success.
	}

	// All attempts exhausted — insert into DLQ.
	if dlqErr := re.insertDLQ(ctx, evt, lastErr); dlqErr != nil {
		return fmt.Errorf("inserting to DLQ after %d retries (last error: %v): %w", MaxRetries, lastErr, dlqErr)
	}
	return fmt.Errorf("event moved to DLQ after %d retries: %w", MaxRetries, lastErr)
}

// insertDLQ creates a DeadLetterEvent record for an exhausted event.
func (re *RetryEngine) insertDLQ(ctx context.Context, evt *InboundEvent, lastErr error) error {
	now := time.Now()
	errMsg := ""
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	payload := evt.Metadata
	if payload == "" {
		payload = "{}"
	}
	dlqEvt := &models.DeadLetterEvent{
		OrgID:         evt.OrgID,
		ChannelType:   evt.ChannelType,
		EventPayload:  payload,
		ErrorMessage:  errMsg,
		Attempts:      MaxRetries,
		LastAttemptAt: &now,
		Status:        models.DLQStatusFailed,
	}
	return re.repo.CreateDLQEvent(ctx, dlqEvt)
}

// computeBackoff returns the backoff duration for the given attempt index (0-based, but
// the delay is computed for the retry *after* attempt 0, so attempt=1 → 1 s, 2 → 2 s, …).
// Delay is capped at MaxDelay and jittered by ±JitterFraction.
func computeBackoff(attempt int) time.Duration {
	delay := BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= MaxDelay {
			delay = MaxDelay
			break
		}
	}
	// Apply ±JitterFraction random jitter.
	jitter := time.Duration(float64(delay) * JitterFraction * (rand.Float64()*2 - 1)) //nolint:gosec
	result := delay + jitter
	if result < 0 {
		result = 0
	}
	return result
}
