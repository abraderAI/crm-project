package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// EscalateInput holds parameters for escalating a voice call to a human agent.
type EscalateInput struct {
	RoomName   string `json:"room_name"`
	ThreadID   string `json:"thread_id"`
	Reason     string `json:"reason,omitempty"`
	EscalateTo string `json:"escalate_to,omitempty"` // Target agent identity.
}

// EscalateResult holds the result of a human escalation.
type EscalateResult struct {
	ThreadID    string `json:"thread_id"`
	RoomName    string `json:"room_name"`
	EscalatedTo string `json:"escalated_to"`
	EscalatedAt string `json:"escalated_at"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// Escalate transfers a voice call participant to a human agent room.
// It updates thread metadata, creates a system message, and publishes
// an event for the WebSocket notification system.
func (s *Service) Escalate(ctx context.Context, input EscalateInput) (*EscalateResult, error) {
	if input.ThreadID == "" && input.RoomName == "" {
		return nil, fmt.Errorf("thread_id or room_name is required")
	}

	var thread *models.Thread
	var err error

	if input.ThreadID != "" {
		var t models.Thread
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", input.ThreadID).First(&t).Error; err != nil {
			return nil, fmt.Errorf("thread not found: %w", err)
		}
		thread = &t
	} else {
		thread, err = s.findThreadByRoomName(ctx, input.RoomName)
		if err != nil || thread == nil {
			return nil, fmt.Errorf("thread not found for room %s", input.RoomName)
		}
	}

	now := time.Now()
	escalateTo := input.EscalateTo
	if escalateTo == "" {
		escalateTo = "human-agent"
	}

	// Update thread metadata with escalation info.
	var meta map[string]any
	if err := json.Unmarshal([]byte(thread.Metadata), &meta); err != nil {
		meta = make(map[string]any)
	}
	meta["escalated"] = true
	meta["escalated_to"] = escalateTo
	meta["escalated_at"] = now.Format(time.RFC3339)
	meta["escalation_reason"] = input.Reason

	metaBytes, _ := json.Marshal(meta)
	if err := s.db.WithContext(ctx).Model(thread).Update("metadata", string(metaBytes)).Error; err != nil {
		return nil, fmt.Errorf("updating thread metadata: %w", err)
	}

	// Update CallLog status.
	s.db.WithContext(ctx).
		Model(&models.CallLog{}).
		Where("thread_id = ?", thread.ID).
		Updates(map[string]any{"status": string(models.CallStatusEscalated)})

	// Create escalation message.
	msgMeta, _ := json.Marshal(map[string]any{
		"event":       "escalated",
		"escalate_to": escalateTo,
		"reason":      input.Reason,
	})
	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     fmt.Sprintf("Call escalated to %s. Reason: %s", escalateTo, input.Reason),
		AuthorID: "system",
		Type:     models.MessageTypeCallLog,
		Metadata: string(msgMeta),
	}
	_ = s.db.WithContext(ctx).Create(msg).Error

	// Publish event for WebSocket notification.
	s.publishEvent(event.ThreadUpdated, "thread", thread.ID, "")

	return &EscalateResult{
		ThreadID:    thread.ID,
		RoomName:    input.RoomName,
		EscalatedTo: escalateTo,
		EscalatedAt: now.Format(time.RFC3339),
		Status:      "escalated",
		Message:     "Call successfully escalated to human agent.",
	}, nil
}
