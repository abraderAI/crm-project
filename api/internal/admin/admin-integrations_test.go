package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Integration & Webhook Delivery Tests ---

func TestService_ListAllWebhookDeliveries_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	deliveries, pageInfo, err := svc.ListAllWebhookDeliveries(context.Background(), pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Empty(t, deliveries)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListAllWebhookDeliveries_WithData(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Create org and subscription (for FK).
	org := models.Org{Name: "WH Org", Slug: "wh-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	sub := models.WebhookSubscription{
		OrgID: org.ID, URL: "http://example.com", Secret: "sec",
		ScopeType: "org", ScopeID: org.ID, EventFilter: "[]",
	}
	require.NoError(t, db.Create(&sub).Error)

	for i := 0; i < 3; i++ {
		db.Create(&models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventType:      "test.event",
			Payload:        "{}",
			StatusCode:     200,
		})
	}

	deliveries, pageInfo, err := svc.ListAllWebhookDeliveries(ctx, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, deliveries, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestHandler_ListWebhookDeliveries(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/webhooks/deliveries", nil)
	h.ListWebhookDeliveries(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestService_GetIntegrationStatus_Unconfigured(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Ensure env vars are not set.
	t.Setenv("CLERK_SECRET_KEY", "")
	t.Setenv("RESEND_API_KEY", "")
	t.Setenv("FLEXPOINT_API_KEY", "")

	status := svc.GetIntegrationStatus(context.Background())
	assert.Equal(t, "unconfigured", status.Clerk)
	assert.Equal(t, "unconfigured", status.Resend)
	assert.Equal(t, "unconfigured", status.FlexPoint)
}

func TestService_GetIntegrationStatus_Configured(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	t.Setenv("CLERK_SECRET_KEY", "test_key")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("FLEXPOINT_API_KEY", "fp_test")

	status := svc.GetIntegrationStatus(context.Background())
	assert.Equal(t, "ok", status.Clerk)
	assert.Equal(t, "ok", status.Resend)
	assert.Equal(t, "ok", status.FlexPoint)
}

func TestHandler_GetIntegrationHealth(t *testing.T) {
	h, _, _ := setupTestHandlerWithPolicy(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/integrations/status", nil)
	h.GetIntegrationHealth(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "clerk")
	assert.Contains(t, w.Body.String(), "resend")
	assert.Contains(t, w.Body.String(), "flexpoint")
}

func TestService_ListAllWebhookDeliveries_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Pag Org", Slug: "pag-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	sub := models.WebhookSubscription{
		OrgID: org.ID, URL: "http://example.com", Secret: "sec",
		ScopeType: "org", ScopeID: org.ID, EventFilter: "[]",
	}
	require.NoError(t, db.Create(&sub).Error)

	for i := 0; i < 5; i++ {
		db.Create(&models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventType:      "test.event",
			Payload:        "{}",
			StatusCode:     200,
		})
	}

	// First page.
	deliveries, pageInfo, err := svc.ListAllWebhookDeliveries(ctx, pagination.Params{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, deliveries, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)

	// Second page.
	deliveries2, pageInfo2, err := svc.ListAllWebhookDeliveries(ctx, pagination.Params{Limit: 3, Cursor: pageInfo.NextCursor})
	require.NoError(t, err)
	assert.Len(t, deliveries2, 2)
	assert.False(t, pageInfo2.HasMore)
}

func TestHandler_ListWebhookDeliveries_Error(t *testing.T) {
	h, _, db := setupTestHandlerWithPolicy(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/webhooks/deliveries", nil)
	h.ListWebhookDeliveries(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
