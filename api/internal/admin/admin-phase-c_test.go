package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Mock Storage Provider ---

type mockStorage struct {
	files map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{files: make(map[string][]byte)}
}

func (m *mockStorage) Store(filename string, content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	m.files[filename] = data
	return filename, nil
}

func (m *mockStorage) Get(storagePath string) (io.ReadCloser, error) {
	data, ok := m.files[storagePath]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", storagePath)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorage) Delete(storagePath string) error {
	delete(m.files, storagePath)
	return nil
}

// --- Impersonation Service Tests ---

func TestImpersonation_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

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
	ctx := context.Background()
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
	ctx := context.Background()

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
	body := `{"reason":"support ticket","duration_minutes":60}`
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/target_u/impersonate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "target_u")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.ImpersonateHandler(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["token"])
	assert.NotEmpty(t, resp["expires_at"])
	assert.Equal(t, "target_u", resp["target_id"])
}

func TestHandler_ImpersonateUser_EmptyUserID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users//impersonate", strings.NewReader(`{}`))
	r = chiCtx(r, "user_id", "")
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImpersonateUser_MissingReason(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/impersonate", strings.NewReader(`{"duration_minutes":30}`))
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
	db.Create(&models.PlatformAdmin{UserID: "admin_target", GrantedBy: "bootstrap", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/admin_target/impersonate",
		strings.NewReader(`{"reason":"test","duration_minutes":30}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "admin_target")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "other_admin", AuthMethod: auth.AuthMethodJWT}))
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImpersonateUser_AuditLog(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "audit_target", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/audit_target/impersonate",
		strings.NewReader(`{"reason":"audit test","duration_minutes":30}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "audit_target")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin_audit", AuthMethod: auth.AuthMethodJWT}))
	h.ImpersonateHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// Give async audit log goroutine time.
	time.Sleep(200 * time.Millisecond)

	var audit models.AuditLog
	err := db.Where("action = ? AND entity_id = ?", "impersonate", "audit_target").First(&audit).Error
	require.NoError(t, err)
	assert.Contains(t, audit.AfterState, "audit test")
}

// --- Export Service Tests ---

func TestExport_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()
	now := time.Now()

	// Create test data.
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "alice@test.com", DisplayName: "Alice", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u2", Email: "bob@test.com", DisplayName: "Bob", LastSeenAt: now, SyncedAt: now})

	// Step 1: Create export.
	export, err := svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
	require.NoError(t, err)
	assert.Equal(t, "pending", export.Status)
	assert.NotEmpty(t, export.ID)

	// Step 2: Wait for background processing.
	time.Sleep(500 * time.Millisecond)

	// Step 3: Poll status.
	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "completed", updated.Status)
	assert.NotEmpty(t, updated.FilePath)
	assert.NotNil(t, updated.CompletedAt)

	// Step 4: Verify stored content.
	require.True(t, len(storage.files) > 0)
	for _, data := range storage.files {
		content := string(data)
		assert.Contains(t, content, "alice@test.com")
		assert.Contains(t, content, "bob@test.com")
	}
}

func TestExport_JSONFormat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()
	now := time.Now()

	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "json@test.com", DisplayName: "JSON User", LastSeenAt: now, SyncedAt: now})

	export, err := svc.CreateExport(ctx, "users", "json", "{}", "admin1", storage)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)

	for _, data := range storage.files {
		assert.True(t, json.Valid(data), "stored JSON export should be valid JSON")
		assert.Contains(t, string(data), "json@test.com")
	}
}

func TestExport_OrgType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.Org{Name: "Export Org", Slug: "export-org", Metadata: "{}"})

	export, err := svc.CreateExport(ctx, "orgs", "csv", "{}", "admin1", storage)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)

	for _, data := range storage.files {
		assert.Contains(t, string(data), "Export Org")
	}
}

func TestExport_AuditType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.AuditLog{UserID: "admin1", Action: "ban", EntityType: "user", EntityID: "u1"})

	export, err := svc.CreateExport(ctx, "audit", "json", "{}", "admin1", storage)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
}

func TestExport_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.CreateExport(ctx, "invalid", "csv", "{}", "admin1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid export type")
}

func TestExport_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.CreateExport(ctx, "users", "xml", "{}", "admin1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid export format")
}

func TestExport_InvalidFilters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.CreateExport(ctx, "users", "csv", "{invalid", "admin1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filters JSON")
}

func TestExport_EmptyRequestedBy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.CreateExport(ctx, "users", "csv", "{}", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requested_by is required")
}

func TestExport_EmptyFiltersDefaultsToEmptyJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	export, err := svc.CreateExport(ctx, "users", "csv", "", "admin1", storage)
	require.NoError(t, err)
	assert.Equal(t, "{}", export.Filters)
}

func TestExport_ListExports(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	// Create multiple exports.
	for i := 0; i < 3; i++ {
		_, err := svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
		require.NoError(t, err)
	}
	time.Sleep(500 * time.Millisecond)

	exports, pageInfo, err := svc.ListExports(ctx, "admin1", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, exports, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestExport_ListExports_FiltersRequestedBy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	_, _ = svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
	_, _ = svc.CreateExport(ctx, "users", "csv", "{}", "admin2", storage)
	time.Sleep(500 * time.Millisecond)

	exports, _, err := svc.ListExports(ctx, "admin1", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, exports, 1)
}

func TestExport_ListExports_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create 5 exports directly in DB.
	for i := 0; i < 5; i++ {
		db.Create(&models.AdminExport{Type: "users", Format: "csv", Status: "completed", RequestedBy: "admin_page"})
	}

	exports, pageInfo, err := svc.ListExports(ctx, "admin_page", pagination.Params{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, exports, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)
}

func TestExport_AuditCSV(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.AuditLog{UserID: "admin1", Action: "ban", EntityType: "user", EntityID: "u1"})

	export, err := svc.CreateExport(ctx, "audit", "csv", "{}", "admin1", storage)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)

	for _, data := range storage.files {
		content := string(data)
		assert.Contains(t, content, "user_id")
		assert.Contains(t, content, "action")
	}
}

func TestExport_OrgsJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.Org{Name: "JSON Org", Slug: "json-org", Metadata: "{}"})

	export, err := svc.CreateExport(ctx, "orgs", "json", "{}", "admin1", storage)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)

	for _, data := range storage.files {
		assert.True(t, json.Valid(data))
		assert.Contains(t, string(data), "JSON Org")
	}
}

func TestExport_GetExport_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	export, err := svc.GetExport(context.Background(), "nonexistent-id")
	require.NoError(t, err)
	assert.Nil(t, export)
}

func TestExport_NilStorage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "s@t.com", DisplayName: "S", LastSeenAt: now, SyncedAt: now})

	// When storage is nil, the export should still complete with filename as path.
	export, err := svc.CreateExport(ctx, "users", "csv", "{}", "admin1", nil)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	updated, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.NotEmpty(t, updated.FilePath)
}

// --- Export Handler Tests ---

func TestHandler_CreateExport(t *testing.T) {
	h, _ := setupTestHandler(t)
	h.SetStorage(newMockStorage())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"users","format":"csv"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.CreateExportHandler(w, r)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["export_id"])
	assert.Equal(t, "pending", resp["status"])
}

func TestHandler_CreateExport_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports", strings.NewReader("invalid"))
	h.CreateExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateExport_MissingFields(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"users"}`))
	r.Header.Set("Content-Type", "application/json")
	h.CreateExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateExport_InvalidType(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"bad","format":"csv"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.CreateExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListExports(t *testing.T) {
	h, _ := setupTestHandler(t)
	h.SetStorage(newMockStorage())

	// Create an export first.
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"users","format":"csv"}`))
	r1.Header.Set("Content-Type", "application/json")
	r1 = r1.WithContext(auth.SetUserContext(r1.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.CreateExportHandler(w1, r1)

	// List exports.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports", nil)
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.ListExportsHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_GetExport(t *testing.T) {
	h, db := setupTestHandler(t)
	export := models.AdminExport{Type: "users", Format: "csv", Status: "completed", RequestedBy: "admin1"}
	db.Create(&export)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports/"+export.ID, nil)
	r = chiCtx(r, "id", export.ID)
	h.GetExportHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "completed")
}

func TestHandler_GetExport_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports/nonexistent", nil)
	r = chiCtx(r, "id", "nonexistent")
	h.GetExportHandler(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetExport_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports/", nil)
	r = chiCtx(r, "id", "")
	h.GetExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- API Usage Tests ---

func TestAPIUsage_CounterMiddleware(t *testing.T) {
	db := setupTestDB(t)
	counter := APIUsageCounter(db)

	handler := counter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make several requests.
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/test/endpoint", nil)
		handler.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Give async goroutines time to complete.
	time.Sleep(300 * time.Millisecond)

	// Verify counts.
	var stat models.APIUsageStat
	err := db.Where("endpoint = ? AND method = ?", "/v1/test/endpoint", "GET").First(&stat).Error
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stat.Count, int64(1))
}

func TestAPIUsage_ServiceQuery(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Insert some usage data directly.
	hour := time.Now().UTC().Format("2006-01-02-15")
	db.Create(&models.APIUsageStat{Endpoint: "/v1/users", Method: "GET", Hour: hour, Count: 100})
	db.Create(&models.APIUsageStat{Endpoint: "/v1/orgs", Method: "POST", Hour: hour, Count: 50})

	results, err := svc.GetAPIUsage(context.Background(), "24h")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	// Sorted by count DESC.
	assert.Equal(t, int64(100), results[0].Count)
	assert.Equal(t, "/v1/users", results[0].Endpoint)
}

func TestAPIUsage_ServiceQuery_Periods(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	hour := time.Now().UTC().Format("2006-01-02-15")
	db.Create(&models.APIUsageStat{Endpoint: "/v1/test", Method: "GET", Hour: hour, Count: 10})

	for _, period := range []string{"24h", "7d", "30d", "unknown"} {
		results, err := svc.GetAPIUsage(context.Background(), period)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1, "period: %s", period)
	}
}

func TestAPIUsage_ServiceQuery_OldData(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Data from 48 hours ago should not appear in 24h query.
	oldHour := time.Now().UTC().Add(-48 * time.Hour).Format("2006-01-02-15")
	db.Create(&models.APIUsageStat{Endpoint: "/v1/old", Method: "GET", Hour: oldHour, Count: 999})

	results, err := svc.GetAPIUsage(context.Background(), "24h")
	require.NoError(t, err)
	for _, r := range results {
		assert.NotEqual(t, "/v1/old", r.Endpoint)
	}
}

// --- API Usage Handler Tests ---

func TestHandler_GetAPIUsage(t *testing.T) {
	h, db := setupTestHandler(t)
	hour := time.Now().UTC().Format("2006-01-02-15")
	db.Create(&models.APIUsageStat{Endpoint: "/v1/test", Method: "GET", Hour: hour, Count: 42})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/api-usage?period=24h", nil)
	h.GetAPIUsageHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "24h", resp["period"])
	assert.NotNil(t, resp["data"])
}

func TestHandler_GetAPIUsage_DefaultPeriod(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/api-usage", nil)
	h.GetAPIUsageHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "24h", resp["period"])
}

// --- LLM Usage Tests ---

func TestLLMUsage_Service(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Insert some LLM usage logs.
	for i := 0; i < 3; i++ {
		db.Create(&models.LLMUsageLog{
			Endpoint:     fmt.Sprintf("/v1/enrich/%d", i),
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 50,
			DurationMs:   200,
		})
	}

	entries, err := svc.GetLLMUsage(ctx, 50)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Verify fields.
	assert.Equal(t, "gpt-4", entries[0].Model)
	assert.Equal(t, int64(100), entries[0].InputTokens)
}

func TestLLMUsage_Service_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Negative or 0 limit defaults to 50.
	entries, err := svc.GetLLMUsage(context.Background(), 0)
	require.NoError(t, err)
	assert.Empty(t, entries) // No data, but no error.

	entries, err = svc.GetLLMUsage(context.Background(), -1)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLLMUsage_Service_OverLimit(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Limit > 100 clamps to 50.
	entries, err := svc.GetLLMUsage(context.Background(), 200)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// --- LLM Usage Handler Tests ---

func TestHandler_GetLLMUsage(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.LLMUsageLog{Endpoint: "/v1/enrich", Model: "gpt-4", InputTokens: 100, OutputTokens: 50, DurationMs: 200})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/llm-usage", nil)
	h.GetLLMUsageHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["data"])
}

// --- Security Monitoring Tests ---

func TestLoginEventRecorder(t *testing.T) {
	db := setupTestDB(t)
	ResetLoginDebounceCache()

	handler := LoginEventRecorder(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "login_user", AuthMethod: auth.AuthMethodJWT})
	r = r.WithContext(ctx)
	r.RemoteAddr = "192.168.1.1:12345"
	r.Header.Set("User-Agent", "Test-Agent/1.0")
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for async write.
	time.Sleep(300 * time.Millisecond)

	var event models.LoginEvent
	err := db.Where("user_id = ?", "login_user").First(&event).Error
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", event.IPAddress)
	assert.Equal(t, "Test-Agent/1.0", event.UserAgent)
}

func TestLoginEventRecorder_Debounce(t *testing.T) {
	db := setupTestDB(t)
	ResetLoginDebounceCache()

	handler := LoginEventRecorder(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request records.
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "debounce_user", AuthMethod: auth.AuthMethodJWT})
		r = r.WithContext(ctx)
		handler.ServeHTTP(w, r)
	}

	time.Sleep(300 * time.Millisecond)

	// Should only have 1 login event due to debouncing.
	var count int64
	db.Model(&models.LoginEvent{}).Where("user_id = ?", "debounce_user").Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestLoginEventRecorder_NoUserContext(t *testing.T) {
	db := setupTestDB(t)
	ResetLoginDebounceCache()

	handler := LoginEventRecorder(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(100 * time.Millisecond)

	var count int64
	db.Model(&models.LoginEvent{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestFailedAuthRecorder(t *testing.T) {
	db := setupTestDB(t)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	handler := FailedAuthRecorder(db)(innerHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("Authorization", "Bearer some-token-value-here-long-enough")
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	time.Sleep(300 * time.Millisecond)

	var failedAuth models.FailedAuth
	err := db.Where("ip_address = ?", "10.0.0.1").First(&failedAuth).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), failedAuth.Count)
	assert.Contains(t, failedAuth.UserID, "bearer:")
}

func TestFailedAuthRecorder_APIKey(t *testing.T) {
	db := setupTestDB(t)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	handler := FailedAuthRecorder(db)(innerHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.2:12345"
	r.Header.Set("X-API-Key", "deft_live_1234567890abcdef")
	handler.ServeHTTP(w, r)

	time.Sleep(300 * time.Millisecond)

	var failedAuth models.FailedAuth
	err := db.Where("ip_address = ?", "10.0.0.2").First(&failedAuth).Error
	require.NoError(t, err)
	assert.Contains(t, failedAuth.UserID, "apikey:")
}

func TestFailedAuthRecorder_NonUnauthorized(t *testing.T) {
	db := setupTestDB(t)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := FailedAuthRecorder(db)(innerHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.3:12345"
	handler.ServeHTTP(w, r)

	time.Sleep(100 * time.Millisecond)

	var count int64
	db.Model(&models.FailedAuth{}).Where("ip_address = ?", "10.0.0.3").Count(&count)
	assert.Equal(t, int64(0), count)
}

// --- Security Service Tests ---

func TestGetRecentLogins(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	for i := 0; i < 3; i++ {
		db.Create(&models.LoginEvent{
			UserID:    fmt.Sprintf("user_%d", i),
			IPAddress: fmt.Sprintf("10.0.0.%d", i),
			UserAgent: "Test-Agent",
		})
	}

	entries, pageInfo, err := svc.GetRecentLogins(context.Background(), pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestGetRecentLogins_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	for i := 0; i < 5; i++ {
		db.Create(&models.LoginEvent{
			UserID:    fmt.Sprintf("page_user_%d", i),
			IPAddress: "10.0.0.1",
			UserAgent: "Test",
		})
	}

	entries, pageInfo, err := svc.GetRecentLogins(context.Background(), pagination.Params{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)
}

func TestGetFailedAuths(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	hour := time.Now().UTC().Format("2006-01-02-15")
	db.Create(&models.FailedAuth{IPAddress: "10.0.0.1", UserID: "bearer:test", Hour: hour, Count: 10})
	db.Create(&models.FailedAuth{IPAddress: "10.0.0.2", UserID: "", Hour: hour, Count: 5})

	entries, err := svc.GetFailedAuths(context.Background(), "24h")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	// Sorted by count DESC.
	assert.Equal(t, int64(10), entries[0].Count)
}

func TestGetFailedAuths_Periods(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	hour := time.Now().UTC().Format("2006-01-02-15")
	db.Create(&models.FailedAuth{IPAddress: "10.0.0.1", UserID: "", Hour: hour, Count: 1})

	for _, period := range []string{"24h", "7d", "unknown"} {
		entries, err := svc.GetFailedAuths(context.Background(), period)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1, "period: %s", period)
	}
}

func TestGetFailedAuths_OldData(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	oldHour := time.Now().UTC().Add(-48 * time.Hour).Format("2006-01-02-15")
	db.Create(&models.FailedAuth{IPAddress: "10.0.0.99", UserID: "", Hour: oldHour, Count: 999})

	entries, err := svc.GetFailedAuths(context.Background(), "24h")
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotEqual(t, "10.0.0.99", e.IPAddress)
	}
}

// --- Security Handler Tests ---

func TestHandler_GetRecentLogins(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.LoginEvent{UserID: "u1", IPAddress: "1.2.3.4", UserAgent: "Chrome"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/security/recent-logins", nil)
	h.GetRecentLoginsHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_GetFailedAuths(t *testing.T) {
	h, db := setupTestHandler(t)
	hour := time.Now().UTC().Format("2006-01-02-15")
	db.Create(&models.FailedAuth{IPAddress: "10.0.0.1", UserID: "test", Hour: hour, Count: 5})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/security/failed-auths?period=24h", nil)
	h.GetFailedAuthsHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "24h", resp["period"])
}

func TestHandler_GetFailedAuths_DefaultPeriod(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/security/failed-auths", nil)
	h.GetFailedAuthsHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "24h", resp["period"])
}

// --- extractIP Tests ---

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		xri      string
		remote   string
		expected string
	}{
		{"XFF first", "1.2.3.4, 5.6.7.8", "", "9.9.9.9:1234", "1.2.3.4"},
		{"XRI", "", "5.6.7.8", "9.9.9.9:1234", "5.6.7.8"},
		{"RemoteAddr with port", "", "", "10.0.0.1:5555", "10.0.0.1"},
		{"RemoteAddr no port", "", "", "10.0.0.2", "10.0.0.2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remote
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" {
				r.Header.Set("X-Real-IP", tc.xri)
			}
			assert.Equal(t, tc.expected, extractIP(r))
		})
	}
}

// --- statusRecorder Tests ---

func TestStatusRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	sr.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, sr.statusCode)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// --- isImpersonationValidationErr Tests ---

func TestIsImpersonationValidationErr(t *testing.T) {
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("cannot impersonate a platform admin")))
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("cannot impersonate yourself")))
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("target user_id is required")))
	assert.True(t, isImpersonationValidationErr(fmt.Errorf("impersonator_id is required")))
	assert.False(t, isImpersonationValidationErr(fmt.Errorf("some other error")))
}

// --- isExportValidationErr Tests ---

func TestIsExportValidationErr(t *testing.T) {
	assert.True(t, isExportValidationErr(fmt.Errorf("invalid export type: foo")))
	assert.True(t, isExportValidationErr(fmt.Errorf("invalid export format: xml")))
	assert.True(t, isExportValidationErr(fmt.Errorf("requested_by is required")))
	assert.True(t, isExportValidationErr(fmt.Errorf("invalid filters JSON")))
	assert.False(t, isExportValidationErr(fmt.Errorf("database error")))
}

// --- ResetLoginDebounceCache Tests ---

func TestResetLoginDebounceCache(t *testing.T) {
	loginDebounceCache.Lock()
	loginDebounceCache.seen["test-key"] = time.Now()
	loginDebounceCache.Unlock()

	ResetLoginDebounceCache()

	loginDebounceCache.Lock()
	assert.Empty(t, loginDebounceCache.seen)
	loginDebounceCache.Unlock()
}

// --- Fuzz Tests ---

func FuzzImpersonationToken(f *testing.F) {
	f.Add("valid.token")
	f.Add("")
	f.Add("dGVzdA.dGVzdA")
	f.Add("no-dot-token")
	f.Add(strings.Repeat("a", 5000))
	f.Add("ab.cd.ef.gh")
	f.Add("eyJ0ZXN0IjoidmFsdWUifQ.AAAA")
	f.Add("payload with spaces.signature with spaces")
	f.Add("<script>alert(1)</script>.<img src=x>")
	f.Add("unicode-日本語.テスト")

	f.Fuzz(func(t *testing.T, token string) {
		// Must not panic for any input.
		_, _ = ValidateImpersonationToken(token)
	})
}

func FuzzExportFilters(f *testing.F) {
	f.Add("{}")
	f.Add("")
	f.Add("{invalid json}")
	f.Add(`{"key":"value"}`)
	f.Add(strings.Repeat("{", 1000))
	f.Add(`[1,2,3]`)
	f.Add(`{"nested":{"deep":{"value":true}}}`)
	f.Add("null")
	f.Add("<script>alert(1)</script>")
	f.Add("unicode: 日本語テスト")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.AdminExport{}, &models.UserShadow{}, &models.Org{}, &models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, filters string) {
		// Must not panic.
		_, _ = svc.CreateExport(ctx, "users", "csv", filters, "admin1", nil)
	})
}

func FuzzUsageQueryParams(f *testing.F) {
	f.Add("24h")
	f.Add("7d")
	f.Add("30d")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("x", 5000))
	f.Add("1h")
	f.Add("-1d")
	f.Add("<script>")
	f.Add("日本語")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.APIUsageStat{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, period string) {
		// Must not panic.
		_, _ = svc.GetAPIUsage(ctx, period)
	})
}

func FuzzExportType(f *testing.F) {
	f.Add("users")
	f.Add("orgs")
	f.Add("audit")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("x", 5000))
	f.Add("<script>alert(1)</script>")
	f.Add("USERS")
	f.Add("日本語")
	f.Add("users; DROP TABLE--")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.AdminExport{}, &models.UserShadow{}, &models.Org{}, &models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, exportType string) {
		// Must not panic.
		_, _ = svc.CreateExport(ctx, exportType, "csv", "{}", "admin1", nil)
	})
}

func FuzzFailedAuthPeriod(f *testing.F) {
	f.Add("24h")
	f.Add("7d")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("a", 5000))
	f.Add("-1d")
	f.Add("1000d")
	f.Add("<script>")
	f.Add("日本語")
	f.Add("'; DROP TABLE--")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.FailedAuth{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, period string) {
		// Must not panic.
		_, _ = svc.GetFailedAuths(ctx, period)
	})
}

func FuzzImpersonationCreate(f *testing.F) {
	f.Add("admin1", "user1", "reason", 30)
	f.Add("", "", "", 0)
	f.Add("admin", "admin", "self", -1)
	f.Add(strings.Repeat("a", 500), "target", "long admin id", 999)
	f.Add("admin1", "target", "<script>alert(1)</script>", 120)
	f.Add("admin1", "target", "unicode: 日本語", 60)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.PlatformAdmin{}, &models.UserShadow{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, impersonator, target, reason string, duration int) {
		// Must not panic.
		_, _, _ = svc.ImpersonateUser(ctx, impersonator, target, reason, duration)
	})
}
