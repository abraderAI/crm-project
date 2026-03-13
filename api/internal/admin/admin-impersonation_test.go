package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Impersonation Service Tests ---

func TestImpersonation_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := adminCtx()

	// Setup: admin and a regular user.
	_, err := svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	require.NoError(t, err)
	svc.SyncUserShadow(ctx, "regular_user", "user@test.com", "Regular User")

	// Step 1: Create impersonation token.
	token, signed, err := svc.ImpersonateUser(ctx, "admin1", "regular_user", "support ticket #123", 0)
	require.NoError(t, err)
	require.NotEmpty(t, signed)
	assert.Equal(t, "admin1", token.ImpersonatorID)
	assert.Equal(t, "regular_user", token.TargetUserID)
	assert.Equal(t, "support ticket #123", token.Reason)
	assert.True(t, token.ExpiresAt.After(time.Now()))
	// Default duration → 30 minutes.
	assert.True(t, token.ExpiresAt.Before(time.Now().Add(31*time.Minute)))

	// Step 2: Validate the token.
	validated, err := ValidateImpersonationToken(signed)
	require.NoError(t, err)
	assert.Equal(t, "admin1", validated.ImpersonatorID)
	assert.Equal(t, "regular_user", validated.TargetUserID)

	// Step 3: Cannot impersonate yourself.
	_, _, err = svc.ImpersonateUser(ctx, "admin1", "admin1", "test", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot impersonate yourself")

	// Step 4: Cannot impersonate another platform admin.
	_, err = svc.AddPlatformAdmin(ctx, "admin2", "admin1")
	require.NoError(t, err)
	_, _, err = svc.ImpersonateUser(ctx, "admin1", "admin2", "test", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot impersonate a platform admin")
}

func TestImpersonation_TokenExpiry(t *testing.T) {
	// Create an already-expired token.
	expiredToken := &ImpersonationToken{
		ImpersonatorID: "admin1",
		TargetUserID:   "target",
		Reason:         "test",
		ExpiresAt:      time.Now().Add(-1 * time.Hour),
	}
	signedExpired, err := signImpersonationToken(expiredToken)
	require.NoError(t, err)

	_, err = ValidateImpersonationToken(signedExpired)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestImpersonation_InvalidTokens(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dot", "nodottoken"},
		{"bad base64 payload", "!!!.dGVzdA"},
		{"bad base64 sig", "dGVzdA.!!!"},
		{"tampered payload", "dGVzdA.dGVzdA"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateImpersonationToken(tc.token)
			require.Error(t, err)
		})
	}
}

func TestImpersonation_DurationClamping(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := adminCtx()
	svc.SyncUserShadow(ctx, "target", "t@test.com", "T")

	// Over max → clamped to 120 min.
	token, _, err := svc.ImpersonateUser(ctx, "admin1", "target", "test", 500)
	require.NoError(t, err)
	assert.True(t, token.ExpiresAt.Before(time.Now().Add(121*time.Minute)))

	// Negative → default 30 min.
	token, _, err = svc.ImpersonateUser(ctx, "admin1", "target", "test", -5)
	require.NoError(t, err)
	assert.True(t, token.ExpiresAt.Before(time.Now().Add(31*time.Minute)))
	assert.True(t, token.ExpiresAt.After(time.Now().Add(29*time.Minute)))

	// Zero → default 30 min.
	token, _, err = svc.ImpersonateUser(ctx, "admin1", "target", "test", 0)
	require.NoError(t, err)
	assert.True(t, token.ExpiresAt.Before(time.Now().Add(31*time.Minute)))

	// Exact custom duration (60 min).
	token, _, err = svc.ImpersonateUser(ctx, "admin1", "target", "test", 60)
	require.NoError(t, err)
	assert.True(t, token.ExpiresAt.After(time.Now().Add(59*time.Minute)))
	assert.True(t, token.ExpiresAt.Before(time.Now().Add(61*time.Minute)))
}

func TestImpersonation_ValidationErrors(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := adminCtx()

	_, _, err := svc.ImpersonateUser(ctx, "admin1", "", "reason", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target user_id is required")

	_, _, err = svc.ImpersonateUser(ctx, "", "target", "reason", 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "impersonator_id is required")
}

func TestImpersonation_SignVerifyRoundtrip(t *testing.T) {
	token := &ImpersonationToken{
		ImpersonatorID: "admin_abc",
		TargetUserID:   "user_xyz",
		Reason:         "testing round trip with unicode: 日本語",
		ExpiresAt:      time.Now().Add(30 * time.Minute),
	}
	signed, err := signImpersonationToken(token)
	require.NoError(t, err)

	validated, err := ValidateImpersonationToken(signed)
	require.NoError(t, err)
	assert.Equal(t, token.ImpersonatorID, validated.ImpersonatorID)
	assert.Equal(t, token.TargetUserID, validated.TargetUserID)
	assert.Equal(t, token.Reason, validated.Reason)
}

// --- Impersonation Handler Tests ---

func TestHandler_ImpersonateUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "target_u", Email: "target@test.com", DisplayName: "Target", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/target_u/impersonate",
		strings.NewReader(`{"reason":"support ticket"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "target_u")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.ImpersonateHandler(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["token"])
}

func TestHandler_ImpersonateUser_EmptyUserID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users//impersonate",
		strings.NewReader(`{"reason":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "")
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImpersonateUser_MissingReason(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/impersonate",
		strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "u1")
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImpersonateUser_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/impersonate", strings.NewReader("invalid"))
	r = chiCtx(r, "user_id", "u1")
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImpersonateUser_CannotImpersonateAdmin(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "admin2", GrantedBy: "bootstrap", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/admin2/impersonate",
		strings.NewReader(`{"reason":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "admin2")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImpersonateUser_AuditLog(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "audit_target", Email: "at@test.com", DisplayName: "AT", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/audit_target/impersonate",
		strings.NewReader(`{"reason":"audit test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "audit_target")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "audit_admin", AuthMethod: auth.AuthMethodJWT}))
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for async audit log goroutine.
	time.Sleep(200 * time.Millisecond)

	// Verify audit log entry.
	var log models.AuditLog
	err := db.Where("action = ? AND entity_type = ?", "impersonate", "user").First(&log).Error
	require.NoError(t, err)
	assert.Equal(t, "audit_admin", log.UserID)
	assert.Equal(t, "audit_target", log.EntityID)
}

// --- isImpersonationValidationErr Tests ---

func TestIsImpersonationValidationErr(t *testing.T) {
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("cannot impersonate a platform admin")))
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("cannot impersonate yourself")))
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("target user_id is required")))
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("impersonator_id is required")))
	assert.False(t, isImpersonationValidationErr(fmt.Errorf("some other error")))
}
