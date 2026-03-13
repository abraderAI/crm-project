package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

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
