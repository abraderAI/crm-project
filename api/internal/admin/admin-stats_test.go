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
)

// --- Platform Stats Tests ---

func TestService_GetPlatformStats(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create test data.
	now := time.Now()
	db.Create(&models.Org{Name: "Org1", Slug: "org1", Metadata: "{}"})
	db.Create(&models.Org{Name: "Org2", Slug: "org2", Metadata: "{}"})
	db.Create(&models.UserShadow{ClerkUserID: "u1", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u2", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u3", LastSeenAt: now, SyncedAt: now})

	stats, err := svc.GetPlatformStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Orgs.Total)
	assert.Equal(t, int64(3), stats.Users.Total)
	assert.True(t, stats.DBSizeBytes > 0)
	assert.Equal(t, 100.0, stats.ApiUptimePct) // No deliveries → 100%.
}

func TestService_GetPlatformStats_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	stats, err := svc.GetPlatformStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.Orgs.Total)
	assert.Equal(t, int64(0), stats.Users.Total)
	assert.True(t, stats.DBSizeBytes > 0)      // DB always has some pages.
	assert.Equal(t, 100.0, stats.ApiUptimePct) // No deliveries → 100%.
}

func TestHandler_GetStats(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	db.Create(&models.Org{Name: "StatOrg", Slug: "stat-org", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/stats", nil)
	h.GetStats(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "orgs")
	assert.Contains(t, w.Body.String(), "db_size_bytes")
	assert.Contains(t, w.Body.String(), "api_uptime_pct")
}

func TestHandler_GetStats_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/stats", nil)
	h.GetStats(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestService_GetPlatformStats_WithWebhooksAndNotifications(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create org + subscription + deliveries.
	org := models.Org{Name: "SO", Slug: "so", Metadata: "{}"}
	db.Create(&org)
	sub := models.WebhookSubscription{
		OrgID: org.ID, URL: "http://example.com", Secret: "sec",
		ScopeType: "org", ScopeID: org.ID, EventFilter: "[]",
	}
	db.Create(&sub)
	db.Create(&models.WebhookDelivery{
		SubscriptionID: sub.ID, EventType: "test", Payload: "{}", StatusCode: 500,
	})
	db.Create(&models.WebhookDelivery{
		SubscriptionID: sub.ID, EventType: "test", Payload: "{}", StatusCode: 200,
	})

	// Create unread notification.
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "nu", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.Notification{UserID: "nu", Type: "mention", Title: "Test", IsRead: false})

	stats, err := svc.GetPlatformStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.FailedWebhooks24h)
	assert.Equal(t, int64(1), stats.PendingNotifications)
	// 2 total deliveries, 1 failed → 50% uptime.
	assert.Equal(t, 50.0, stats.ApiUptimePct)
}
