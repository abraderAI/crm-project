// Package channel provides the unified IO Channel Gateway for normalizing inbound
// events from email, voice, and chat channels into CRM threads and messages.
package channel

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// InboundEvent is a normalized representation of an inbound event from any IO channel.
// All channel adapters (email, voice, chat) convert their raw events to this format
// before passing them to the ChannelGateway.
type InboundEvent struct {
	// ID is a UUIDv7 assigned at event receipt time.
	ID string
	// ChannelType is the source channel (email, voice, or chat).
	ChannelType models.ChannelType
	// OrgID is the organization this event belongs to.
	OrgID string
	// ExternalID is the channel-specific message identifier (e.g. RFC 5322 Message-ID).
	ExternalID string
	// SenderIdentifier is an email address, phone number, or chat session token.
	SenderIdentifier string
	// Subject is the email subject or call/chat title.
	Subject string
	// Body is the message body (plain text).
	Body string
	// Metadata holds arbitrary channel-specific attributes as a JSON string.
	Metadata string
	// Attachments lists any file attachments associated with the event.
	Attachments []AttachmentRef
	// ReceivedAt is when the event was received by the channel adapter.
	ReceivedAt time.Time
}

// AttachmentRef describes a file attachment associated with an InboundEvent.
type AttachmentRef struct {
	// Filename is the original file name.
	Filename string `json:"filename"`
	// ContentType is the MIME type of the attachment.
	ContentType string `json:"content_type"`
	// Size is the file size in bytes.
	Size int64 `json:"size"`
	// URL is the source URL before upload to StorageProvider (may be empty after upload).
	URL string `json:"url,omitempty"`
}

// Normalizer converts channel-specific raw event bytes into a unified InboundEvent.
// Each channel adapter (email, voice, chat) implements this interface.
type Normalizer interface {
	// Normalize converts raw event bytes received for orgID into an InboundEvent.
	Normalize(orgID string, raw []byte) (*InboundEvent, error)
}

// HealthStatus represents the operational health of a single channel for an org.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the channel is operating normally.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates elevated error rate but still processing.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusDown indicates the channel is disabled or experiencing high failure rates.
	HealthStatusDown HealthStatus = "down"
)

// ChannelHealth summarizes the operational health of a single channel for an org.
type ChannelHealth struct {
	ChannelType models.ChannelType `json:"channel_type"`
	Status      HealthStatus       `json:"status"`
	Enabled     bool               `json:"enabled"`
	// LastEventAt is the timestamp of the most recent event (success or failure).
	LastEventAt *time.Time `json:"last_event_at,omitempty"`
	// ErrorCount is the number of failed/retrying events in the last 24 hours.
	ErrorCount int64 `json:"error_count_24h"`
}

// --- Channel-specific config validation types ---

// EmailConfig holds validated settings for the email (Gmail IMAP) channel.
type EmailConfig struct {
	IMAPHost string `json:"imap_host"`
	IMAPPort int    `json:"imap_port"`
	Username string `json:"username"`
	// Password is write-only; masked as "[REDACTED]" in GET responses.
	Password string `json:"password,omitempty"`
	// OAuthRefreshToken is write-only; masked as "[REDACTED]" in GET responses.
	OAuthRefreshToken string `json:"oauth_refresh_token,omitempty"`
	Mailbox           string `json:"mailbox"`
}

// Validate checks that required EmailConfig fields are present.
func (c *EmailConfig) Validate() error {
	if c.IMAPHost == "" {
		return fmt.Errorf("imap_host is required")
	}
	if c.IMAPPort <= 0 {
		return fmt.Errorf("imap_port must be a positive integer")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

// MaskSecrets returns a copy of the config with sensitive fields redacted.
func (c EmailConfig) MaskSecrets() EmailConfig {
	if c.Password != "" {
		c.Password = "[REDACTED]"
	}
	if c.OAuthRefreshToken != "" {
		c.OAuthRefreshToken = "[REDACTED]"
	}
	return c
}

// VoiceConfig holds validated settings for the voice (LiveKit) channel.
type VoiceConfig struct {
	LiveKitAPIKey string `json:"livekit_api_key"`
	// LiveKitAPISecret is write-only; masked as "[REDACTED]" in GET responses.
	LiveKitAPISecret  string   `json:"livekit_api_secret,omitempty"`
	LiveKitProjectURL string   `json:"livekit_project_url"`
	PhoneNumberIDs    []string `json:"phone_number_ids,omitempty"`
	RecordingEnabled  bool     `json:"recording_enabled"`
	EscalationPhone   string   `json:"escalation_phone,omitempty"`
	AgentDeploymentID string   `json:"agent_deployment_id,omitempty"`
	DefaultSTTModel   string   `json:"default_stt_model,omitempty"`
	DefaultTTSModel   string   `json:"default_tts_model,omitempty"`
	SystemPrompt      string   `json:"system_prompt,omitempty"`
}

// Validate checks that required VoiceConfig fields are present.
func (c *VoiceConfig) Validate() error {
	if c.LiveKitAPIKey == "" {
		return fmt.Errorf("livekit_api_key is required")
	}
	if c.LiveKitProjectURL == "" {
		return fmt.Errorf("livekit_project_url is required")
	}
	return nil
}

// MaskSecrets returns a copy of the config with sensitive fields redacted.
func (c VoiceConfig) MaskSecrets() VoiceConfig {
	if c.LiveKitAPISecret != "" {
		c.LiveKitAPISecret = "[REDACTED]"
	}
	return c
}

// WidgetTheme describes the visual appearance of the embeddable chat widget.
type WidgetTheme struct {
	// PrimaryColor is the main brand color (hex, e.g. "#3B82F6").
	PrimaryColor string `json:"primary_color,omitempty"`
	// LogoURL is an optional URL to the organization logo.
	LogoURL string `json:"logo_url,omitempty"`
	// Greeting is the initial greeting message displayed to visitors.
	Greeting string `json:"greeting,omitempty"`
}

// OperatingHours defines when the chat channel is active.
type OperatingHours struct {
	// Enabled controls whether operating hours restrictions are enforced.
	Enabled bool `json:"enabled"`
	// Timezone is the IANA timezone name (e.g. "America/New_York").
	Timezone string `json:"timezone,omitempty"`
	// Start is the daily start time in HH:MM 24-hour format.
	Start string `json:"start,omitempty"`
	// End is the daily end time in HH:MM 24-hour format.
	End string `json:"end,omitempty"`
	// Days lists active weekday numbers (0=Sunday, 6=Saturday).
	Days []int `json:"days,omitempty"`
}

// ChatConfig holds validated settings for the embeddable web chat channel.
type ChatConfig struct {
	// EmbedKey is the public widget authentication token (auto-generated UUID).
	EmbedKey string `json:"embed_key"`
	// WidgetTheme describes widget appearance.
	WidgetTheme WidgetTheme `json:"widget_theme,omitempty"`
	// AISystemPrompt is the system prompt sent to the LLM for this org's chat.
	AISystemPrompt string `json:"ai_system_prompt,omitempty"`
	// EscalationTimeoutSeconds is how long to wait for a human agent before AI resumes.
	EscalationTimeoutSeconds int `json:"escalation_timeout_seconds,omitempty"`
	// AllowedDomains restricts which domains may embed the widget.
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	// OperatingHours defines when the chat channel is active.
	OperatingHours OperatingHours `json:"operating_hours,omitempty"`
}

// Validate checks ChatConfig; all fields are optional (embed key is auto-generated).
func (c *ChatConfig) Validate() error {
	if c.EscalationTimeoutSeconds < 0 {
		return fmt.Errorf("escalation_timeout_seconds must be non-negative")
	}
	return nil
}

// MaskSecrets returns the config unchanged; chat has no write-only secrets.
func (c ChatConfig) MaskSecrets() ChatConfig {
	return c
}

// ValidateSettings validates a Settings JSON string against the channel-specific struct.
// Empty or "{}" settings are always valid.
func ValidateSettings(channelType models.ChannelType, settingsJSON string) error {
	if settingsJSON == "" || settingsJSON == "{}" {
		return nil
	}
	switch channelType {
	case models.ChannelTypeEmail:
		var cfg EmailConfig
		if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
			return fmt.Errorf("invalid email settings: %w", err)
		}
		if cfg.IMAPHost != "" {
			return cfg.Validate()
		}
	case models.ChannelTypeVoice:
		var cfg VoiceConfig
		if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
			return fmt.Errorf("invalid voice settings: %w", err)
		}
		if cfg.LiveKitAPIKey != "" {
			return cfg.Validate()
		}
	case models.ChannelTypeChat:
		var cfg ChatConfig
		if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
			return fmt.Errorf("invalid chat settings: %w", err)
		}
	default:
		return fmt.Errorf("unknown channel type: %s", channelType)
	}
	return nil
}

// MaskSettingsSecrets returns the Settings JSON string with sensitive fields redacted
// according to the channel type. Returns the original string on any decode error.
func MaskSettingsSecrets(channelType models.ChannelType, settingsJSON string) string {
	if settingsJSON == "" || settingsJSON == "{}" {
		return settingsJSON
	}
	switch channelType {
	case models.ChannelTypeEmail:
		var cfg EmailConfig
		if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
			return settingsJSON
		}
		b, err := json.Marshal(cfg.MaskSecrets())
		if err != nil {
			return settingsJSON
		}
		return string(b)
	case models.ChannelTypeVoice:
		var cfg VoiceConfig
		if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
			return settingsJSON
		}
		b, err := json.Marshal(cfg.MaskSecrets())
		if err != nil {
			return settingsJSON
		}
		return string(b)
	case models.ChannelTypeChat:
		// Chat config contains no write-only secrets.
		return settingsJSON
	default:
		return settingsJSON
	}
}
