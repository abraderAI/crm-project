package voice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// RecordingService manages call recording lifecycle.
type RecordingService struct {
	provider LiveKitProvider
	service  *Service
}

// NewRecordingService creates a new RecordingService.
func NewRecordingService(provider LiveKitProvider, service *Service) *RecordingService {
	return &RecordingService{provider: provider, service: service}
}

// StartRecording begins a composite audio recording for the specified room.
// Returns the recording info or an error.
func (rs *RecordingService) StartRecording(ctx context.Context, roomName string) (*RecordingInfo, error) {
	if roomName == "" {
		return nil, fmt.Errorf("room_name is required")
	}

	rec, err := rs.provider.StartRecording(ctx, roomName)
	if err != nil {
		return nil, fmt.Errorf("starting recording: %w", err)
	}

	// Add a system message to the thread noting that recording started.
	thread, tErr := rs.service.findThreadByRoomName(ctx, roomName)
	if tErr == nil && thread != nil {
		msgMeta, _ := json.Marshal(map[string]any{
			"event":        "recording_started",
			"recording_id": rec.RecordingID,
			"room_name":    roomName,
		})
		msg := &models.Message{
			ThreadID: thread.ID,
			Body:     "Call recording started.",
			AuthorID: "system",
			Type:     models.MessageTypeCallLog,
			Metadata: string(msgMeta),
		}
		_ = rs.service.db.WithContext(ctx).Create(msg).Error
	}

	return rec, nil
}

// StopRecording stops an active recording and stores the result.
func (rs *RecordingService) StopRecording(ctx context.Context, recordingID string) (*RecordingInfo, error) {
	if recordingID == "" {
		return nil, fmt.Errorf("recording_id is required")
	}

	rec, err := rs.provider.StopRecording(ctx, recordingID)
	if err != nil {
		return nil, fmt.Errorf("stopping recording: %w", err)
	}

	// Add recording metadata to the thread.
	if rec.RoomName != "" {
		thread, tErr := rs.service.findThreadByRoomName(ctx, rec.RoomName)
		if tErr == nil && thread != nil {
			msgMeta, _ := json.Marshal(map[string]any{
				"event":        "recording_stopped",
				"recording_id": rec.RecordingID,
				"duration":     rec.Duration,
				"file_url":     rec.FileURL,
			})
			msg := &models.Message{
				ThreadID: thread.ID,
				Body:     "Call recording stopped.",
				AuthorID: "system",
				Type:     models.MessageTypeCallLog,
				Metadata: string(msgMeta),
			}
			_ = rs.service.db.WithContext(ctx).Create(msg).Error
		}
	}

	return rec, nil
}
