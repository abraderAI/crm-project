package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, database.Migrate(db))
	return db
}

func TestService_LogSync(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	entry := models.AuditLog{
		UserID:     "user1",
		Action:     models.AuditActionCreate,
		EntityType: "org",
		EntityID:   "org-123",
		AfterState: `{"name":"Test Org"}`,
		RequestID:  "req-1",
	}
	err := svc.LogSync(context.Background(), entry)
	require.NoError(t, err)

	// Verify written.
	logs, _, err := svc.List(context.Background(), ListParams{
		Params: pagination.Params{Limit: 50},
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "org", logs[0].EntityType)
	assert.Equal(t, "org-123", logs[0].EntityID)
}

func TestService_List_Filters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Create multiple entries.
	entries := []models.AuditLog{
		{UserID: "user1", Action: models.AuditActionCreate, EntityType: "org", EntityID: "o1"},
		{UserID: "user2", Action: models.AuditActionUpdate, EntityType: "thread", EntityID: "t1"},
		{UserID: "user1", Action: models.AuditActionDelete, EntityType: "org", EntityID: "o2"},
	}
	for _, e := range entries {
		require.NoError(t, svc.LogSync(context.Background(), e))
	}

	// Filter by entity type.
	logs, _, err := svc.List(context.Background(), ListParams{
		Params:     pagination.Params{Limit: 50},
		EntityType: "org",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)

	// Filter by action.
	logs, _, err = svc.List(context.Background(), ListParams{
		Params: pagination.Params{Limit: 50},
		Action: "create",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	// Filter by user.
	logs, _, err = svc.List(context.Background(), ListParams{
		Params: pagination.Params{Limit: 50},
		UserID: "user2",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestService_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.LogSync(context.Background(), models.AuditLog{
			UserID: "user1", Action: models.AuditActionCreate, EntityType: "org", EntityID: "o",
		}))
	}

	logs, pageInfo, err := svc.List(context.Background(), ListParams{
		Params: pagination.Params{Limit: 2},
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)
}

func TestHandler_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	handler := NewHandler(svc)

	require.NoError(t, svc.LogSync(context.Background(), models.AuditLog{
		UserID: "u1", Action: models.AuditActionCreate, EntityType: "org", EntityID: "o1",
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/test/audit-log", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_List_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	handler := NewHandler(svc)

	require.NoError(t, svc.LogSync(context.Background(), models.AuditLog{
		UserID: "u1", Action: models.AuditActionCreate, EntityType: "org", EntityID: "o1",
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/test/audit-log?entity_type=org&action=create", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestService_Log_Async(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	svc.Log(context.Background(), models.AuditLog{
		UserID:     "async_user",
		Action:     models.AuditActionCreate,
		EntityType: "org",
		EntityID:   "org-async",
		RequestID:  "req-async",
	})

	// Wait for async write.
	time.Sleep(500 * time.Millisecond)

	logs, _, err := svc.List(context.Background(), ListParams{
		Params: pagination.Params{Limit: 50},
		UserID: "async_user",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "org-async", logs[0].EntityID)
}

func TestCreateAuditEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Call without auth context.
	CreateAuditEntry(context.Background(), svc, models.AuditActionCreate, "org", "org-1", nil, map[string]string{"name": "Test"})

	// Wait for async write.
	time.Sleep(500 * time.Millisecond)

	logs, _, err := svc.List(context.Background(), ListParams{
		Params:     pagination.Params{Limit: 50},
		EntityType: "org",
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 1)
}
