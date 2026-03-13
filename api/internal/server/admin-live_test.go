package server

import (
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
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// --- Admin Phase A Live API Tests ---

// TestLive_Admin_Unauthorized verifies admin endpoints return 401 without auth.
func TestLive_Admin_Unauthorized(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/admin/users"},
		{http.MethodGet, "/v1/admin/orgs"},
		{http.MethodGet, "/v1/admin/audit-log"},
		{http.MethodGet, "/v1/admin/platform-admins"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, err := http.NewRequest(ep.method, env.BaseURL+ep.path, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
		})
	}
}

// TestLive_Admin_Forbidden_RegularUser verifies admin endpoints return 403 for non-admin users.
func TestLive_Admin_Forbidden_RegularUser(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "regular_user",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/admin/users"},
		{http.MethodGet, "/v1/admin/orgs"},
		{http.MethodGet, "/v1/admin/audit-log"},
		{http.MethodGet, "/v1/admin/platform-admins"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, err := http.NewRequest(ep.method, env.BaseURL+ep.path, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

			var problem apierrors.ProblemDetail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
			assert.Equal(t, 403, problem.Status)
		})
	}
}

// helper to create an admin user and get a signed token.
func setupAdminToken(t *testing.T, env *liveAuthEnv) string {
	t.Helper()
	adminUserID := "admin_" + t.Name()

	// Insert platform admin directly.
	admin := models.PlatformAdmin{
		UserID:    adminUserID,
		GrantedBy: "test",
		IsActive:  true,
	}
	require.NoError(t, env.DB.Create(&admin).Error)

	return env.SignToken(auth.JWTClaims{
		Subject:   adminUserID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
}

// TestLive_Admin_ListUsers verifies GET /v1/admin/users works for platform admins.
func TestLive_Admin_ListUsers(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	// Create some user shadows.
	now := time.Now()
	for _, u := range []models.UserShadow{
		{ClerkUserID: "user_a", Email: "alice@test.com", DisplayName: "Alice", LastSeenAt: now, SyncedAt: now},
		{ClerkUserID: "user_b", Email: "bob@test.com", DisplayName: "Bob", LastSeenAt: now, SyncedAt: now},
	} {
		require.NoError(t, env.DB.Create(&u).Error)
	}

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/users", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.GreaterOrEqual(t, len(body.Data), 2)
}

// TestLive_Admin_ListUsers_FilterEmail verifies email filter works.
func TestLive_Admin_ListUsers_FilterEmail(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	now := time.Now()
	require.NoError(t, env.DB.Create(&models.UserShadow{
		ClerkUserID: "u_alice", Email: "alice@filter.com", DisplayName: "Alice", LastSeenAt: now, SyncedAt: now,
	}).Error)
	require.NoError(t, env.DB.Create(&models.UserShadow{
		ClerkUserID: "u_bob", Email: "bob@filter.com", DisplayName: "Bob", LastSeenAt: now, SyncedAt: now,
	}).Error)

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/users?email=alice", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, 1, len(body.Data))
}

// TestLive_Admin_GetUser verifies GET /v1/admin/users/{user_id}.
func TestLive_Admin_GetUser(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	now := time.Now()
	require.NoError(t, env.DB.Create(&models.UserShadow{
		ClerkUserID: "detail_user", Email: "detail@test.com", DisplayName: "Detail User", LastSeenAt: now, SyncedAt: now,
	}).Error)

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/users/detail_user", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "detail@test.com", body["email"])
}

// TestLive_Admin_GetUser_NotFound verifies 404 for unknown user.
func TestLive_Admin_GetUser_NotFound(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/users/nonexistent", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Admin_BanUnbanFlow verifies the complete ban/unban flow.
func TestLive_Admin_BanUnbanFlow(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	adminToken := setupAdminToken(t, env)

	targetUserID := "ban_target"
	now := time.Now()
	require.NoError(t, env.DB.Create(&models.UserShadow{
		ClerkUserID: targetUserID, Email: "target@test.com", DisplayName: "Target", LastSeenAt: now, SyncedAt: now,
	}).Error)

	// Get a token for the target user.
	targetToken := env.SignToken(auth.JWTClaims{
		Subject:   targetUserID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// 1. Verify target can make requests.
	req, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+targetToken)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 2. Ban the user.
	banReq, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/admin/users/"+targetUserID+"/ban",
		strings.NewReader(`{"reason":"test ban"}`))
	banReq.Header.Set("Authorization", "Bearer "+adminToken)
	banReq.Header.Set("Content-Type", "application/json")
	banResp, err := http.DefaultClient.Do(banReq)
	require.NoError(t, err)
	defer func() { _ = banResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, banResp.StatusCode)

	var banBody map[string]string
	require.NoError(t, json.NewDecoder(banResp.Body).Decode(&banBody))
	assert.Equal(t, "banned", banBody["status"])

	// 3. Verify banned user gets 403.
	req2, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs", nil)
	req2.Header.Set("Authorization", "Bearer "+targetToken)
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	_ = resp2.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode)

	// 4. Unban the user.
	unbanReq, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/admin/users/"+targetUserID+"/unban",
		strings.NewReader("{}"))
	unbanReq.Header.Set("Authorization", "Bearer "+adminToken)
	unbanReq.Header.Set("Content-Type", "application/json")
	unbanResp, err := http.DefaultClient.Do(unbanReq)
	require.NoError(t, err)
	defer func() { _ = unbanResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, unbanResp.StatusCode)

	// 5. Verify access is restored.
	req3, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs", nil)
	req3.Header.Set("Authorization", "Bearer "+targetToken)
	resp3, err := http.DefaultClient.Do(req3)
	require.NoError(t, err)
	_ = resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
}

// TestLive_Admin_OrgSuspensionFlow verifies suspend/unsuspend flow.
func TestLive_Admin_OrgSuspensionFlow(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	adminToken := setupAdminToken(t, env)

	org := &models.Org{Name: "Suspend Test", Slug: "suspend-test", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	userToken := env.SignToken(auth.JWTClaims{
		Subject:   "org_user",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// 1. Verify writes work before suspension.
	writeReq, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+org.Slug+"/spaces",
		strings.NewReader(`{"name":"Test Space"}`))
	writeReq.Header.Set("Authorization", "Bearer "+userToken)
	writeReq.Header.Set("Content-Type", "application/json")
	writeResp, err := http.DefaultClient.Do(writeReq)
	require.NoError(t, err)
	_ = writeResp.Body.Close()
	// Space creation may fail for other reasons but should not be 503.
	assert.NotEqual(t, http.StatusServiceUnavailable, writeResp.StatusCode)

	// 2. Suspend the org.
	suspendReq, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/admin/orgs/"+org.Slug+"/suspend",
		strings.NewReader(`{"reason":"test suspension"}`))
	suspendReq.Header.Set("Authorization", "Bearer "+adminToken)
	suspendReq.Header.Set("Content-Type", "application/json")
	suspendResp, err := http.DefaultClient.Do(suspendReq)
	require.NoError(t, err)
	defer func() { _ = suspendResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, suspendResp.StatusCode)

	// 3. Verify writes are blocked.
	writeReq2, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+org.Slug+"/spaces",
		strings.NewReader(`{"name":"Another Space"}`))
	writeReq2.Header.Set("Authorization", "Bearer "+userToken)
	writeReq2.Header.Set("Content-Type", "application/json")
	writeResp2, err := http.DefaultClient.Do(writeReq2)
	require.NoError(t, err)
	_ = writeResp2.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, writeResp2.StatusCode)

	// 4. Verify reads still work.
	readReq, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.Slug, nil)
	readReq.Header.Set("Authorization", "Bearer "+userToken)
	readResp, err := http.DefaultClient.Do(readReq)
	require.NoError(t, err)
	_ = readResp.Body.Close()
	assert.Equal(t, http.StatusOK, readResp.StatusCode)

	// 5. Unsuspend the org.
	unsuspendReq, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/admin/orgs/"+org.Slug+"/unsuspend",
		strings.NewReader("{}"))
	unsuspendReq.Header.Set("Authorization", "Bearer "+adminToken)
	unsuspendReq.Header.Set("Content-Type", "application/json")
	unsuspendResp, err := http.DefaultClient.Do(unsuspendReq)
	require.NoError(t, err)
	defer func() { _ = unsuspendResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, unsuspendResp.StatusCode)

	// 6. Verify writes work again.
	writeReq3, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+org.Slug+"/spaces",
		strings.NewReader(`{"name":"Final Space"}`))
	writeReq3.Header.Set("Authorization", "Bearer "+userToken)
	writeReq3.Header.Set("Content-Type", "application/json")
	writeResp3, err := http.DefaultClient.Do(writeReq3)
	require.NoError(t, err)
	_ = writeResp3.Body.Close()
	assert.NotEqual(t, http.StatusServiceUnavailable, writeResp3.StatusCode)
}

// TestLive_Admin_ListOrgs verifies GET /v1/admin/orgs.
func TestLive_Admin_ListOrgs(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	require.NoError(t, env.DB.Create(&models.Org{Name: "Org A", Slug: "org-a-live", Metadata: "{}"}).Error)
	require.NoError(t, env.DB.Create(&models.Org{Name: "Org B", Slug: "org-b-live", Metadata: "{}"}).Error)

	req, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.GreaterOrEqual(t, len(body.Data), 2)
}

// TestLive_Admin_GetOrg verifies GET /v1/admin/orgs/{org}.
func TestLive_Admin_GetOrg(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	org := &models.Org{Name: "Detail Org", Slug: "detail-org-live", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	req, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/orgs/"+org.Slug, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "Detail Org", body["name"])
	assert.NotNil(t, body["member_count"])
}

// TestLive_Admin_TransferOwnership verifies POST /v1/admin/orgs/{org}/transfer-ownership.
func TestLive_Admin_TransferOwnership(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	org := &models.Org{Name: "Transfer Live", Slug: "transfer-live", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)
	require.NoError(t, env.DB.Create(&models.OrgMembership{OrgID: org.ID, UserID: "old_owner", Role: models.RoleOwner}).Error)

	req, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/admin/orgs/"+org.Slug+"/transfer-ownership",
		strings.NewReader(`{"new_owner_user_id":"new_owner"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "transferred", body["status"])
}

// TestLive_Admin_PlatformAdminCRUD verifies platform admin CRUD.
func TestLive_Admin_PlatformAdminCRUD(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	// List — should have at least the bootstrap admin.
	listReq, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/platform-admins", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	listBody, _ := io.ReadAll(listResp.Body)
	_ = listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	assert.Contains(t, string(listBody), "data")

	// Add a new admin.
	addReq, _ := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":"new_admin_live"}`))
	addReq.Header.Set("Authorization", "Bearer "+token)
	addReq.Header.Set("Content-Type", "application/json")
	addResp, err := http.DefaultClient.Do(addReq)
	require.NoError(t, err)
	_ = addResp.Body.Close()
	assert.Equal(t, http.StatusCreated, addResp.StatusCode)

	// Remove the new admin.
	delReq, _ := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/platform-admins/new_admin_live", nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	_ = delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
}

// TestLive_Admin_CannotRemoveLastAdmin verifies removing the last platform admin fails.
func TestLive_Admin_CannotRemoveLastAdmin(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	adminUserID := "sole_admin"
	require.NoError(t, env.DB.Create(&models.PlatformAdmin{
		UserID: adminUserID, GrantedBy: "test", IsActive: true,
	}).Error)

	token := env.SignToken(auth.JWTClaims{
		Subject:   adminUserID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	delReq, _ := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/platform-admins/"+adminUserID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer func() { _ = delResp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, delResp.StatusCode)
}

// TestLive_Admin_AuditLog verifies GET /v1/admin/audit-log.
func TestLive_Admin_AuditLog(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	// Create an audit entry.
	require.NoError(t, env.DB.Create(&models.AuditLog{
		UserID: "admin1", Action: "ban", EntityType: "user", EntityID: "u1",
	}).Error)

	req, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/audit-log", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.GreaterOrEqual(t, len(body.Data), 1)
}

// TestLive_Admin_PurgeUser verifies DELETE /v1/admin/users/{user_id}/purge.
func TestLive_Admin_PurgeUser(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	now := time.Now()
	require.NoError(t, env.DB.Create(&models.UserShadow{
		ClerkUserID: "purge_target", Email: "purge@test.com", DisplayName: "Purge", LastSeenAt: now, SyncedAt: now,
	}).Error)

	req, _ := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/users/purge_target/purge", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "purged", body["status"])
}

// TestLive_Admin_BootstrapFromEnvVar verifies admin bootstrap from env var.
func TestLive_Admin_BootstrapFromEnvVar(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Simulate bootstrap by creating admin directly.
	bootstrapUser := "bootstrap_env_user"
	require.NoError(t, env.DB.Create(&models.PlatformAdmin{
		UserID: bootstrapUser, GrantedBy: "bootstrap", IsActive: true,
	}).Error)

	token := env.SignToken(auth.JWTClaims{
		Subject:   bootstrapUser,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// The bootstrapped user should be able to access admin endpoints.
	req, _ := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/admin/platform-admins", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestLive_Admin_OrgPurge verifies DELETE /v1/admin/orgs/{org}/purge.
func TestLive_Admin_OrgPurge(t *testing.T) {
	env := liveAuthServer(t)
	defer func() {
		// Force WAL checkpoint before cleanup to avoid TempDir issues with SQLite.
		env.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		env.Cleanup()
	}()
	token := setupAdminToken(t, env)

	org := &models.Org{Name: "Purge Org", Slug: "purge-org-live", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	req, _ := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/orgs/"+org.ID+"/purge",
		strings.NewReader(`{"confirm":"purge `+org.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestLive_Admin_OrgPurge_BadConfirm verifies purge org requires confirm.
func TestLive_Admin_OrgPurge_BadConfirm(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()
	token := setupAdminToken(t, env)

	org := &models.Org{Name: "NoPurge Org", Slug: "nopurge-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	req, _ := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/admin/orgs/"+org.ID+"/purge",
		strings.NewReader(`{"confirm":"wrong"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
