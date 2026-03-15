package email

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// OAuthService generates XOAUTH2 tokens for IMAP authentication.
type OAuthService interface {
	// GetXOAUTH2Token returns a valid XOAUTH2 token string for the given email.
	// The token is auto-refreshed before expiry.
	GetXOAUTH2Token(email string) (string, error)
}

// OAuthCredentials holds Google service account credentials loaded from Settings JSONB.
type OAuthCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"oauth_refresh_token"`
	TokenURL     string `json:"token_url,omitempty"`
}

// Validate checks that required credential fields are present.
func (c *OAuthCredentials) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required for OAuth")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required for OAuth")
	}
	if c.RefreshToken == "" {
		return fmt.Errorf("oauth_refresh_token is required for OAuth")
	}
	return nil
}

// cachedToken holds a token with its expiry time.
type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// GoogleOAuthService implements OAuthService for Google XOAUTH2.
type GoogleOAuthService struct {
	mu          sync.Mutex
	credentials OAuthCredentials
	token       *cachedToken
	// refreshFunc is injectable for tests; in production, it performs HTTP token refresh.
	refreshFunc func(creds OAuthCredentials) (string, time.Duration, error)
	// nowFunc is injectable for tests; defaults to time.Now.
	nowFunc func() time.Time
}

// NewGoogleOAuthService creates a new Google OAuth service from credentials.
func NewGoogleOAuthService(creds OAuthCredentials) (*GoogleOAuthService, error) {
	if err := creds.Validate(); err != nil {
		return nil, err
	}
	return &GoogleOAuthService{
		credentials: creds,
		refreshFunc: defaultRefreshFunc,
		nowFunc:     time.Now,
	}, nil
}

// newGoogleOAuthServiceForTest creates a Google OAuth service with injectable functions for testing.
func newGoogleOAuthServiceForTest(creds OAuthCredentials, refreshFn func(OAuthCredentials) (string, time.Duration, error), nowFn func() time.Time) *GoogleOAuthService {
	return &GoogleOAuthService{
		credentials: creds,
		refreshFunc: refreshFn,
		nowFunc:     nowFn,
	}
}

// GetXOAUTH2Token returns a valid XOAUTH2 token for the given email address.
// Tokens are cached and automatically refreshed 60 seconds before expiry.
func (s *GoogleOAuthService) GetXOAUTH2Token(email string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return cached token if still valid (with 60s buffer).
	if s.token != nil && s.nowFunc().Add(60*time.Second).Before(s.token.expiresAt) {
		return buildXOAUTH2String(email, s.token.accessToken), nil
	}

	// Refresh the token.
	accessToken, ttl, err := s.refreshFunc(s.credentials)
	if err != nil {
		return "", fmt.Errorf("refreshing OAuth token: %w", err)
	}

	s.token = &cachedToken{
		accessToken: accessToken,
		expiresAt:   s.nowFunc().Add(ttl),
	}

	return buildXOAUTH2String(email, accessToken), nil
}

// buildXOAUTH2String constructs the XOAUTH2 SASL string per RFC 7628.
// Format: "user=" + user + "\x01auth=Bearer " + accessToken + "\x01\x01"
func buildXOAUTH2String(email, accessToken string) string {
	authStr := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", email, accessToken)
	return base64.StdEncoding.EncodeToString([]byte(authStr))
}

// defaultRefreshFunc is a stub for the real HTTP-based token refresh.
// In production, this would call Google's token endpoint.
// For unit tests, this is replaced with a mock.
func defaultRefreshFunc(_ OAuthCredentials) (string, time.Duration, error) {
	return "", 0, fmt.Errorf("OAuth token refresh not implemented: configure integration test or provide mock")
}

// ParseOAuthCredentials extracts OAuth credentials from a Settings JSON string.
func ParseOAuthCredentials(settingsJSON string) (*OAuthCredentials, error) {
	if settingsJSON == "" || settingsJSON == "{}" {
		return nil, fmt.Errorf("empty settings")
	}
	var creds OAuthCredentials
	if err := json.Unmarshal([]byte(settingsJSON), &creds); err != nil {
		return nil, fmt.Errorf("parsing OAuth credentials: %w", err)
	}
	return &creds, nil
}

// MockOAuthService is a test double for OAuthService.
type MockOAuthService struct {
	mu           sync.Mutex
	Token        string
	Err          error
	GetCallCount int
}

// GetXOAUTH2Token returns the configured mock token or error.
func (m *MockOAuthService) GetXOAUTH2Token(_ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetCallCount++
	return m.Token, m.Err
}
