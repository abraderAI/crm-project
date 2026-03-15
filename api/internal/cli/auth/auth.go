// Package auth provides CLI authentication via OS keyring or config-based credential storage.
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// KeyringService is the service name used in the OS keyring.
	KeyringService = "deft-cli"
	// KeyringAccount is the account name used in the OS keyring.
	KeyringAccount = "credentials"
)

// Credentials holds the stored authentication credentials.
type Credentials struct {
	APIKey string `json:"api_key,omitempty"`
	Token  string `json:"token,omitempty"`
}

// IsEmpty returns true if no credentials are stored.
func (c *Credentials) IsEmpty() bool {
	return c.APIKey == "" && c.Token == ""
}

// AuthHeader returns the appropriate HTTP auth header value.
func (c *Credentials) AuthHeader() (string, string) {
	if c.Token != "" {
		return "Authorization", "Bearer " + c.Token
	}
	if c.APIKey != "" {
		return "X-API-Key", c.APIKey
	}
	return "", ""
}

// Keyring abstracts OS keyring operations for testability.
type Keyring interface {
	Set(service, account, password string) error
	Get(service, account string) (string, error)
	Delete(service, account string) error
}

// Store manages credential storage via a keyring backend.
type Store struct {
	keyring Keyring
}

// NewStore creates a new credential store.
func NewStore(kr Keyring) *Store {
	return &Store{keyring: kr}
}

// Save stores credentials in the keyring.
func (s *Store) Save(creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}
	if err := s.keyring.Set(KeyringService, KeyringAccount, string(data)); err != nil {
		return fmt.Errorf("saving to keyring: %w", err)
	}
	return nil
}

// Load retrieves credentials from the keyring.
// Returns empty credentials if none are stored.
func (s *Store) Load() (*Credentials, error) {
	data, err := s.keyring.Get(KeyringService, KeyringAccount)
	if err != nil {
		// Treat missing key as empty credentials.
		return &Credentials{}, nil
	}
	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return &Credentials{}, nil
	}
	return &creds, nil
}

// Clear removes stored credentials from the keyring.
func (s *Store) Clear() error {
	if err := s.keyring.Delete(KeyringService, KeyringAccount); err != nil {
		// Ignore errors if key doesn't exist.
		return nil
	}
	return nil
}

// ValidateCredentials checks credentials by calling GET /v1/ and verifying a 200 response.
func ValidateCredentials(apiURL string, creds *Credentials) error {
	if creds.IsEmpty() {
		return fmt.Errorf("no credentials provided")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL+"/v1/", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	headerKey, headerVal := creds.AuthHeader()
	if headerKey != "" {
		req.Header.Set(headerKey, headerVal)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (status %d)", resp.StatusCode)
	}
	return nil
}
