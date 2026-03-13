package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Audit Log Tests ---

func TestService_ListAuditLogs(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create some audit entries.
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&models.AuditLog{
			UserID:     "admin1",
			Action:     models.AuditActionCreate,
			EntityType: "user",
			EntityID:   "user" + string(rune('1'+i)),
		}).Error)
	}

	logs, pageInfo, err := svc.ListAuditLogs(ctx, AuditListParams{
		Params: pagination.Params{Limit: 50},
	})
	require.NoError(t, err)
	assert.Len(t, logs, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListAuditLogs_Filters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.AuditLog{
		UserID: "admin1", Action: "ban", EntityType: "user", EntityID: "u1",
	}).Error)
	require.NoError(t, db.Create(&models.AuditLog{
		UserID: "admin2", Action: "suspend", EntityType: "org", EntityID: "o1",
	}).Error)

	logs, _, err := svc.ListAuditLogs(ctx, AuditListParams{
		Params: pagination.Params{Limit: 50},
		Action: "ban",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	logs, _, err = svc.ListAuditLogs(ctx, AuditListParams{
		Params:     pagination.Params{Limit: 50},
		EntityType: "org",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	logs, _, err = svc.ListAuditLogs(ctx, AuditListParams{
		Params: pagination.Params{Limit: 50},
		UserID: "admin1",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestHandler_ListAuditLog(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.AuditLog{UserID: "admin", Action: "ban", EntityType: "user", EntityID: "u1"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-log", nil)
	h.ListAuditLog(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_ListAuditLog_DateFilters(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	now := time.Now()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-log?after="+now.Add(-24*time.Hour).Format(time.RFC3339)+"&before="+now.Add(time.Hour).Format(time.RFC3339)+"&action=create&entity_type=org&user=u1&ip=127.0.0.1", nil)
	h.ListAuditLog(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListAuditLog_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-log", nil)
	h.ListAuditLog(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
