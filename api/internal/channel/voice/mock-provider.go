package voice

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockProvider is a test implementation of LiveKitProvider that stores
// state in memory for assertions.
type MockProvider struct {
	mu           sync.Mutex
	rooms        map[string]*Room
	participants map[string][]Participant
	rules        map[string]*SIPDispatchRule
	numbers      map[string]*PhoneNumber
	recordings   map[string]*RecordingInfo

	// Error injection fields for testing error paths.
	CreateRoomErr            error
	ListParticipantsErr      error
	RemoveParticipantErr     error
	CreateSIPDispatchRuleErr error
	DeleteSIPDispatchRuleErr error
	SearchPhoneNumbersErr    error
	PurchasePhoneNumberErr   error
	ListPhoneNumbersErr      error
	StartRecordingErr        error
	StopRecordingErr         error
}

// NewMockProvider creates a new MockProvider with empty state.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		rooms:        make(map[string]*Room),
		participants: make(map[string][]Participant),
		rules:        make(map[string]*SIPDispatchRule),
		numbers:      make(map[string]*PhoneNumber),
		recordings:   make(map[string]*RecordingInfo),
	}
}

// CreateRoom creates a mock room.
func (m *MockProvider) CreateRoom(_ context.Context, name, metadata string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateRoomErr != nil {
		return nil, m.CreateRoomErr
	}
	room := &Room{
		ID:              uuid.New().String(),
		Name:            name,
		NumParticipants: 0,
		CreatedAt:       time.Now(),
	}
	m.rooms[name] = room
	return room, nil
}

// ListParticipants returns mock participants for a room.
func (m *MockProvider) ListParticipants(_ context.Context, roomName string) ([]Participant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ListParticipantsErr != nil {
		return nil, m.ListParticipantsErr
	}
	return m.participants[roomName], nil
}

// RemoveParticipant removes a mock participant from a room.
func (m *MockProvider) RemoveParticipant(_ context.Context, roomName, identity string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.RemoveParticipantErr != nil {
		return m.RemoveParticipantErr
	}
	parts := m.participants[roomName]
	for i, p := range parts {
		if p.Identity == identity {
			m.participants[roomName] = append(parts[:i], parts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("participant %s not found in room %s", identity, roomName)
}

// AddMockParticipant adds a participant to a room's mock state for testing.
func (m *MockProvider) AddMockParticipant(roomName string, p Participant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.participants[roomName] = append(m.participants[roomName], p)
	if room, ok := m.rooms[roomName]; ok {
		room.NumParticipants = len(m.participants[roomName])
	}
}

// CreateSIPDispatchRule creates a mock dispatch rule.
func (m *MockProvider) CreateSIPDispatchRule(_ context.Context, rule SIPDispatchRule) (*SIPDispatchRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateSIPDispatchRuleErr != nil {
		return nil, m.CreateSIPDispatchRuleErr
	}
	rule.ID = uuid.New().String()
	m.rules[rule.ID] = &rule
	return &rule, nil
}

// DeleteSIPDispatchRule removes a mock dispatch rule.
func (m *MockProvider) DeleteSIPDispatchRule(_ context.Context, ruleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteSIPDispatchRuleErr != nil {
		return m.DeleteSIPDispatchRuleErr
	}
	if _, ok := m.rules[ruleID]; !ok {
		return fmt.Errorf("dispatch rule %s not found", ruleID)
	}
	delete(m.rules, ruleID)
	return nil
}

// SearchPhoneNumbers returns mock available phone numbers for an area code.
func (m *MockProvider) SearchPhoneNumbers(_ context.Context, areaCode string) ([]PhoneNumber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SearchPhoneNumbersErr != nil {
		return nil, m.SearchPhoneNumbersErr
	}
	// Return synthetic results based on area code.
	return []PhoneNumber{
		{ID: uuid.New().String(), Number: "+1" + areaCode + "5550001", AreaCode: areaCode, Country: "US"},
		{ID: uuid.New().String(), Number: "+1" + areaCode + "5550002", AreaCode: areaCode, Country: "US"},
	}, nil
}

// PurchasePhoneNumber marks a mock phone number as purchased.
func (m *MockProvider) PurchasePhoneNumber(_ context.Context, numberID string) (*PhoneNumber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.PurchasePhoneNumberErr != nil {
		return nil, m.PurchasePhoneNumberErr
	}
	num := &PhoneNumber{
		ID:          numberID,
		Number:      "+15555550099",
		AreaCode:    "555",
		Country:     "US",
		Provisioned: true,
	}
	m.numbers[numberID] = num
	return num, nil
}

// ListPhoneNumbers returns all mock provisioned numbers.
func (m *MockProvider) ListPhoneNumbers(_ context.Context) ([]PhoneNumber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ListPhoneNumbersErr != nil {
		return nil, m.ListPhoneNumbersErr
	}
	result := make([]PhoneNumber, 0, len(m.numbers))
	for _, n := range m.numbers {
		result = append(result, *n)
	}
	return result, nil
}

// StartRecording starts a mock recording for a room.
func (m *MockProvider) StartRecording(_ context.Context, roomName string) (*RecordingInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StartRecordingErr != nil {
		return nil, m.StartRecordingErr
	}
	rec := &RecordingInfo{
		RecordingID: uuid.New().String(),
		RoomName:    roomName,
		Status:      "recording",
	}
	m.recordings[rec.RecordingID] = rec
	return rec, nil
}

// StopRecording stops a mock recording.
func (m *MockProvider) StopRecording(_ context.Context, recordingID string) (*RecordingInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StopRecordingErr != nil {
		return nil, m.StopRecordingErr
	}
	rec, ok := m.recordings[recordingID]
	if !ok {
		return nil, fmt.Errorf("recording %s not found", recordingID)
	}
	rec.Status = "stopped"
	rec.Duration = 120 // Mock 2-minute call.
	return rec, nil
}

// GetRoom returns a room from mock state. Used for test assertions.
func (m *MockProvider) GetRoom(name string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rooms[name]
}

// GetRecording returns a recording from mock state. Used for test assertions.
func (m *MockProvider) GetRecording(id string) *RecordingInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.recordings[id]
}

// Ensure MockProvider implements LiveKitProvider at compile time.
var _ LiveKitProvider = (*MockProvider)(nil)
