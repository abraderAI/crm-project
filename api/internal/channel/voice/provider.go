// Package voice provides LiveKit-based voice channel integration for
// room management, SIP dispatch rules, phone number provisioning,
// call recording, and transcript handling.
package voice

import (
	"context"
	"time"
)

// Room represents a LiveKit room for a voice call.
type Room struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	OrgID           string    `json:"org_id"`
	NumParticipants int       `json:"num_participants"`
	CreatedAt       time.Time `json:"created_at"`
}

// Participant represents a participant in a LiveKit room.
type Participant struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
	// State is "joining", "joined", "active", or "disconnected".
	State    string    `json:"state"`
	JoinedAt time.Time `json:"joined_at"`
}

// SIPDispatchRule maps an inbound SIP trunk to a LiveKit room.
type SIPDispatchRule struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	TrunkID         string `json:"trunk_id"`
	PhoneNumberID   string `json:"phone_number_id"`
	RoomPrefix      string `json:"room_prefix"`
	AgentDeployment string `json:"agent_deployment,omitempty"`
}

// PhoneNumber represents a phone number available for purchase or already owned.
type PhoneNumber struct {
	ID          string `json:"id"`
	Number      string `json:"number"`
	AreaCode    string `json:"area_code"`
	Country     string `json:"country"`
	Provisioned bool   `json:"provisioned"`
}

// RecordingInfo holds metadata about a room-level recording.
type RecordingInfo struct {
	RecordingID string `json:"recording_id"`
	RoomName    string `json:"room_name"`
	// Status is "recording", "stopped", or "exported".
	Status   string `json:"status"`
	Duration int    `json:"duration"`
	// FileURL is populated after export completes.
	FileURL string `json:"file_url,omitempty"`
}

// LiveKitProvider defines the interface for LiveKit voice operations.
// Implementations range from a mock (for testing) to a real LiveKit Cloud client.
type LiveKitProvider interface {
	// CreateRoom creates a new LiveKit room for a voice call.
	CreateRoom(ctx context.Context, name string, metadata string) (*Room, error)

	// ListParticipants returns all participants in a room.
	ListParticipants(ctx context.Context, roomName string) ([]Participant, error)

	// RemoveParticipant disconnects a participant from a room.
	RemoveParticipant(ctx context.Context, roomName, identity string) error

	// CreateSIPDispatchRule creates a dispatch rule for inbound SIP calls.
	CreateSIPDispatchRule(ctx context.Context, rule SIPDispatchRule) (*SIPDispatchRule, error)

	// DeleteSIPDispatchRule removes a dispatch rule by ID.
	DeleteSIPDispatchRule(ctx context.Context, ruleID string) error

	// SearchPhoneNumbers searches for available phone numbers by area code.
	SearchPhoneNumbers(ctx context.Context, areaCode string) ([]PhoneNumber, error)

	// PurchasePhoneNumber purchases and provisions a phone number.
	PurchasePhoneNumber(ctx context.Context, numberID string) (*PhoneNumber, error)

	// ListPhoneNumbers returns all provisioned phone numbers for the account.
	ListPhoneNumbers(ctx context.Context) ([]PhoneNumber, error)

	// StartRecording starts a composite audio recording for a room.
	StartRecording(ctx context.Context, roomName string) (*RecordingInfo, error)

	// StopRecording stops an active recording and returns the result.
	StopRecording(ctx context.Context, recordingID string) (*RecordingInfo, error)
}
