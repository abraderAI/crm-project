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

	"github.com/abraderAI/crm-project/api/internal/models"
)

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
