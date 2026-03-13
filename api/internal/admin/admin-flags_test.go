package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Feature Flags & Maintenance Mode Tests ---

func TestService_SeedFeatureFlags(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	flags, err := svc.ListFeatureFlags(ctx)
	require.NoError(t, err)
	assert.Len(t, flags, 3)

	// Idempotent.
	require.NoError(t, svc.SeedFeatureFlags(ctx))
	flags, _ = svc.ListFeatureFlags(ctx)
	assert.Len(t, flags, 3)
}

func TestService_ListFeatureFlags(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	flags, err := svc.ListFeatureFlags(ctx)
	require.NoError(t, err)
	assert.Len(t, flags, 3)
	// Should be sorted by key.
	assert.Equal(t, "community_voting", flags[0].Key)
}

func TestService_GetFeatureFlag(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	flag, err := svc.GetFeatureFlag(ctx, "maintenance_mode")
	require.NoError(t, err)
	require.NotNil(t, flag)
	assert.False(t, flag.Enabled)

	// Not found.
	flag, err = svc.GetFeatureFlag(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, flag)
}

func TestService_ToggleFeatureFlag(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	// Enable maintenance_mode.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", true, nil))

	flag, _ := svc.GetFeatureFlag(ctx, "maintenance_mode")
	assert.True(t, flag.Enabled)

	// Disable.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", false, nil))
	flag, _ = svc.GetFeatureFlag(ctx, "maintenance_mode")
	assert.False(t, flag.Enabled)
}

func TestService_ToggleFeatureFlag_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	err := svc.ToggleFeatureFlag(context.Background(), "nonexistent", true, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_ToggleFeatureFlag_WithOrgScope(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	orgScope := "org-123"
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "community_voting", true, &orgScope))

	flag, _ := svc.GetFeatureFlag(ctx, "community_voting")
	require.NotNil(t, flag.OrgScope)
	assert.Equal(t, "org-123", *flag.OrgScope)

	// Clear org scope.
	emptyScope := ""
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "community_voting", true, &emptyScope))
	flag, _ = svc.GetFeatureFlag(ctx, "community_voting")
	assert.Nil(t, flag.OrgScope)
}

func TestService_IsFeatureEnabled(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	enabled, err := svc.IsFeatureEnabled(ctx, "community_voting")
	require.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = svc.IsFeatureEnabled(ctx, "maintenance_mode")
	require.NoError(t, err)
	assert.False(t, enabled)

	// Nonexistent flag.
	enabled, err = svc.IsFeatureEnabled(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestHandler_ListFeatureFlags(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/feature-flags", nil)
	h.ListFeatureFlags(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "community_voting")
	assert.Contains(t, w.Body.String(), "maintenance_mode")
}

func TestHandler_PatchFeatureFlag(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"enabled":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/maintenance_mode", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "maintenance_mode")
	r = r.WithContext(context.WithValue(adminCtx(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"enabled":true`)
}

func TestHandler_PatchFeatureFlag_NotFound(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"enabled":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/nonexistent", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_PatchFeatureFlag_MissingEnabled(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"org_scope":"org1"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/maintenance_mode", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "maintenance_mode")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchFeatureFlag_InvalidBody(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/x", strings.NewReader("invalid"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "x")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PatchFeatureFlag_EmptyKey(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/", strings.NewReader(`{"enabled":true}`))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMaintenanceMode_Disabled(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_Enabled_BlocksPost(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "maintenance mode")
}

func TestMaintenanceMode_Enabled_AllowsGet(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/orgs", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_Enabled_AllowsAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/settings", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_NoFlagTable(t *testing.T) {
	// When flag not found, should fail open.
	db := setupTestDB(t)
	svc := NewService(db)
	// Don't seed flags — maintenance_mode doesn't exist.

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFeatureFlagToggle_MaintenanceMode_E2E(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, svc.SeedFeatureFlags(ctx))

	// Initially disabled.
	enabled, _ := svc.IsFeatureEnabled(ctx, "maintenance_mode")
	assert.False(t, enabled)

	// Enable maintenance mode.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", true, nil))

	// Verify non-GET blocked.
	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Disable maintenance mode.
	require.NoError(t, svc.ToggleFeatureFlag(ctx, "maintenance_mode", false, nil))

	// Verify requests work again.
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/v1/orgs", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListFeatureFlags_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/feature-flags", nil)
	h.ListFeatureFlags(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PatchFeatureFlag_ServiceError(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	body := `{"enabled":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/maintenance_mode", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "maintenance_mode")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMaintenanceMode_Enabled_BlocksPut(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/v1/orgs/test", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMaintenanceMode_Enabled_BlocksDelete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/orgs/test", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMaintenanceMode_Enabled_AllowsHead(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.SeedFeatureFlags(context.Background())
	_ = svc.ToggleFeatureFlag(context.Background(), "maintenance_mode", true, nil)

	handler := MaintenanceMode(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/v1/orgs", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PatchFeatureFlag_WithOrgScope(t *testing.T) {
	h, svc, _ := setupTestHandlerWithPolicy(t)
	require.NoError(t, svc.SeedFeatureFlags(context.Background()))

	body := `{"enabled":true,"org_scope":"org-x"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/v1/admin/feature-flags/community_voting", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "community_voting")
	r = r.WithContext(context.WithValue(adminCtx(), chi.RouteCtxKey, rctx))
	h.PatchFeatureFlag(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "org-x")
}
