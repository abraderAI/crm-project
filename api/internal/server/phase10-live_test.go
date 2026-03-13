package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// liveValidClaims returns valid JWT claims for testing.
func liveValidClaims(issuerURL string) auth.JWTClaims {
	return auth.JWTClaims{
		Subject:   "user_test123",
		Issuer:    issuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}
}

// --- Phase 10 Live API Tests: Voice Stubs ---

// TestLive_Phase10_VoiceLogCall logs a call via the real API and verifies 201 response.
func TestLive_Phase10_VoiceLogCall(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create an org first.
	orgID := liveCreateOrg(t, env, "voice-org")

	// Log a call.
	callBody := map[string]interface{}{
		"caller_id": "user_voice_test",
		"direction": "inbound",
		"duration":  120,
		"status":    "completed",
	}
	bodyJSON, _ := json.Marshal(callBody)

	req, err := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/calls", bytes.NewReader(bodyJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var callLog models.CallLog
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&callLog))
	assert.NotEmpty(t, callLog.ID)
	assert.Equal(t, "user_voice_test", callLog.CallerID)
	assert.Equal(t, models.CallDirectionInbound, callLog.Direction)
}

// TestLive_Phase10_VoiceGetTranscript retrieves a stub transcript via real HTTP.
func TestLive_Phase10_VoiceGetTranscript(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID := liveCreateOrg(t, env, "transcript-org")
	callID := liveLogCall(t, env, orgID)

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+orgID+"/calls/"+callID, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, callID, body["call_id"])
	assert.Contains(t, body["transcript"], "[stub]")
}

// TestLive_Phase10_VoiceEscalate escalates a call and verifies the response.
func TestLive_Phase10_VoiceEscalate(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID := liveCreateOrg(t, env, "escalate-org")
	callID := liveLogCall(t, env, orgID)

	req, err := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/calls/"+callID+"/escalate", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "escalated", result["status"])
}

// TestLive_Phase10_VoiceCallNotFound returns 404 for nonexistent call.
func TestLive_Phase10_VoiceCallNotFound(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID := liveCreateOrg(t, env, "notfound-org")

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+orgID+"/calls/nonexistent-id", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Phase10_VoiceUnauthorized returns 401 without auth.
func TestLive_Phase10_VoiceUnauthorized(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp, err := http.Get(env.BaseURL + "/v1/orgs/any-org/calls/any-call")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// --- Phase 10 Live API Tests: GDPR ---

// TestLive_Phase10_GDPRExportUserData exports user data via real HTTP.
func TestLive_Phase10_GDPRExportUserData(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	adminToken := setupAdminToken(t, env)

	// Create some data for the user.
	orgID := liveCreateOrg(t, env, "gdpr-export-org")
	_ = orgID

	userID := "user_test123"
	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/users/"+userID+"/export", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), userID)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var export map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &export))
	assert.Equal(t, userID, export["user_id"])
}

// TestLive_Phase10_GDPRPurgeUser purges user data and verifies.
func TestLive_Phase10_GDPRPurgeUser(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	adminToken := setupAdminToken(t, env)

	// Create data for the user.
	orgID := liveCreateOrg(t, env, "gdpr-purge-org")

	// Create org membership in DB directly for the test user.
	env.DB.Create(&models.OrgMembership{
		OrgID:  orgID,
		UserID: "user_purge_target",
		Role:   models.RoleViewer,
	})

	// Create a notification.
	env.DB.Create(&models.Notification{
		UserID: "user_purge_target",
		Type:   "test",
		Title:  "Test",
	})

	// Purge the user.
	req, err := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/users/user_purge_target/purge", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "purged", result["status"])

	// Verify data is gone — export should return empty arrays.
	req2, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/users/user_purge_target/export", nil)
	require.NoError(t, err)
	req2.Header.Set("Authorization", "Bearer "+adminToken)

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	body, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)

	var export map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &export))
	memberships := export["memberships"].(map[string]interface{})
	orgs := memberships["orgs"].([]interface{})
	assert.Empty(t, orgs)
}

// TestLive_Phase10_GDPRPurgeOrg cascade-deletes an org.
func TestLive_Phase10_GDPRPurgeOrg(t *testing.T) {
	env := liveAuthServer(t)
	defer func() {
		env.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		env.Cleanup()
	}()
	adminToken := setupAdminToken(t, env)

	orgID := liveCreateOrg(t, env, "gdpr-org-purge")

	// Purge the org (admin handler requires confirm body).
	req, err := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/orgs/"+orgID+"/purge",
		strings.NewReader(`{"confirm":"purge `+orgID+`"}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify org is gone via GET.
	req2, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+orgID, nil)
	require.NoError(t, err)
	req2.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// TestLive_Phase10_GDPRUnauthorized returns 401 without auth.
func TestLive_Phase10_GDPRUnauthorized(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp, err := http.Get(env.BaseURL + "/v1/admin/users/any/export")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// --- Phase 10 Live API Tests: OTel Headers ---

// TestLive_Phase10_OTelTraceHeaders verifies that X-Request-ID is present
// after OTel middleware is added to the stack.
func TestLive_Phase10_OTelTraceHeaders(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp, err := http.Get(env.BaseURL + "/healthz")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

// --- Helper functions ---

// liveCreateOrg creates an org via the live API and returns its ID.
func liveCreateOrg(t *testing.T, env *liveAuthEnv, slug string) string {
	t.Helper()

	body := map[string]string{
		"name":        slug + " Org",
		"description": "Test org for " + slug,
	}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs", bytes.NewReader(bodyJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var org map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&org))
	return org["id"].(string)
}

// liveLogCall logs a call via the live API and returns the call ID.
func liveLogCall(t *testing.T, env *liveAuthEnv, orgID string) string {
	t.Helper()

	body := map[string]interface{}{
		"caller_id": "user_test123",
		"direction": "inbound",
		"duration":  60,
		"status":    "active",
	}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/calls", bytes.NewReader(bodyJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.SignToken(liveValidClaims(env.IssuerURL)))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var callLog map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&callLog))
	return callLog["id"].(string)
}
