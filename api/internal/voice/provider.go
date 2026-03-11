// Package voice provides the VoiceProvider interface and stub implementation
// for call logging, transcript retrieval, and escalation.
package voice

import (
	"context"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// LogCallInput holds the data needed to log a call.
type LogCallInput struct {
	OrgID     string               `json:"org_id"`
	CallerID  string               `json:"caller_id"`
	Direction models.CallDirection `json:"direction"`
	Duration  int                  `json:"duration"`
	Status    models.CallStatus    `json:"status"`
	Metadata  string               `json:"metadata,omitempty"`
}

// EscalateResult holds the result of an escalation action.
type EscalateResult struct {
	CallID   string `json:"call_id"`
	ThreadID string `json:"thread_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// VoiceProvider defines the interface for voice call operations.
// Implementations can range from a stub to real integrations
// (e.g., Bland.ai, Retell, Twilio).
type VoiceProvider interface {
	// LogCall records a call event and returns the created CallLog.
	LogCall(ctx context.Context, input LogCallInput) (*models.CallLog, error)

	// GetTranscript retrieves the transcript for a given call.
	GetTranscript(ctx context.Context, callID string) (string, error)

	// Escalate triggers escalation for a call, creating or updating
	// a support thread and returning the result.
	Escalate(ctx context.Context, callID string) (*EscalateResult, error)
}
