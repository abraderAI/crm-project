package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClerkUser holds basic profile info fetched from the Clerk Backend API.
type ClerkUser struct {
	// Email is the user's primary email address.
	Email string
	// DisplayName is the user's full name, or username when no name is set.
	DisplayName string
}

// ClerkClient calls the Clerk Backend API to fetch user profiles.
// It is used as a fallback when JWT claims do not include identity info.
type ClerkClient struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

// NewClerkClient creates a ClerkClient for the given Clerk secret key.
// Returns nil when secretKey is empty so callers can check for nil before use.
func NewClerkClient(secretKey string) *ClerkClient {
	if secretKey == "" {
		return nil
	}
	return &ClerkClient{
		secretKey:  secretKey,
		baseURL:    "https://api.clerk.com",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// NewClerkClientForTest creates a ClerkClient with a custom base URL.
// This is intended for use in tests that need to point at a mock HTTP server
// instead of the real Clerk API.
func NewClerkClientForTest(secretKey, baseURL string) *ClerkClient {
	return &ClerkClient{
		secretKey:  secretKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// GetUser fetches the Clerk user profile for the given Clerk user ID.
// Returns an error if the API call fails or returns a non-200 status.
func (c *ClerkClient) GetUser(ctx context.Context, userID string) (*ClerkUser, error) {
	url := c.baseURL + "/v1/users/" + userID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building clerk request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("clerk API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clerk API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10)) // 64 KB limit
	if err != nil {
		return nil, fmt.Errorf("reading clerk response: %w", err)
	}

	// Parse only the fields we need from the Clerk user object.
	var raw struct {
		EmailAddresses []struct {
			EmailAddress string `json:"email_address"`
		} `json:"email_addresses"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing clerk response: %w", err)
	}

	user := &ClerkUser{}
	if len(raw.EmailAddresses) > 0 {
		user.Email = raw.EmailAddresses[0].EmailAddress
	}
	// Prefer full name; fall back to username.
	user.DisplayName = buildDisplayName(raw.FirstName, raw.LastName)
	if user.DisplayName == "" {
		user.DisplayName = strings.TrimSpace(raw.Username)
	}

	return user, nil
}
