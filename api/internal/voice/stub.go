package voice

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"gorm.io/gorm"
)

// StubProvider is a stub implementation of VoiceProvider that persists
// call logs to the database but returns canned transcript/escalation data.
type StubProvider struct {
	db *gorm.DB
}

// NewStubProvider creates a new StubProvider.
func NewStubProvider(db *gorm.DB) *StubProvider {
	return &StubProvider{db: db}
}

// LogCall persists a call log to the database.
func (s *StubProvider) LogCall(ctx context.Context, input LogCallInput) (*models.CallLog, error) {
	if input.CallerID == "" {
		return nil, fmt.Errorf("caller_id is required")
	}
	if input.OrgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}
	if !input.Direction.IsValid() {
		input.Direction = models.CallDirectionInbound
	}
	if !input.Status.IsValid() {
		input.Status = models.CallStatusCompleted
	}
	if input.Metadata == "" {
		input.Metadata = "{}"
	}

	callLog := &models.CallLog{
		OrgID:     input.OrgID,
		CallerID:  input.CallerID,
		Direction: input.Direction,
		Duration:  input.Duration,
		Status:    input.Status,
		Metadata:  input.Metadata,
	}

	if err := s.db.WithContext(ctx).Create(callLog).Error; err != nil {
		return nil, fmt.Errorf("creating call log: %w", err)
	}

	return callLog, nil
}

// GetTranscript returns a stub transcript for the given call.
func (s *StubProvider) GetTranscript(ctx context.Context, callID string) (string, error) {
	var callLog models.CallLog
	if err := s.db.WithContext(ctx).Where("id = ?", callID).First(&callLog).Error; err != nil {
		return "", fmt.Errorf("call not found: %w", err)
	}

	if callLog.Transcript != "" {
		return callLog.Transcript, nil
	}

	// Stub: return a canned transcript.
	return "[stub] No real transcript available. Voice provider integration pending.", nil
}

// Escalate marks the call as escalated and returns a stub result.
func (s *StubProvider) Escalate(ctx context.Context, callID string) (*EscalateResult, error) {
	var callLog models.CallLog
	if err := s.db.WithContext(ctx).Where("id = ?", callID).First(&callLog).Error; err != nil {
		return nil, fmt.Errorf("call not found: %w", err)
	}

	callLog.Status = models.CallStatusEscalated
	if err := s.db.WithContext(ctx).Save(&callLog).Error; err != nil {
		return nil, fmt.Errorf("updating call status: %w", err)
	}

	return &EscalateResult{
		CallID:   callLog.ID,
		ThreadID: callLog.ThreadID,
		Status:   string(models.CallStatusEscalated),
		Message:  "[stub] Call escalated. Real escalation integration pending.",
	}, nil
}

// Ensure StubProvider implements VoiceProvider.
var _ VoiceProvider = (*StubProvider)(nil)
