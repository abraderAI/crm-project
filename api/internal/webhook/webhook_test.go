package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/event"
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

func TestSignPayload(t *testing.T) {
	sig := SignPayload("secret", "payload")
	assert.NotEmpty(t, sig)
	assert.Len(t, sig, 64) // SHA256 hex = 64 chars

	// Same input gives same output.
	sig2 := SignPayload("secret", "payload")
	assert.Equal(t, sig, sig2)

	// Different secret gives different output.
	sig3 := SignPayload("other", "payload")
	assert.NotEqual(t, sig, sig3)
}

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	payload := `{"event":"test"}`
	sig := SignPayload(secret, payload)

	assert.True(t, VerifySignature(secret, payload, sig))
	assert.False(t, VerifySignature(secret, payload, "wrong"))
	assert.False(t, VerifySignature("wrong-secret", payload, sig))
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name       string
		filterJSON string
		eventType  string
		want       bool
	}{
		{"empty filter matches all", "[]", "org.created", true},
		{"nil filter matches all", "", "org.created", true},
		{"matching filter", `["org.created","org.updated"]`, "org.created", true},
		{"non-matching filter", `["org.created"]`, "thread.created", false},
		{"invalid JSON matches all", "not-json", "anything", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesFilter(tt.filterJSON, tt.eventType))
		})
	}
}

func TestIsValidWebhookURL(t *testing.T) {
	assert.True(t, isValidWebhookURL("http://example.com/hook"))
	assert.True(t, isValidWebhookURL("https://example.com/hook"))
	assert.False(t, isValidWebhookURL("ftp://example.com"))
	assert.False(t, isValidWebhookURL("not-a-url"))
	assert.False(t, isValidWebhookURL(""))
}

func TestService_CreateSubscription(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "WH Org", Slug: "wh-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	sub, err := svc.Create(context.Background(), org.ID, CreateInput{
		URL:    "https://example.com/hook",
		Secret: "s3cret",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, org.ID, sub.OrgID)
	assert.True(t, sub.IsActive)
}

func TestService_CreateSubscription_Validation(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), "org1", CreateInput{URL: "", Secret: "s"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")

	_, err = svc.Create(context.Background(), "org1", CreateInput{URL: "not-valid", Secret: "s"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook URL")

	_, err = svc.Create(context.Background(), "org1", CreateInput{URL: "http://x.com", Secret: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret is required")
}

func TestService_ListSubscriptions(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "List Org", Slug: "list-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), org.ID, CreateInput{
			URL: "https://example.com/hook", Secret: "s3cret",
		})
		require.NoError(t, err)
	}

	subs, pageInfo, err := svc.List(context.Background(), org.ID, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, subs, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestService_GetSubscription(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "Get Org", Slug: "get-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "https://example.com/hook", Secret: "s3cret",
	})

	got, err := svc.Get(context.Background(), sub.ID)
	require.NoError(t, err)
	assert.Equal(t, sub.ID, got.ID)

	// Not found.
	got, err = svc.Get(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_DeleteSubscription(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "Del Org", Slug: "del-wh-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "https://example.com/hook", Secret: "s3cret",
	})

	err := svc.Delete(context.Background(), sub.ID)
	require.NoError(t, err)

	err = svc.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestService_ListDeliveries(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "Delivery Org", Slug: "dlv-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "https://example.com/hook", Secret: "s3cret",
	})

	// Create delivery records manually.
	for i := 0; i < 2; i++ {
		dlv := &models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventType:      "org.created",
			Payload:        `{"test":true}`,
			StatusCode:     200,
			Attempts:       1,
		}
		require.NoError(t, db.Create(dlv).Error)
	}

	deliveries, pageInfo, err := svc.ListDeliveries(context.Background(), sub.ID, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, deliveries, 2)
	assert.False(t, pageInfo.HasMore)
}

func TestService_Replay(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "Replay Org", Slug: "replay-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	// Start a test HTTP server to receive webhook.
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer receiver.Close()

	svc := NewService(db)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: receiver.URL, Secret: "replay-secret",
	})

	// Create a delivery record.
	dlv := &models.WebhookDelivery{
		SubscriptionID: sub.ID,
		EventType:      "org.created",
		Payload:        `{"id":"test"}`,
		StatusCode:     500,
		Attempts:       1,
	}
	require.NoError(t, db.Create(dlv).Error)

	// Replay.
	replayed, err := svc.Replay(context.Background(), dlv.ID)
	require.NoError(t, err)
	assert.Equal(t, 200, replayed.StatusCode)
	assert.Equal(t, 2, replayed.Attempts)

	// Replay not found.
	_, err = svc.Replay(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestService_HandleEvent(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "Event Org", Slug: "event-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	svc := NewService(db)
	_, err := svc.Create(context.Background(), org.ID, CreateInput{
		URL: receiver.URL, Secret: "event-secret",
	})
	require.NoError(t, err)

	// HandleEvent is async; call it and wait briefly.
	svc.HandleEvent(event.Event{
		Type:       "org.created",
		OrgID:      org.ID,
		EntityType: "org",
		EntityID:   "test-entity-id",
	})

	// Give goroutine time to deliver.
	time.Sleep(2 * time.Second)
}

func TestHandler_Create_Webhook(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "HC Org", Slug: "hc-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	h := NewHandler(svc)

	body := `{"url":"http://example.com/hook","secret":"s3cret"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", org.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_BadBody(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/org1/webhooks", strings.NewReader("not json"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", "org1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List_Webhook(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "HL Org", Slug: "hl-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/webhooks", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", org.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_Webhook(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "HGet Org", Slug: "hget-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	h := NewHandler(svc)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "http://example.com/hook", Secret: "s3cret",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/webhooks/"+sub.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", org.ID)
	rctx.URLParams.Add("id", sub.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Not found.
	req = httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/webhooks/nonexistent", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Delete_Webhook(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "HDel Org", Slug: "hdel-wh-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	h := NewHandler(svc)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "http://example.com/hook", Secret: "s3cret",
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+org.ID+"/webhooks/"+sub.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", sub.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Delete not found.
	req = httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+org.ID+"/webhooks/nonexistent", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ListDeliveries(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "HLD Org", Slug: "hld-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	h := NewHandler(svc)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "http://example.com/hook", Secret: "s3cret",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/webhooks/"+sub.ID+"/deliveries", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", sub.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.ListDeliveries(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Replay(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "HR Org", Slug: "hr-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	svc := NewService(db)
	h := NewHandler(svc)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: receiver.URL, Secret: "replay-s",
	})

	dlv := &models.WebhookDelivery{
		SubscriptionID: sub.ID,
		EventType:      "org.created",
		Payload:        `{"id":"test"}`,
		StatusCode:     500,
		Attempts:       1,
	}
	require.NoError(t, db.Create(dlv).Error)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/webhooks/"+sub.ID+"/deliveries/"+dlv.ID+"/replay", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", sub.ID)
	rctx.URLParams.Add("deliveryID", dlv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Replay(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Replay not found.
	req = httptest.NewRequest(http.MethodPost, "/replay", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("deliveryID", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	h.Replay(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestService_ListPagination(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "LP Org", Slug: "lp-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	for i := 0; i < 5; i++ {
		_, err := svc.Create(context.Background(), org.ID, CreateInput{
			URL: "https://example.com/hook", Secret: "s3cret",
		})
		require.NoError(t, err)
	}

	// First page.
	subs, pageInfo, err := svc.List(context.Background(), org.ID, pagination.Params{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, subs, 2)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)

	// Second page with cursor.
	subs2, _, err := svc.List(context.Background(), org.ID, pagination.Params{Limit: 2, Cursor: pageInfo.NextCursor})
	require.NoError(t, err)
	assert.Len(t, subs2, 2)
}

func TestService_ListDeliveriesPagination(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "LDP Org", Slug: "ldp-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	sub, _ := svc.Create(context.Background(), org.ID, CreateInput{
		URL: "https://example.com/hook", Secret: "s3cret",
	})

	for i := 0; i < 5; i++ {
		dlv := &models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventType:      "org.created",
			Payload:        `{"test":true}`,
			StatusCode:     200,
			Attempts:       1,
		}
		require.NoError(t, db.Create(dlv).Error)
	}

	// First page.
	deliveries, pageInfo, err := svc.ListDeliveries(context.Background(), sub.ID, pagination.Params{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, deliveries, 2)
	assert.True(t, pageInfo.HasMore)

	// Second page with cursor.
	deliveries2, _, err := svc.ListDeliveries(context.Background(), sub.ID, pagination.Params{Limit: 2, Cursor: pageInfo.NextCursor})
	require.NoError(t, err)
	assert.Len(t, deliveries2, 2)
}

func TestService_CreateWithEventFilter(t *testing.T) {
	db := setupTestDB(t)
	org := &models.Org{Name: "EF Org", Slug: "ef-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db)
	sub, err := svc.Create(context.Background(), org.ID, CreateInput{
		URL:         "http://example.com/hook",
		Secret:      "s3cret",
		EventFilter: []string{"org.created", "org.updated"},
	})
	require.NoError(t, err)
	assert.Contains(t, sub.EventFilter, "org.created")
}
