package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// TranscriptEntry represents a single utterance in a call transcript.
type TranscriptEntry struct {
	// Speaker is the participant identity (e.g., "agent", "caller").
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
	// StartTime is seconds from call start.
	StartTime float64 `json:"start_time"`
	// EndTime is seconds from call start.
	EndTime float64 `json:"end_time"`
}

// TranscriptEvent is the raw event emitted by the LiveKit agent during a call.
type TranscriptEvent struct {
	RoomName  string  `json:"room_name"`
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	IsFinal   bool    `json:"is_final"`
}

// Transcript holds the full compiled transcript for a call.
type Transcript struct {
	RoomName string            `json:"room_name"`
	Entries  []TranscriptEntry `json:"entries"`
	// FullText is the human-readable rendering with speaker labels.
	FullText string `json:"full_text"`
}

// CompileTranscript assembles a sorted, speaker-labeled transcript from
// a slice of TranscriptEvent entries.
func CompileTranscript(roomName string, events []TranscriptEvent) *Transcript {
	// Filter to final events only and sort by start time.
	var finals []TranscriptEvent
	for _, e := range events {
		if e.IsFinal && e.Text != "" {
			finals = append(finals, e)
		}
	}
	sort.Slice(finals, func(i, j int) bool {
		return finals[i].StartTime < finals[j].StartTime
	})

	entries := make([]TranscriptEntry, 0, len(finals))
	var sb strings.Builder
	for _, e := range finals {
		entries = append(entries, TranscriptEntry{
			Speaker:   e.Speaker,
			Text:      e.Text,
			StartTime: e.StartTime,
			EndTime:   e.EndTime,
		})
		fmt.Fprintf(&sb, "[%s] %s\n", e.Speaker, e.Text)
	}

	return &Transcript{
		RoomName: roomName,
		Entries:  entries,
		FullText: sb.String(),
	}
}

// StoreTranscript compiles the transcript and saves it as a call_log message
// on the associated thread.
func (s *Service) StoreTranscript(ctx context.Context, roomName string, events []TranscriptEvent) error {
	transcript := CompileTranscript(roomName, events)

	thread, err := s.findThreadByRoomName(ctx, roomName)
	if err != nil || thread == nil {
		return fmt.Errorf("thread not found for room %s", roomName)
	}

	transcriptJSON, _ := json.Marshal(transcript)
	msgMeta, _ := json.Marshal(map[string]any{
		"event":      "transcript",
		"room_name":  roomName,
		"entries":    len(transcript.Entries),
		"transcript": json.RawMessage(transcriptJSON),
	})

	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     transcript.FullText,
		AuthorID: "system",
		Type:     models.MessageTypeCallLog,
		Metadata: string(msgMeta),
	}
	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("storing transcript: %w", err)
	}

	// Also update the CallLog record transcript field.
	s.db.WithContext(ctx).
		Model(&models.CallLog{}).
		Where("thread_id = ?", thread.ID).
		Update("transcript", transcript.FullText)

	return nil
}
