package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- System Settings Service Tests ---

func TestService_GetAllSettings_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	settings, err := svc.GetAllSettings(context.Background())
	require.NoError(t, err)
	assert.Empty(t, settings)
}

func TestService_UpdateSettings_CreateNew(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	patch := map[string]json.RawMessage{
		"file_upload_limits": json.RawMessage(`{"max_size":10485760,"allowed_types":["image/png","image/jpeg"]}`),
	}
	err := svc.UpdateSettings(ctx, patch, "admin1")
	require.NoError(t, err)

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	assert.Contains(t, settings, "file_upload_limits")
	assert.Contains(t, string(settings["file_upload_limits"]), "10485760")
}

func TestService_UpdateSettings_DeepMerge(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create initial setting.
	patch1 := map[string]json.RawMessage{
		"webhook_retry_policy": json.RawMessage(`{"max_attempts":3,"backoff_multiplier":2.0}`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch1, "admin1"))

	// Deep-merge update.
	patch2 := map[string]json.RawMessage{
		"webhook_retry_policy": json.RawMessage(`{"max_attempts":5}`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch2, "admin1"))

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(settings["webhook_retry_policy"], &result))
	assert.Equal(t, float64(5), result["max_attempts"])
	assert.Equal(t, float64(2.0), result["backoff_multiplier"]) // Preserved from original.
}

func TestService_UpdateSettings_UnknownKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	patch := map[string]json.RawMessage{
		"unknown_key": json.RawMessage(`{}`),
	}
	err := svc.UpdateSettings(context.Background(), patch, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown setting key")
}

func TestService_UpdateSettings_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	patch := map[string]json.RawMessage{
		"file_upload_limits": json.RawMessage(`{invalid`),
	}
	err := svc.UpdateSettings(context.Background(), patch, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestService_UpdateSettings_MultipleKeys(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	patch := map[string]json.RawMessage{
		"file_upload_limits":      json.RawMessage(`{"max_size":1024}`),
		"default_pipeline_stages": json.RawMessage(`["new_lead","contacted","qualified"]`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch, "admin1"))

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	assert.Len(t, settings, 2)
}

func TestService_GetSetting(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Not found.
	setting, err := svc.GetSetting(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, setting)

	// Create and retrieve.
	patch := map[string]json.RawMessage{
		"llm_rate_limits": json.RawMessage(`{"requests_per_minute":60}`),
	}
	require.NoError(t, svc.UpdateSettings(ctx, patch, "admin1"))

	setting, err = svc.GetSetting(ctx, "llm_rate_limits")
	require.NoError(t, err)
	require.NotNil(t, setting)
	assert.Equal(t, "admin1", setting.UpdatedBy)
}

func TestService_UpdateSettings_AllValidKeys(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for key := range KnownSettingKeys {
		patch := map[string]json.RawMessage{key: json.RawMessage(`{"test":true}`)}
		require.NoError(t, svc.UpdateSettings(ctx, patch, "admin1"), "failed for key: %s", key)
	}

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	assert.Len(t, settings, len(KnownSettingKeys))
}

// --- System Settings Handler Tests ---

func TestHandler_GetSettings(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/settings", nil)
	h.GetSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PatchSettings(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"file_upload_limits":{"max_size":5242880}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "5242880")
}

func TestHandler_PatchSettings_InvalidBody(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader("invalid"))
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchSettings_EmptyPatch(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchSettings_UnknownKey(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"bad_key":{"value":1}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSettings_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/settings", nil)
	h.GetSettings(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PatchSettings_ServiceError(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)

	// Create a valid setting, then close DB.
	body := `{"file_upload_limits":{"max_size":1}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r2.Header.Set("Content-Type", "application/json")
	r2 = r2.WithContext(adminCtx())
	h.PatchSettings(w2, r2)
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestHandler_PatchSettings_NoAuthContext(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	body := `{"file_upload_limits":{"max_size":1}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	// No adminCtx — updatedBy will be empty.
	h.PatchSettings(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
