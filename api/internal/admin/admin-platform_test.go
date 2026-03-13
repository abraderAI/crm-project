package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Ensure unused imports are referenced.
var _ = context.Background

// --- Platform Admin Service Tests ---

func TestService_IsPlatformAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Not an admin.
	isAdmin, err := svc.IsPlatformAdmin(ctx, "user1")
	require.NoError(t, err)
	assert.False(t, isAdmin)

	// Add admin.
	_, err = svc.AddPlatformAdmin(ctx, "user1", "bootstrap")
	require.NoError(t, err)

	isAdmin, err = svc.IsPlatformAdmin(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, isAdmin)
}

func TestService_AddPlatformAdmin_EmptyUserID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.AddPlatformAdmin(ctx, "", "granter")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestService_AddPlatformAdmin_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, err := svc.AddPlatformAdmin(ctx, "user1", "bootstrap")
	require.NoError(t, err)

	_, err = svc.AddPlatformAdmin(ctx, "user1", "bootstrap")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestService_ListPlatformAdmins(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	_, _ = svc.AddPlatformAdmin(ctx, "admin2", "admin1")

	admins, err := svc.ListPlatformAdmins(ctx)
	require.NoError(t, err)
	assert.Len(t, admins, 2)
}

func TestService_RemovePlatformAdmin_CannotRemoveLast(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")

	err := svc.RemovePlatformAdmin(ctx, "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove the last platform admin")
}

func TestService_RemovePlatformAdmin_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	_, _ = svc.AddPlatformAdmin(ctx, "admin2", "admin1")

	err := svc.RemovePlatformAdmin(ctx, "admin1")
	require.NoError(t, err)

	isAdmin, _ := svc.IsPlatformAdmin(ctx, "admin1")
	assert.False(t, isAdmin)
}

func TestService_RemovePlatformAdmin_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_, _ = svc.AddPlatformAdmin(ctx, "admin1", "bootstrap")
	_, _ = svc.AddPlatformAdmin(ctx, "admin2", "admin1")

	err := svc.RemovePlatformAdmin(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_BootstrapAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Empty user ID — no-op.
	err := svc.BootstrapAdmin(ctx, "")
	require.NoError(t, err)

	// Bootstrap first admin.
	err = svc.BootstrapAdmin(ctx, "bootstrap_user")
	require.NoError(t, err)

	isAdmin, _ := svc.IsPlatformAdmin(ctx, "bootstrap_user")
	assert.True(t, isAdmin)

	// Idempotent — calling again should not error.
	err = svc.BootstrapAdmin(ctx, "bootstrap_user")
	require.NoError(t, err)
}

func TestHandler_ListPlatformAdmins(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "bootstrap", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/platform-admins", nil)
	h.ListPlatformAdmins(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_AddPlatformAdmin(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":"new_admin"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(adminCtx())
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddPlatformAdmin_EmptyUserID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":""}`))
	r.Header.Set("Content-Type", "application/json")
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddPlatformAdmin_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins", strings.NewReader("invalid"))
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddPlatformAdmin_Duplicate(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "dup", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/platform-admins",
		strings.NewReader(`{"user_id":"dup"}`))
	r.Header.Set("Content-Type", "application/json")
	h.AddPlatformAdmin(w, r)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_RemovePlatformAdmin(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "test", IsActive: true})
	db.Create(&models.PlatformAdmin{UserID: "a2", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/a1", nil)
	r = chiCtx(r, "user_id", "a1")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_RemovePlatformAdmin_LastAdmin(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/a1", nil)
	r = chiCtx(r, "user_id", "a1")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemovePlatformAdmin_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.PlatformAdmin{UserID: "a1", GrantedBy: "test", IsActive: true})
	db.Create(&models.PlatformAdmin{UserID: "a2", GrantedBy: "test", IsActive: true})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/nope", nil)
	r = chiCtx(r, "user_id", "nope")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RemovePlatformAdmin_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/platform-admins/", nil)
	r = chiCtx(r, "user_id", "")
	h.RemovePlatformAdmin(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListPlatformAdmins_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/platform-admins", nil)
	h.ListPlatformAdmins(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
