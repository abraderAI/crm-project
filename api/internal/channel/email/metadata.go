package email

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MessageMetadata holds email-specific metadata stored in the message's Metadata JSONB field.
type MessageMetadata struct {
	ChannelType string   `json:"channel_type"`
	MessageID   string   `json:"message_id,omitempty"`
	InReplyTo   string   `json:"in_reply_to,omitempty"`
	References  []string `json:"references,omitempty"`
	From        string   `json:"from"`
	To          []string `json:"to,omitempty"`
	CC          []string `json:"cc,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	EventID     string   `json:"event_id,omitempty"`
	SenderID    string   `json:"sender,omitempty"`
	ExternalID  string   `json:"external_id,omitempty"`
}

// BuildMessageMetadata creates the metadata JSON string for a message created from an email.
func BuildMessageMetadata(parsed *ParsedEmail, eventID string) string {
	meta := MessageMetadata{
		ChannelType: "email",
		MessageID:   parsed.MessageID,
		InReplyTo:   parsed.InReplyTo,
		References:  parsed.References,
		From:        parsed.From,
		To:          parsed.To,
		CC:          parsed.CC,
		Subject:     parsed.Subject,
		EventID:     eventID,
		SenderID:    parsed.From,
		ExternalID:  parsed.MessageID,
	}
	b, err := json.Marshal(meta)
	if err != nil {
		// Fallback to minimal JSON.
		return fmt.Sprintf(`{"channel_type":"email","from":%q}`, parsed.From)
	}
	return string(b)
}

// ThreadMetadata holds email-specific metadata stored in the thread's Metadata JSONB field.
type ThreadMetadata struct {
	Source       string   `json:"source"`
	ContactEmail string   `json:"contact_email"`
	EmailAddress string   `json:"email_address"`
	ChannelType  string   `json:"channel_type"`
	MessageID    string   `json:"message_id,omitempty"`
	MessageIDs   []string `json:"message_ids,omitempty"`
	ExternalID   string   `json:"external_id,omitempty"`
}

// BuildThreadMetadata creates the metadata JSON string for a new lead thread from an email.
func BuildThreadMetadata(parsed *ParsedEmail) string {
	meta := ThreadMetadata{
		Source:       "inbound_email",
		ContactEmail: parsed.From,
		EmailAddress: parsed.From,
		ChannelType:  "email",
		MessageID:    parsed.MessageID,
		ExternalID:   parsed.MessageID,
	}
	if parsed.MessageID != "" {
		meta.MessageIDs = []string{parsed.MessageID}
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Sprintf(`{"source":"inbound_email","contact_email":%q,"channel_type":"email"}`, parsed.From)
	}
	return string(b)
}

// UpdateThreadMetadataWithMessageID merges a new message ID into existing thread metadata JSON.
// Returns the updated metadata JSON string.
func UpdateThreadMetadataWithMessageID(existingMeta, newMessageID string) (string, error) {
	if newMessageID == "" {
		return existingMeta, nil
	}

	var meta map[string]any
	if existingMeta == "" || existingMeta == "{}" {
		meta = make(map[string]any)
	} else {
		if err := json.Unmarshal([]byte(existingMeta), &meta); err != nil {
			meta = make(map[string]any)
		}
	}

	// Get existing message_ids array.
	var messageIDs []string
	if existing, ok := meta["message_ids"]; ok {
		switch v := existing.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					messageIDs = append(messageIDs, s)
				}
			}
		case []string:
			messageIDs = v
		}
	}

	// Check for duplicate.
	for _, id := range messageIDs {
		if id == newMessageID {
			return existingMeta, nil
		}
	}

	messageIDs = append(messageIDs, newMessageID)
	meta["message_ids"] = messageIDs

	b, err := json.Marshal(meta)
	if err != nil {
		return existingMeta, fmt.Errorf("marshaling metadata: %w", err)
	}
	return string(b), nil
}

// ExtractMessageIDs extracts the message_ids array from thread metadata JSON.
func ExtractMessageIDs(metadataJSON string) []string {
	if metadataJSON == "" || metadataJSON == "{}" {
		return nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil
	}
	existing, ok := meta["message_ids"]
	if !ok {
		return nil
	}
	arr, ok := existing.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// ExtractEmailAddress extracts the email_address field from thread metadata JSON.
func ExtractEmailAddressFromMeta(metadataJSON string) string {
	if metadataJSON == "" || metadataJSON == "{}" {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return ""
	}
	if addr, ok := meta["email_address"].(string); ok {
		return addr
	}
	if addr, ok := meta["contact_email"].(string); ok {
		return addr
	}
	return ""
}

// SanitizeEmailHeader removes control characters and normalizes whitespace in a header value.
func SanitizeEmailHeader(value string) string {
	// Replace control characters with spaces.
	var sb strings.Builder
	for _, r := range value {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	// Collapse whitespace.
	result := strings.Join(strings.Fields(sb.String()), " ")
	return strings.TrimSpace(result)
}
