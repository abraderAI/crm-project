package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Test helpers ---

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func createTestOrg(t *testing.T, db *gorm.DB) *models.Org {
	t.Helper()
	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org
}

func createTestSpaceAndBoard(t *testing.T, db *gorm.DB, orgID string) (*models.Space, *models.Board) {
	t.Helper()
	space := &models.Space{OrgID: orgID, Name: "CRM Space", Slug: "crm-space", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Pipeline", Slug: "pipeline", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return space, board
}

func createAdminMembership(t *testing.T, db *gorm.DB, orgID, userID string) {
	t.Helper()
	m := &models.OrgMembership{OrgID: orgID, UserID: userID, Role: models.RoleAdmin}
	require.NoError(t, db.Create(m).Error)
}

// --- MockProvider tests ---

func TestMockProvider_Implements_Interface(t *testing.T) {
	var _ LiveKitProvider = NewMockProvider()
}

func TestMockProvider_CreateRoom(t *testing.T) {
	p := NewMockProvider()
	room, err := p.CreateRoom(context.Background(), "test-room", "{}")
	require.NoError(t, err)
	assert.NotEmpty(t, room.ID)
	assert.Equal(t, "test-room", room.Name)
	assert.NotNil(t, p.GetRoom("test-room"))
}

func TestMockProvider_CreateRoom_Error(t *testing.T) {
	p := NewMockProvider()
	p.CreateRoomErr = fmt.Errorf("create room error")
	_, err := p.CreateRoom(context.Background(), "room", "{}")
	assert.Error(t, err)
}

func TestMockProvider_ListParticipants(t *testing.T) {
	p := NewMockProvider()
	_, _ = p.CreateRoom(context.Background(), "room-1", "{}")
	p.AddMockParticipant("room-1", Participant{Identity: "user-1", Name: "User One", State: "joined"})

	parts, err := p.ListParticipants(context.Background(), "room-1")
	require.NoError(t, err)
	assert.Len(t, parts, 1)
	assert.Equal(t, "user-1", parts[0].Identity)
}

func TestMockProvider_RemoveParticipant(t *testing.T) {
	p := NewMockProvider()
	p.AddMockParticipant("room-1", Participant{Identity: "user-1"})

	err := p.RemoveParticipant(context.Background(), "room-1", "user-1")
	require.NoError(t, err)

	parts, _ := p.ListParticipants(context.Background(), "room-1")
	assert.Len(t, parts, 0)
}

func TestMockProvider_RemoveParticipant_NotFound(t *testing.T) {
	p := NewMockProvider()
	err := p.RemoveParticipant(context.Background(), "room-1", "unknown")
	assert.Error(t, err)
}

func TestMockProvider_SIPDispatchRule(t *testing.T) {
	p := NewMockProvider()
	rule, err := p.CreateSIPDispatchRule(context.Background(), SIPDispatchRule{
		Name: "test-rule", TrunkID: "trunk-1", PhoneNumberID: "phone-1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)

	err = p.DeleteSIPDispatchRule(context.Background(), rule.ID)
	require.NoError(t, err)
}

func TestMockProvider_SIPDispatchRule_DeleteNotFound(t *testing.T) {
	p := NewMockProvider()
	err := p.DeleteSIPDispatchRule(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestMockProvider_SearchPhoneNumbers(t *testing.T) {
	p := NewMockProvider()
	nums, err := p.SearchPhoneNumbers(context.Background(), "415")
	require.NoError(t, err)
	assert.Len(t, nums, 2)
	assert.Equal(t, "415", nums[0].AreaCode)
}

func TestMockProvider_PurchasePhoneNumber(t *testing.T) {
	p := NewMockProvider()
	num, err := p.PurchasePhoneNumber(context.Background(), "num-1")
	require.NoError(t, err)
	assert.True(t, num.Provisioned)
}

func TestMockProvider_ListPhoneNumbers(t *testing.T) {
	p := NewMockProvider()
	_, _ = p.PurchasePhoneNumber(context.Background(), "num-1")
	nums, err := p.ListPhoneNumbers(context.Background())
	require.NoError(t, err)
	assert.Len(t, nums, 1)
}

func TestMockProvider_Recording(t *testing.T) {
	p := NewMockProvider()
	rec, err := p.StartRecording(context.Background(), "room-1")
	require.NoError(t, err)
	assert.Equal(t, "recording", rec.Status)

	stopped, err := p.StopRecording(context.Background(), rec.RecordingID)
	require.NoError(t, err)
	assert.Equal(t, "stopped", stopped.Status)
}

func TestMockProvider_StopRecording_NotFound(t *testing.T) {
	p := NewMockProvider()
	_, err := p.StopRecording(context.Background(), "bad-id")
	assert.Error(t, err)
}

func TestMockProvider_ErrorInjection(t *testing.T) {
	p := NewMockProvider()
	p.ListParticipantsErr = fmt.Errorf("test")
	_, err := p.ListParticipants(context.Background(), "r")
	assert.Error(t, err)

	p.RemoveParticipantErr = fmt.Errorf("test")
	err = p.RemoveParticipant(context.Background(), "r", "p")
	assert.Error(t, err)

	p.CreateSIPDispatchRuleErr = fmt.Errorf("test")
	_, err = p.CreateSIPDispatchRule(context.Background(), SIPDispatchRule{})
	assert.Error(t, err)

	p.DeleteSIPDispatchRuleErr = fmt.Errorf("test")
	err = p.DeleteSIPDispatchRule(context.Background(), "id")
	assert.Error(t, err)

	p.SearchPhoneNumbersErr = fmt.Errorf("test")
	_, err = p.SearchPhoneNumbers(context.Background(), "415")
	assert.Error(t, err)

	p.PurchasePhoneNumberErr = fmt.Errorf("test")
	_, err = p.PurchasePhoneNumber(context.Background(), "id")
	assert.Error(t, err)

	p.ListPhoneNumbersErr = fmt.Errorf("test")
	_, err = p.ListPhoneNumbers(context.Background())
	assert.Error(t, err)

	p.StartRecordingErr = fmt.Errorf("test")
	_, err = p.StartRecording(context.Background(), "r")
	assert.Error(t, err)

	p.StopRecordingErr = fmt.Errorf("test")
	_, err = p.StopRecording(context.Background(), "id")
	assert.Error(t, err)
}

// --- Service tests ---

func TestService_HandleWebhookEvent_RoomStarted(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	provider := NewMockProvider()
	bus := event.NewBus()
	svc := NewService(db, provider, bus)

	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID, CallerID: "caller-1", Phone: "+15551234567"})
	evt := WebhookEvent{Event: "room_started"}
	evt.Room.Name = "room-abc"
	evt.Room.SID = "sid-abc"
	evt.Room.Metadata = string(roomMeta)

	err := svc.HandleWebhookEvent(context.Background(), evt)
	require.NoError(t, err)

	// Verify thread was created.
	var thread models.Thread
	require.NoError(t, db.Where("json_extract(metadata, '$.room_name') = ?", "room-abc").First(&thread).Error)
	assert.Contains(t, thread.Title, "+15551234567")
	assert.Contains(t, thread.Metadata, "voice_channel")

	// Verify call_log message was created.
	var msg models.Message
	require.NoError(t, db.Where("thread_id = ? AND type = ?", thread.ID, models.MessageTypeCallLog).First(&msg).Error)
	assert.Equal(t, "Voice call started.", msg.Body)

	// Verify CallLog record.
	var callLog models.CallLog
	require.NoError(t, db.Where("thread_id = ?", thread.ID).First(&callLog).Error)
	assert.Equal(t, models.CallStatusActive, callLog.Status)
}

func TestService_HandleWebhookEvent_RoomStarted_NoOrgID(t *testing.T) {
	db := testDB(t)
	svc := NewService(db, NewMockProvider(), nil)

	evt := WebhookEvent{Event: "room_started"}
	evt.Room.Metadata = `{}`
	err := svc.HandleWebhookEvent(context.Background(), evt)
	assert.Error(t, err)
}

func TestService_HandleWebhookEvent_ParticipantJoined(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	svc := NewService(db, NewMockProvider(), nil)

	// First create a room.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID, CallerID: "caller"})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-join"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	// Then a participant joins.
	joinEvt := WebhookEvent{Event: "participant_joined"}
	joinEvt.Room.Name = "room-join"
	joinEvt.Participant.Identity = "agent-1"
	joinEvt.Participant.Name = "Agent One"
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), joinEvt))

	// Verify participant message.
	var count int64
	db.Model(&models.Message{}).Where("body LIKE ?", "%Agent One joined%").Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestService_HandleWebhookEvent_RoomFinished(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	bus := event.NewBus()
	svc := NewService(db, NewMockProvider(), bus)

	// Create room.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-end"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	// End room.
	endEvt := WebhookEvent{Event: "room_finished"}
	endEvt.Room.Name = "room-end"
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), endEvt))

	// Verify thread metadata updated.
	var thread models.Thread
	require.NoError(t, db.Where("json_extract(metadata, '$.room_name') = ?", "room-end").First(&thread).Error)
	assert.Contains(t, thread.Metadata, "completed")
}

func TestService_HandleWebhookEvent_EgressEnded(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	svc := NewService(db, NewMockProvider(), nil)

	// Create room first.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-rec"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	// Egress (recording) completed.
	egressEvt := WebhookEvent{
		Event: "egress_ended",
		EgressInfo: &EgressInfo{
			EgressID: "egress-1",
			RoomName: "room-rec",
			FileURL:  "https://storage.example.com/rec.ogg",
			Duration: 180,
		},
	}
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), egressEvt))

	// Verify recording message.
	var msg models.Message
	err := db.Where("body = ?", "Call recording available.").First(&msg).Error
	require.NoError(t, err)
	assert.Contains(t, msg.Metadata, "recording_completed")
}

func TestService_HandleWebhookEvent_UnknownType(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	err := svc.HandleWebhookEvent(context.Background(), WebhookEvent{Event: "unknown_event"})
	assert.NoError(t, err) // Unknown events are silently ignored.
}

func TestService_HandleWebhookEvent_EgressEnded_NilInfo(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	err := svc.HandleWebhookEvent(context.Background(), WebhookEvent{Event: "egress_ended"})
	assert.NoError(t, err)
}

// --- Escalation tests ---

func TestService_Escalate_ByThreadID(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	bus := event.NewBus()
	svc := NewService(db, NewMockProvider(), bus)

	// Create a room/thread.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-esc"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	var thread models.Thread
	require.NoError(t, db.Where("json_extract(metadata, '$.room_name') = ?", "room-esc").First(&thread).Error)

	result, err := svc.Escalate(context.Background(), EscalateInput{
		ThreadID:   thread.ID,
		Reason:     "Customer unhappy",
		EscalateTo: "senior-agent",
	})
	require.NoError(t, err)
	assert.Equal(t, "escalated", result.Status)
	assert.Equal(t, "senior-agent", result.EscalatedTo)

	// Verify thread metadata.
	var updated models.Thread
	require.NoError(t, db.Where("id = ?", thread.ID).First(&updated).Error)
	assert.Contains(t, updated.Metadata, "escalated")
}

func TestService_Escalate_MissingInput(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	_, err := svc.Escalate(context.Background(), EscalateInput{})
	assert.Error(t, err)
}

func TestService_Escalate_ThreadNotFound(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	_, err := svc.Escalate(context.Background(), EscalateInput{ThreadID: "nonexistent"})
	assert.Error(t, err)
}

// --- Transcript tests ---

func TestCompileTranscript(t *testing.T) {
	events := []TranscriptEvent{
		{RoomName: "room-1", Speaker: "agent", Text: "Hello!", StartTime: 0.0, EndTime: 1.0, IsFinal: true},
		{RoomName: "room-1", Speaker: "caller", Text: "Hi there", StartTime: 1.5, EndTime: 3.0, IsFinal: true},
		{RoomName: "room-1", Speaker: "agent", Text: "partial", StartTime: 3.5, EndTime: 4.0, IsFinal: false}, // Not final — excluded.
		{RoomName: "room-1", Speaker: "agent", Text: "How can I help?", StartTime: 3.5, EndTime: 5.0, IsFinal: true},
	}

	transcript := CompileTranscript("room-1", events)
	assert.Equal(t, "room-1", transcript.RoomName)
	assert.Len(t, transcript.Entries, 3)
	assert.Contains(t, transcript.FullText, "[agent] Hello!")
	assert.Contains(t, transcript.FullText, "[caller] Hi there")
	assert.Contains(t, transcript.FullText, "[agent] How can I help?")
	assert.NotContains(t, transcript.FullText, "partial")
}

func TestCompileTranscript_Empty(t *testing.T) {
	transcript := CompileTranscript("room-empty", nil)
	assert.Empty(t, transcript.Entries)
	assert.Empty(t, transcript.FullText)
}

func TestCompileTranscript_SortsbyTime(t *testing.T) {
	events := []TranscriptEvent{
		{Speaker: "b", Text: "second", StartTime: 2.0, EndTime: 3.0, IsFinal: true},
		{Speaker: "a", Text: "first", StartTime: 1.0, EndTime: 2.0, IsFinal: true},
	}
	transcript := CompileTranscript("room", events)
	assert.Equal(t, "a", transcript.Entries[0].Speaker)
	assert.Equal(t, "b", transcript.Entries[1].Speaker)
}

func TestService_StoreTranscript(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	svc := NewService(db, NewMockProvider(), nil)

	// Create room.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-trans"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	events := []TranscriptEvent{
		{Speaker: "agent", Text: "Hello", StartTime: 0, EndTime: 1, IsFinal: true},
		{Speaker: "caller", Text: "Hi", StartTime: 1, EndTime: 2, IsFinal: true},
	}
	err := svc.StoreTranscript(context.Background(), "room-trans", events)
	require.NoError(t, err)

	// Verify transcript message.
	var msg models.Message
	err = db.Where("body LIKE ? AND type = ?", "%[agent] Hello%", models.MessageTypeCallLog).First(&msg).Error
	require.NoError(t, err)
	assert.Contains(t, msg.Metadata, "transcript")
}

func TestService_StoreTranscript_NoThread(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	err := svc.StoreTranscript(context.Background(), "nonexistent-room", nil)
	assert.Error(t, err)
}

// --- Recording tests ---

func TestRecordingService_StartStop(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	provider := NewMockProvider()
	svc := NewService(db, provider, nil)
	recSvc := NewRecordingService(provider, svc)

	// Create room.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-rec2"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	rec, err := recSvc.StartRecording(context.Background(), "room-rec2")
	require.NoError(t, err)
	assert.Equal(t, "recording", rec.Status)

	stopped, err := recSvc.StopRecording(context.Background(), rec.RecordingID)
	require.NoError(t, err)
	assert.Equal(t, "stopped", stopped.Status)
}

func TestRecordingService_StartEmpty(t *testing.T) {
	recSvc := NewRecordingService(NewMockProvider(), NewService(testDB(t), NewMockProvider(), nil))
	_, err := recSvc.StartRecording(context.Background(), "")
	assert.Error(t, err)
}

func TestRecordingService_StopEmpty(t *testing.T) {
	recSvc := NewRecordingService(NewMockProvider(), NewService(testDB(t), NewMockProvider(), nil))
	_, err := recSvc.StopRecording(context.Background(), "")
	assert.Error(t, err)
}

// --- Webhook handler tests ---

func TestWebhookHandler_Success(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	svc := NewService(db, NewMockProvider(), nil)
	handler := NewWebhookHandler(svc, "test-token")

	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	payload, _ := json.Marshal(WebhookEvent{
		Event: "room_started",
		Room: struct {
			Name     string `json:"name"`
			SID      string `json:"sid"`
			Metadata string `json:"metadata"`
		}{Name: "room-wh", SID: "sid-wh", Metadata: string(roomMeta)},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/livekit", bytes.NewReader(payload))
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookHandler_InvalidAuth(t *testing.T) {
	handler := NewWebhookHandler(NewService(testDB(t), NewMockProvider(), nil), "secret")
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/livekit", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "wrong")
	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookHandler_InvalidPayload(t *testing.T) {
	handler := NewWebhookHandler(NewService(testDB(t), NewMockProvider(), nil), "")
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/livekit", bytes.NewReader([]byte(`not json`)))
	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookHandler_NoAuthRequired(t *testing.T) {
	handler := NewWebhookHandler(NewService(testDB(t), NewMockProvider(), nil), "")
	payload, _ := json.Marshal(WebhookEvent{Event: "unknown"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookHandler_BearerAuth(t *testing.T) {
	handler := NewWebhookHandler(NewService(testDB(t), NewMockProvider(), nil), "my-token")
	payload, _ := json.Marshal(WebhookEvent{Event: "unknown"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer my-token")
	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Bridge handler tests ---

func TestBridgeHandler_LookupContact(t *testing.T) {
	db := testDB(t)
	svc := NewService(db, NewMockProvider(), nil)
	handler := NewBridgeHandler(svc, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/internal/contacts/lookup?email=test@test.com", nil)
	w := httptest.NewRecorder()
	handler.LookupContact(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBridgeHandler_LookupContact_MissingParams(t *testing.T) {
	handler := NewBridgeHandler(NewService(testDB(t), NewMockProvider(), nil), "")
	req := httptest.NewRequest(http.MethodGet, "/v1/internal/contacts/lookup", nil)
	w := httptest.NewRecorder()
	handler.LookupContact(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBridgeHandler_LookupContact_WithInternalKey(t *testing.T) {
	handler := NewBridgeHandler(NewService(testDB(t), NewMockProvider(), nil), "secret-key")

	// Without key.
	req := httptest.NewRequest(http.MethodGet, "/v1/internal/contacts/lookup?email=a@b.com", nil)
	w := httptest.NewRecorder()
	handler.LookupContact(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// With correct key.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/internal/contacts/lookup?email=a@b.com", nil)
	req2.Header.Set("X-Internal-Key", "secret-key")
	w2 := httptest.NewRecorder()
	handler.LookupContact(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestBridgeHandler_GetThreadSummary(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	svc := NewService(db, NewMockProvider(), nil)
	handler := NewBridgeHandler(svc, "")

	// Create a thread.
	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-summary"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	var thread models.Thread
	require.NoError(t, db.Where("json_extract(metadata, '$.room_name') = ?", "room-summary").First(&thread).Error)

	r := chi.NewRouter()
	r.Get("/v1/internal/threads/{id}/summary", handler.GetThreadSummary)

	req := httptest.NewRequest(http.MethodGet, "/v1/internal/threads/"+thread.ID+"/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBridgeHandler_GetThreadSummary_NotFound(t *testing.T) {
	handler := NewBridgeHandler(NewService(testDB(t), NewMockProvider(), nil), "")
	r := chi.NewRouter()
	r.Get("/v1/internal/threads/{id}/summary", handler.GetThreadSummary)

	req := httptest.NewRequest(http.MethodGet, "/v1/internal/threads/nonexistent/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Phone handler tests ---

func TestPhoneHandler_ListNumbers(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createAdminMembership(t, db, org.ID, "admin-user")
	provider := NewMockProvider()
	handler := NewPhoneHandler(provider, db)

	r := chi.NewRouter()
	r.Get("/v1/orgs/{org}/channels/voice/numbers", handler.ListNumbers)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/channels/voice/numbers", nil)
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: "admin-user"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPhoneHandler_ListNumbers_Forbidden(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	handler := NewPhoneHandler(NewMockProvider(), db)

	r := chi.NewRouter()
	r.Get("/v1/orgs/{org}/channels/voice/numbers", handler.ListNumbers)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/channels/voice/numbers", nil)
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: "non-admin"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPhoneHandler_SearchNumbers(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createAdminMembership(t, db, org.ID, "admin-user")
	handler := NewPhoneHandler(NewMockProvider(), db)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/channels/voice/numbers/search", handler.SearchNumbers)

	body, _ := json.Marshal(SearchNumbersRequest{AreaCode: "415"})
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/channels/voice/numbers/search", bytes.NewReader(body))
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: "admin-user"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPhoneHandler_SearchNumbers_MissingAreaCode(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createAdminMembership(t, db, org.ID, "admin-user")
	handler := NewPhoneHandler(NewMockProvider(), db)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/channels/voice/numbers/search", handler.SearchNumbers)

	body, _ := json.Marshal(SearchNumbersRequest{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/channels/voice/numbers/search", bytes.NewReader(body))
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: "admin-user"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPhoneHandler_PurchaseNumber(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createAdminMembership(t, db, org.ID, "admin-user")
	handler := NewPhoneHandler(NewMockProvider(), db)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/channels/voice/numbers/purchase", handler.PurchaseNumber)

	body, _ := json.Marshal(PurchaseNumberRequest{NumberID: "num-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/channels/voice/numbers/purchase", bytes.NewReader(body))
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: "admin-user"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPhoneHandler_PurchaseNumber_MissingID(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createAdminMembership(t, db, org.ID, "admin-user")
	handler := NewPhoneHandler(NewMockProvider(), db)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/channels/voice/numbers/purchase", handler.PurchaseNumber)

	body, _ := json.Marshal(PurchaseNumberRequest{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/channels/voice/numbers/purchase", bytes.NewReader(body))
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: "admin-user"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPhoneHandler_Unauthenticated(t *testing.T) {
	handler := NewPhoneHandler(NewMockProvider(), testDB(t))
	r := chi.NewRouter()
	r.Get("/v1/orgs/{org}/channels/voice/numbers", handler.ListNumbers)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/org-1/channels/voice/numbers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Service helper tests ---

func TestService_GetThreadSummary(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	createTestSpaceAndBoard(t, db, org.ID)
	svc := NewService(db, NewMockProvider(), nil)

	roomMeta, _ := json.Marshal(RoomMetadata{OrgID: org.ID})
	roomEvt := WebhookEvent{Event: "room_started"}
	roomEvt.Room.Name = "room-sum"
	roomEvt.Room.Metadata = string(roomMeta)
	require.NoError(t, svc.HandleWebhookEvent(context.Background(), roomEvt))

	var thread models.Thread
	require.NoError(t, db.Where("json_extract(metadata, '$.room_name') = ?", "room-sum").First(&thread).Error)

	summary, err := svc.GetThreadSummary(context.Background(), thread.ID)
	require.NoError(t, err)
	assert.Equal(t, thread.ID, summary["id"])
}

func TestService_GetThreadSummary_NotFound(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	_, err := svc.GetThreadSummary(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestService_LookupContact_Email(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	_, board := createTestSpaceAndBoard(t, db, org.ID)

	// Create a thread with contact_email metadata.
	thread := &models.Thread{
		BoardID: board.ID, Title: "Contact Thread", Slug: "ct-1",
		Metadata: `{"contact_email":"test@example.com"}`, AuthorID: "system",
	}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, NewMockProvider(), nil)
	results, err := svc.LookupContact(context.Background(), "test@example.com", "")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestService_LookupContact_Phone(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	_, board := createTestSpaceAndBoard(t, db, org.ID)

	thread := &models.Thread{
		BoardID: board.ID, Title: "Phone Thread", Slug: "pt-1",
		Metadata: `{"phone":"+15551234567"}`, AuthorID: "system",
	}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, NewMockProvider(), nil)
	results, err := svc.LookupContact(context.Background(), "", "+15551234567")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestService_LookupContact_NoParams(t *testing.T) {
	svc := NewService(testDB(t), NewMockProvider(), nil)
	_, err := svc.LookupContact(context.Background(), "", "")
	assert.Error(t, err)
}

func TestParseRoomMetadata(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid", `{"org_id":"o-1","caller_id":"c-1"}`, false},
		{"invalid json", `{bad`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := parseRoomMetadata(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, meta)
			}
		})
	}
}

// --- Fuzz tests ---

func FuzzWebhookPayload(f *testing.F) {
	// Seed corpus with 50+ entries.
	f.Add(`{"event":"room_started","room":{"name":"r1","sid":"s1","metadata":"{\"org_id\":\"o1\"}"}}`)
	f.Add(`{"event":"participant_joined","participant":{"identity":"u1","name":"User"}}`)
	f.Add(`{"event":"room_finished","room":{"name":"r1"}}`)
	f.Add(`{"event":"egress_ended","egress_info":{"egress_id":"e1","room_name":"r1"}}`)
	f.Add(`{"event":"unknown_event"}`)
	f.Add(`{}`)
	f.Add(`{"event":""}`)
	f.Add(`{"event":"room_started","room":{"name":"","sid":"","metadata":""}}`)
	f.Add(`{"event":"room_started","room":{"metadata":"{}"}}`)
	f.Add(`{"event":"room_started","room":{"metadata":"invalid json"}}`)
	f.Add(`{"event":"participant_joined","room":{"name":""},"participant":{"identity":""}}`)
	f.Add(`not json at all`)
	f.Add(`null`)
	f.Add(`[]`)
	f.Add(`""`)
	f.Add(`0`)
	f.Add(`true`)
	f.Add(`{"event":"room_started","room":{"name":"a","metadata":"{\"org_id\":\"\"}"}}`)
	f.Add(`{"event":"egress_ended"}`)
	f.Add(`{"event":"egress_ended","egress_info":null}`)
	f.Add(`{"event":"egress_ended","egress_info":{"egress_id":"","room_name":"","file_url":""}}`)
	f.Add(`{"event":"room_started","room":{"name":"` + string(make([]byte, 1000)) + `"}}`)
	f.Add(`{"event":"room_started","created_at":0}`)
	f.Add(`{"event":"room_started","created_at":-1}`)
	f.Add(`{"event":"room_started","created_at":9999999999999}`)
	f.Add(`{"event":"track_published"}`)
	f.Add(`{"event":"track_unpublished"}`)
	f.Add(`{"event":"participant_left"}`)
	f.Add(`{"event":"room_started","room":{"name":"x","metadata":"{\"org_id\":\"x\",\"caller_id\":\"y\"}"}}`)
	f.Add(`{"event":"participant_joined","participant":{"identity":"a\x00b"}}`)
	f.Add(`{"event":"room_started","room":{"metadata":"{\"org_id\":\"o\",\"phone\":\"+1555\"}"}}`)
	f.Add(`{"event":"room_finished","room":{"name":"x","sid":"y"}}`)
	f.Add(`{"event":"egress_ended","egress_info":{"duration":-1}}`)
	f.Add(`{"event":"egress_ended","egress_info":{"duration":999999}}`)
	f.Add(`{"event":"room_started","room":{"name":"r","sid":"s","metadata":"{\"org_id\":\"o\",\"thread_id\":\"t\"}"}}`)
	f.Add(`{`)
	f.Add(`}`)
	f.Add(`{"event":"\x00"}`)
	f.Add(`{"event":"room_started","room":{"name":"\n\t"}}`)
	f.Add(`{"event":"room_started","room":{"name":"abc","metadata":"{\"org_id\":123}"}}`)
	f.Add(`{"event":"room_started","room":{"name":"abc","metadata":"{\"org_id\":null}"}}`)
	f.Add(`{"event":"room_started","room":{"name":"abc","metadata":"{\"org_id\":true}"}}`)
	f.Add(`{"event":"room_started","room":{"name":"abc","metadata":"[1,2,3]"}}`)
	f.Add(`{"event":"participant_joined","room":{"name":"abc"},"participant":{"identity":"id","name":"name","sid":"sid","metadata":"{}"}}`)
	f.Add(`{"event":"room_started","room":{"name":"z","metadata":"{\"org_id\":\"o\",\"caller_id\":\"c\",\"phone\":\"+15551234567\"}"}}`)
	f.Add(`{"event":"egress_ended","egress_info":{"egress_id":"eid","room_name":"rn","status":"complete","file_url":"https://example.com/file.ogg","duration":120}}`)
	f.Add(`{"event":"room_started","room":{"name":"` + "a" + `","sid":"s1","metadata":"{\"org_id\":\"o1\"}"}, "created_at": 1234567890}`)
	f.Add(`{"event":123}`)
	f.Add(`{"event":null}`)
	f.Add(`{"event":true}`)
	f.Add(`{"event":[]}`)
	f.Add(`{"event":"room_started","extra_field":"ignored"}`)

	f.Fuzz(func(t *testing.T, payload string) {
		var evt WebhookEvent
		_ = json.Unmarshal([]byte(payload), &evt)
		// Must not panic.
	})
}

func FuzzTranscriptEvent(f *testing.F) {
	// 50+ seed corpus entries.
	f.Add("agent", "Hello!", 0.0, 1.0, true)
	f.Add("caller", "Hi there", 1.0, 2.0, true)
	f.Add("agent", "How can I help?", 2.0, 4.0, true)
	f.Add("caller", "", 0.0, 0.0, true)
	f.Add("agent", "partial", 0.5, 0.6, false)
	f.Add("", "text", 0.0, 1.0, true)
	f.Add("agent", "a", -1.0, -0.5, true)
	f.Add("caller", "b", 999999.0, 999999.5, true)
	f.Add("agent", string(make([]byte, 500)), 0.0, 1.0, true)
	f.Add("x", "y", 0.0, 0.0, true)
	f.Add("agent", "test\nline", 0.0, 1.0, true)
	f.Add("caller", "test\ttab", 0.0, 1.0, true)
	f.Add("agent", "test\x00null", 0.0, 1.0, true)
	f.Add("caller", "unicode: 你好", 0.0, 1.0, true)
	f.Add("agent", "emoji: 👋", 0.0, 1.0, true)
	f.Add("agent", "Hello!", 0.0, 1.0, false)
	f.Add("caller", "long text "+string(make([]byte, 1000)), 0.0, 1.0, true)
	f.Add("agent", "special <>&\"'", 0.0, 1.0, true)
	f.Add("caller", "  spaces  ", 0.0, 1.0, true)
	f.Add("agent", "end.", 100.0, 200.0, true)
	f.Add("caller", "start.", 0.001, 0.002, true)
	f.Add("agent", "overlap", 1.0, 0.5, true)
	f.Add("caller", "same time", 5.0, 5.0, true)
	f.Add("agent", "negative end", 1.0, -1.0, true)
	f.Add("caller", "max float", 1.7976931348623157e+308, 1.7976931348623157e+308, true)
	f.Add("agent", "tiny", 5e-324, 5e-324, true)
	f.Add("caller", "inf-like", 1e+300, 1e+300, true)
	f.Add("system", "text", 0.0, 1.0, true)
	f.Add("bot", "text", 0.0, 1.0, true)
	f.Add("human", "text", 0.0, 1.0, true)
	f.Add("agent", "text", 0.0, 1.0, true)
	f.Add("caller", "text", 0.0, 1.0, true)
	f.Add("agent", "text", 0.0, 1.0, true)
	f.Add("caller", "text", 0.0, 1.0, true)
	f.Add("agent", "text1", 1.0, 2.0, true)
	f.Add("caller", "text2", 2.0, 3.0, true)
	f.Add("agent", "text3", 3.0, 4.0, true)
	f.Add("caller", "text4", 4.0, 5.0, true)
	f.Add("agent", "text5", 5.0, 6.0, true)
	f.Add("caller", "text6", 6.0, 7.0, true)
	f.Add("agent", "text7", 7.0, 8.0, true)
	f.Add("caller", "text8", 8.0, 9.0, true)
	f.Add("agent", "text9", 9.0, 10.0, true)
	f.Add("caller", "text10", 10.0, 11.0, true)
	f.Add("agent", "text11", 11.0, 12.0, false)
	f.Add("caller", "text12", 12.0, 13.0, false)
	f.Add("agent", "mixed\r\n\ttabs", 0.0, 1.0, true)
	f.Add("caller", "\x1b[31mcolored\x1b[0m", 0.0, 1.0, true)
	f.Add("agent", "path/traversal/../test", 0.0, 1.0, true)
	f.Add("caller", `json"escape`, 0.0, 1.0, true)

	f.Fuzz(func(t *testing.T, speaker, text string, startTime, endTime float64, isFinal bool) {
		events := []TranscriptEvent{{
			Speaker:   speaker,
			Text:      text,
			StartTime: startTime,
			EndTime:   endTime,
			IsFinal:   isFinal,
		}}
		// Must not panic.
		transcript := CompileTranscript("fuzz-room", events)
		_ = transcript.FullText
	})
}
