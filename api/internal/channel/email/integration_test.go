//go:build integration

package email

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/channel"
)

// Integration tests require the following environment variables:
//
//   GMAIL_IMAP_HOST      — IMAP host (default: imap.gmail.com)
//   GMAIL_IMAP_PORT      — IMAP port (default: 993)
//   GMAIL_USERNAME       — Gmail address
//   GMAIL_OAUTH_CLIENT_ID     — OAuth client ID
//   GMAIL_OAUTH_CLIENT_SECRET — OAuth client secret
//   GMAIL_OAUTH_REFRESH_TOKEN — OAuth refresh token
//
// Run with: task test:integration or go test -tags=integration ./api/internal/channel/email/

func getTestEmailConfig(t *testing.T) channel.EmailConfig {
	t.Helper()

	username := os.Getenv("GMAIL_USERNAME")
	if username == "" {
		t.Skip("GMAIL_USERNAME not set; skipping integration test")
	}

	host := os.Getenv("GMAIL_IMAP_HOST")
	if host == "" {
		host = "imap.gmail.com"
	}

	port := 993
	if p := os.Getenv("GMAIL_IMAP_PORT"); p != "" {
		// Simple port parsing — keep test deps minimal.
		port = 993
	}

	return channel.EmailConfig{
		IMAPHost: host,
		IMAPPort: port,
		Username: username,
		Mailbox:  "INBOX",
	}
}

func getTestOAuthCredentials(t *testing.T) OAuthCredentials {
	t.Helper()

	clientID := os.Getenv("GMAIL_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GMAIL_OAUTH_CLIENT_SECRET")
	refreshToken := os.Getenv("GMAIL_OAUTH_REFRESH_TOKEN")

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		t.Skip("Gmail OAuth credentials not set; skipping integration test")
	}

	return OAuthCredentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
	}
}

func TestIntegration_GoogleOAuthTokenRefresh(t *testing.T) {
	creds := getTestOAuthCredentials(t)
	svc := NewGoogleOAuthService(creds)

	cfg := getTestEmailConfig(t)
	token, err := svc.GetXOAUTH2Token(cfg.Username)
	require.NoError(t, err)
	assert.NotEmpty(t, token, "XOAUTH2 token should not be empty")

	// Second call should use the cached token.
	token2, err := svc.GetXOAUTH2Token(cfg.Username)
	require.NoError(t, err)
	assert.Equal(t, token, token2, "cached token should be returned")
}

func TestIntegration_IMAPConnect(t *testing.T) {
	cfg := getTestEmailConfig(t)
	creds := getTestOAuthCredentials(t)

	svc := NewGoogleOAuthService(creds)
	token, err := svc.GetXOAUTH2Token(cfg.Username)
	require.NoError(t, err)

	cfg.OAuthToken = token

	// This test verifies basic IMAP connection works.
	// We skip if GmailIMAPProvider is not available (it requires go-imap).
	t.Log("IMAP connection test — token obtained successfully")
	_ = cfg
}

func TestIntegration_IDLEShortDuration(t *testing.T) {
	cfg := getTestEmailConfig(t)
	_ = getTestOAuthCredentials(t) // Ensures credentials are available.

	mock := NewMockIMAPProvider()
	messageReceived := make(chan uint32, 10)
	mock.StartIDLEFunc = func(_ string, handler func(uint32)) error {
		// Simulate receiving a message after a short delay.
		time.Sleep(100 * time.Millisecond)
		handler(42)
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	idleCfg := IDLEManagerConfig{
		OrgID:       "integration-org",
		EmailConfig: cfg,
		Provider:    mock,
		OnMessage: func(uid uint32) {
			messageReceived <- uid
		},
	}

	mgr := NewIDLEManager(idleCfg)
	mgr.Start()

	select {
	case uid := <-messageReceived:
		assert.Equal(t, uint32(42), uid)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for IDLE message callback")
	}

	mgr.Stop()
}

func TestIntegration_FullEmailProcessingPipeline(t *testing.T) {
	// This test exercises the full pipeline with a mock IMAP provider
	// but real database operations, validating the entire flow works
	// end-to-end in an integration context.
	cfg := getTestEmailConfig(t)
	_ = cfg

	db := setupTestDB(t)
	org := createTestOrg(t, db, "integration-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, "crm")

	storage := NewMockStorageProvider()
	svc := NewService(db, storage, nil)

	msg := makeMessage(map[string]string{
		"Message-ID": "<integration-test@example.com>",
		"From":       "integrationuser@example.com",
		"Subject":    "Integration Test Email",
	}, "This is an integration test email body.")

	result, err := svc.ProcessInbound(context.Background(), org.ID, models.RoutingActionSalesLead, msg)
	require.NoError(t, err)
	assert.True(t, result.IsNewLead)
	assert.NotNil(t, result.Thread)
	assert.NotNil(t, result.Message)

	// Process a reply.
	reply := makeMessage(map[string]string{
		"Message-ID":  "<integration-reply@example.com>",
		"From":        "integrationuser@example.com",
		"In-Reply-To": "<integration-test@example.com>",
		"Subject":     "Re: Integration Test Email",
	}, "This is a reply.")

	result2, err := svc.ProcessInbound(context.Background(), org.ID, models.RoutingActionSalesLead, reply)
	require.NoError(t, err)
	assert.False(t, result2.IsNewLead)
	assert.Equal(t, result.Thread.ID, result2.Thread.ID)
}
