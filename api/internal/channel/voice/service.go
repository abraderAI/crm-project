package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// Service orchestrates voice call lifecycle: inbound webhook processing,
// thread/message creation, recording management, and transcript compilation.
type Service struct {
	db       *gorm.DB
	provider LiveKitProvider
	eventBus *event.Bus
}

// NewService creates a new voice Service.
func NewService(db *gorm.DB, provider LiveKitProvider, eventBus *event.Bus) *Service {
	return &Service{db: db, provider: provider, eventBus: eventBus}
}

// WebhookEvent represents a parsed LiveKit webhook event.
type WebhookEvent struct {
	// Event is the event type (e.g., "room_started", "participant_joined",
	// "room_finished", "track_published").
	Event string `json:"event"`
	Room  struct {
		Name     string `json:"name"`
		SID      string `json:"sid"`
		Metadata string `json:"metadata"`
	} `json:"room"`
	Participant struct {
		Identity string `json:"identity"`
		Name     string `json:"name"`
		SID      string `json:"sid"`
		Metadata string `json:"metadata"`
	} `json:"participant"`
	// EgressInfo is populated for recording-related events.
	EgressInfo *EgressInfo `json:"egress_info,omitempty"`
	CreatedAt  int64       `json:"created_at"`
}

// EgressInfo holds recording egress details from LiveKit webhooks.
type EgressInfo struct {
	EgressID string `json:"egress_id"`
	RoomName string `json:"room_name"`
	Status   string `json:"status"`
	// FileURL is the output recording URL.
	FileURL  string `json:"file_url,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// RoomMetadata holds structured metadata stored in LiveKit room metadata JSON.
type RoomMetadata struct {
	OrgID    string `json:"org_id"`
	ThreadID string `json:"thread_id,omitempty"`
	CallerID string `json:"caller_id,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

// HandleWebhookEvent processes a LiveKit webhook event, managing the full
// call lifecycle from room creation through to transcript/recording storage.
func (s *Service) HandleWebhookEvent(ctx context.Context, evt WebhookEvent) error {
	switch evt.Event {
	case "room_started":
		return s.handleRoomStarted(ctx, evt)
	case "participant_joined":
		return s.handleParticipantJoined(ctx, evt)
	case "room_finished":
		return s.handleRoomFinished(ctx, evt)
	case "egress_ended":
		return s.handleEgressEnded(ctx, evt)
	default:
		// Unknown events are ignored without error.
		return nil
	}
}

// handleRoomStarted creates a new thread and initial call_log message for an inbound call.
func (s *Service) handleRoomStarted(ctx context.Context, evt WebhookEvent) error {
	meta, err := parseRoomMetadata(evt.Room.Metadata)
	if err != nil || meta.OrgID == "" {
		return fmt.Errorf("parsing room metadata: invalid or missing org_id")
	}

	// Find a CRM space and board in the org.
	board, err := s.findOrgBoard(ctx, meta.OrgID)
	if err != nil {
		return fmt.Errorf("finding board for org %s: %w", meta.OrgID, err)
	}

	slugID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generating slug: %w", err)
	}

	title := fmt.Sprintf("Voice call: %s", evt.Room.Name)
	if meta.Phone != "" {
		title = fmt.Sprintf("Voice call from %s", meta.Phone)
	}

	threadMeta, _ := json.Marshal(map[string]any{
		"source":       "voice_channel",
		"room_name":    evt.Room.Name,
		"room_sid":     evt.Room.SID,
		"caller_id":    meta.CallerID,
		"phone":        meta.Phone,
		"channel_type": "voice",
		"status":       "active",
	})

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    title,
		Body:     "Inbound voice call.",
		Slug:     "call-" + slugID.String()[:8],
		Metadata: string(threadMeta),
		AuthorID: "system",
	}
	if err := s.db.WithContext(ctx).Create(thread).Error; err != nil {
		return fmt.Errorf("creating call thread: %w", err)
	}

	// Create the initial call_log message.
	msgMeta, _ := json.Marshal(map[string]any{
		"event":     "room_started",
		"room_name": evt.Room.Name,
		"room_sid":  evt.Room.SID,
		"caller_id": meta.CallerID,
	})
	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     "Voice call started.",
		AuthorID: "system",
		Type:     models.MessageTypeCallLog,
		Metadata: string(msgMeta),
	}
	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("creating call_log message: %w", err)
	}

	// Also create a CallLog record.
	callLog := &models.CallLog{
		OrgID:     meta.OrgID,
		ThreadID:  thread.ID,
		CallerID:  meta.CallerID,
		Direction: models.CallDirectionInbound,
		Status:    models.CallStatusActive,
		Metadata:  string(threadMeta),
	}
	if err := s.db.WithContext(ctx).Create(callLog).Error; err != nil {
		return fmt.Errorf("creating call log: %w", err)
	}

	s.publishEvent(event.ThreadCreated, "thread", thread.ID, meta.OrgID)
	return nil
}

// handleParticipantJoined updates the thread metadata with participant info.
func (s *Service) handleParticipantJoined(ctx context.Context, evt WebhookEvent) error {
	thread, err := s.findThreadByRoomName(ctx, evt.Room.Name)
	if err != nil {
		return err
	}
	if thread == nil {
		return nil // No thread for this room; skip.
	}

	// Add a system message noting the participant joined.
	msgMeta, _ := json.Marshal(map[string]any{
		"event":    "participant_joined",
		"identity": evt.Participant.Identity,
		"name":     evt.Participant.Name,
	})
	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     fmt.Sprintf("Participant %s joined the call.", evt.Participant.Name),
		AuthorID: "system",
		Type:     models.MessageTypeCallLog,
		Metadata: string(msgMeta),
	}
	return s.db.WithContext(ctx).Create(msg).Error
}

// handleRoomFinished updates the thread when a call ends: marks status as completed,
// stores call duration.
func (s *Service) handleRoomFinished(ctx context.Context, evt WebhookEvent) error {
	thread, err := s.findThreadByRoomName(ctx, evt.Room.Name)
	if err != nil {
		return err
	}
	if thread == nil {
		return nil
	}

	// Update thread metadata to mark call as completed.
	var meta map[string]any
	if err := json.Unmarshal([]byte(thread.Metadata), &meta); err != nil {
		meta = make(map[string]any)
	}
	meta["status"] = "completed"
	meta["ended_at"] = time.Now().Format(time.RFC3339)

	metaBytes, _ := json.Marshal(meta)
	if err := s.db.WithContext(ctx).Model(thread).Update("metadata", string(metaBytes)).Error; err != nil {
		return fmt.Errorf("updating thread metadata: %w", err)
	}

	// Update the CallLog record.
	s.db.WithContext(ctx).
		Model(&models.CallLog{}).
		Where("thread_id = ?", thread.ID).
		Updates(map[string]any{"status": string(models.CallStatusCompleted)})

	// Create end-of-call message.
	msgMeta, _ := json.Marshal(map[string]any{
		"event":     "room_finished",
		"room_name": evt.Room.Name,
	})
	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     "Voice call ended.",
		AuthorID: "system",
		Type:     models.MessageTypeCallLog,
		Metadata: string(msgMeta),
	}
	_ = s.db.WithContext(ctx).Create(msg).Error

	s.publishEvent(event.ThreadUpdated, "thread", thread.ID, "")
	return nil
}

// handleEgressEnded processes recording completion: stores file URL in thread.
func (s *Service) handleEgressEnded(ctx context.Context, evt WebhookEvent) error {
	if evt.EgressInfo == nil {
		return nil
	}
	thread, err := s.findThreadByRoomName(ctx, evt.EgressInfo.RoomName)
	if err != nil || thread == nil {
		return err
	}

	msgMeta, _ := json.Marshal(map[string]any{
		"event":        "recording_completed",
		"recording_id": evt.EgressInfo.EgressID,
		"file_url":     evt.EgressInfo.FileURL,
		"duration":     evt.EgressInfo.Duration,
	})
	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     "Call recording available.",
		AuthorID: "system",
		Type:     models.MessageTypeCallLog,
		Metadata: string(msgMeta),
	}
	return s.db.WithContext(ctx).Create(msg).Error
}

// findThreadByRoomName looks up a thread by room_name in metadata.
func (s *Service) findThreadByRoomName(ctx context.Context, roomName string) (*models.Thread, error) {
	var thread models.Thread
	err := s.db.WithContext(ctx).
		Where("json_extract(metadata, '$.room_name') = ?", roomName).
		Where("deleted_at IS NULL").
		First(&thread).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding thread by room name: %w", err)
	}
	return &thread, nil
}

// findOrgBoard finds the first available board in an org (preferring CRM spaces).
func (s *Service) findOrgBoard(ctx context.Context, orgID string) (*models.Board, error) {
	var space models.Space
	err := s.db.WithContext(ctx).
		Where("org_id = ? AND type = ? AND deleted_at IS NULL", orgID, models.SpaceTypeCRM).
		First(&space).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.WithContext(ctx).
			Where("org_id = ? AND deleted_at IS NULL", orgID).
			First(&space).Error
	}
	if err != nil {
		return nil, fmt.Errorf("finding space: %w", err)
	}

	var board models.Board
	if err := s.db.WithContext(ctx).
		Where("space_id = ? AND deleted_at IS NULL", space.ID).
		First(&board).Error; err != nil {
		return nil, fmt.Errorf("finding board: %w", err)
	}
	return &board, nil
}

// GetThreadSummary returns a brief summary of a thread for the agent sidecar.
func (s *Service) GetThreadSummary(ctx context.Context, threadID string) (map[string]any, error) {
	var thread models.Thread
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", threadID).First(&thread).Error; err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}

	var msgCount int64
	s.db.WithContext(ctx).Model(&models.Message{}).Where("thread_id = ?", threadID).Count(&msgCount)

	return map[string]any{
		"id":            thread.ID,
		"title":         thread.Title,
		"body":          thread.Body,
		"metadata":      thread.Metadata,
		"message_count": msgCount,
		"created_at":    thread.CreatedAt,
	}, nil
}

// LookupContact searches for threads associated with an email or phone number.
func (s *Service) LookupContact(ctx context.Context, email, phone string) ([]map[string]any, error) {
	var threads []models.Thread
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")

	if email != "" {
		query = query.Where("json_extract(metadata, '$.contact_email') = ?", email)
	} else if phone != "" {
		query = query.Where("json_extract(metadata, '$.phone') = ?", phone)
	} else {
		return nil, fmt.Errorf("email or phone required")
	}

	if err := query.Order("created_at DESC").Limit(10).Find(&threads).Error; err != nil {
		return nil, fmt.Errorf("looking up contact: %w", err)
	}

	results := make([]map[string]any, 0, len(threads))
	for _, t := range threads {
		results = append(results, map[string]any{
			"id":       t.ID,
			"title":    t.Title,
			"metadata": t.Metadata,
		})
	}
	return results, nil
}

// publishEvent publishes a domain event if the event bus is available.
func (s *Service) publishEvent(evtType event.Type, entityType, entityID, orgID string) {
	if s.eventBus != nil {
		s.eventBus.Publish(event.Event{
			Type:       evtType,
			EntityType: entityType,
			EntityID:   entityID,
			OrgID:      orgID,
		})
	}
}

// parseRoomMetadata decodes room metadata JSON into a RoomMetadata struct.
func parseRoomMetadata(raw string) (*RoomMetadata, error) {
	if raw == "" {
		return &RoomMetadata{}, nil
	}
	var meta RoomMetadata
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
