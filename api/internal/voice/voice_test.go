package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func createTestOrg(t *testing.T, db *gorm.DB) *models.Org {
	t.Helper()
	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org
}

// --- Provider Interface Tests ---

func TestStubProviderImplementsInterface(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	var _ VoiceProvider = provider
}

// --- StubProvider.LogCall Tests ---

func TestStubProvider_LogCall_Success(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:     org.ID,
		CallerID:  "user_123",
		Direction: models.CallDirectionInbound,
		Duration:  120,
		Status:    models.CallStatusCompleted,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, callLog.ID)
	assert.Equal(t, org.ID, callLog.OrgID)
	assert.Equal(t, "user_123", callLog.CallerID)
	assert.Equal(t, models.CallDirectionInbound, callLog.Direction)
	assert.Equal(t, 120, callLog.Duration)
	assert.Equal(t, models.CallStatusCompleted, callLog.Status)
}

func TestStubProvider_LogCall_OutboundDirection(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:     org.ID,
		CallerID:  "user_456",
		Direction: models.CallDirectionOutbound,
		Duration:  60,
		Status:    models.CallStatusCompleted,
	})

	require.NoError(t, err)
	assert.Equal(t, models.CallDirectionOutbound, callLog.Direction)
}

func TestStubProvider_LogCall_DefaultsInvalidDirection(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_789",
	})

	require.NoError(t, err)
	assert.Equal(t, models.CallDirectionInbound, callLog.Direction)
	assert.Equal(t, models.CallStatusCompleted, callLog.Status)
}

func TestStubProvider_LogCall_MissingCallerID(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)

	_, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID: "some-org",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "caller_id is required")
}

func TestStubProvider_LogCall_MissingOrgID(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)

	_, err := provider.LogCall(context.Background(), LogCallInput{
		CallerID: "user_123",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "org_id is required")
}

func TestStubProvider_LogCall_WithMetadata(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_meta",
		Metadata: `{"topic":"billing"}`,
	})

	require.NoError(t, err)
	assert.Equal(t, `{"topic":"billing"}`, callLog.Metadata)
}

func TestStubProvider_LogCall_DefaultMetadata(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_empty",
	})

	require.NoError(t, err)
	assert.Equal(t, "{}", callLog.Metadata)
}

// --- StubProvider.GetTranscript Tests ---

func TestStubProvider_GetTranscript_StubResponse(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_123",
	})
	require.NoError(t, err)

	transcript, err := provider.GetTranscript(context.Background(), callLog.ID)
	require.NoError(t, err)
	assert.Contains(t, transcript, "[stub]")
}

func TestStubProvider_GetTranscript_WithRealTranscript(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_123",
	})
	require.NoError(t, err)

	// Set a real transcript.
	db.Model(&callLog).Update("transcript", "Hello, how can I help you?")

	transcript, err := provider.GetTranscript(context.Background(), callLog.ID)
	require.NoError(t, err)
	assert.Equal(t, "Hello, how can I help you?", transcript)
}

func TestStubProvider_GetTranscript_NotFound(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)

	_, err := provider.GetTranscript(context.Background(), "nonexistent-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "call not found")
}

// --- StubProvider.Escalate Tests ---

func TestStubProvider_Escalate_Success(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_123",
		Status:   models.CallStatusActive,
	})
	require.NoError(t, err)

	result, err := provider.Escalate(context.Background(), callLog.ID)
	require.NoError(t, err)
	assert.Equal(t, callLog.ID, result.CallID)
	assert.Equal(t, "escalated", result.Status)
	assert.Contains(t, result.Message, "[stub]")

	// Verify status updated in DB.
	var updated models.CallLog
	require.NoError(t, db.Where("id = ?", callLog.ID).First(&updated).Error)
	assert.Equal(t, models.CallStatusEscalated, updated.Status)
}

func TestStubProvider_Escalate_NotFound(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)

	_, err := provider.Escalate(context.Background(), "nonexistent-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "call not found")
}

// --- Handler Tests ---

func TestHandler_LogCall_Success(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	body := CreateCallRequest{
		CallerID:  "user_handler",
		Direction: models.CallDirectionInbound,
		Duration:  90,
		Status:    models.CallStatusCompleted,
	}
	bodyJSON, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/calls", handler.LogCall)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/calls", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.CallLog
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "user_handler", resp.CallerID)
}

func TestHandler_LogCall_InvalidBody(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/calls", handler.LogCall)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/test-org/calls", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_LogCall_MissingCallerID(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	body := CreateCallRequest{Duration: 90}
	bodyJSON, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/calls", handler.LogCall)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/test-org/calls", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTranscript_Success(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_transcript",
	})
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/v1/orgs/{org}/calls/{call}", handler.GetTranscript)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/calls/"+callLog.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, callLog.ID, resp["call_id"])
	assert.Contains(t, resp["transcript"], "[stub]")
}

func TestHandler_GetTranscript_NotFound(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	r := chi.NewRouter()
	r.Get("/v1/orgs/{org}/calls/{call}", handler.GetTranscript)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/test-org/calls/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Escalate_Success(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	callLog, err := provider.LogCall(context.Background(), LogCallInput{
		OrgID:    org.ID,
		CallerID: "user_escalate",
		Status:   models.CallStatusActive,
	})
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/calls/{call}/escalate", handler.Escalate)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/calls/"+callLog.ID+"/escalate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp EscalateResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "escalated", resp.Status)
}

func TestHandler_Escalate_NotFound(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/calls/{call}/escalate", handler.Escalate)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/test-org/calls/nonexistent/escalate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Handler Edge Case Tests ---

func TestHandler_LogCall_EmptyOrgParam(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	// Call directly without chi router — URLParam returns empty.
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs//calls", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.LogCall(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTranscript_EmptyCallParam(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/test-org/calls/", nil)
	w := httptest.NewRecorder()
	handler.GetTranscript(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Escalate_EmptyCallParam(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/test-org/calls//escalate", nil)
	w := httptest.NewRecorder()
	handler.Escalate(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_LogCall_ProviderError(t *testing.T) {
	db := testDB(t)
	provider := NewStubProvider(db)
	handler := NewHandler(provider)

	// Use a non-existent org to trigger a FK constraint error.
	body := CreateCallRequest{
		CallerID:  "user_err",
		Direction: models.CallDirectionInbound,
	}
	bodyJSON, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/calls", handler.LogCall)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/nonexistent-org/calls", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should be 500 because the org FK constraint fails.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStubProvider_LogCall_AllStatuses(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	statuses := []models.CallStatus{
		models.CallStatusRinging,
		models.CallStatusActive,
		models.CallStatusFailed,
		models.CallStatusEscalated,
	}
	for _, s := range statuses {
		cl, err := provider.LogCall(context.Background(), LogCallInput{
			OrgID:    org.ID,
			CallerID: "user-" + string(s),
			Status:   s,
		})
		require.NoError(t, err)
		assert.Equal(t, s, cl.Status)
	}
}

func TestStubProvider_LogCall_AllDirections(t *testing.T) {
	db := testDB(t)
	org := createTestOrg(t, db)
	provider := NewStubProvider(db)

	for _, d := range []models.CallDirection{models.CallDirectionInbound, models.CallDirectionOutbound} {
		cl, err := provider.LogCall(context.Background(), LogCallInput{
			OrgID:     org.ID,
			CallerID:  "user-" + string(d),
			Direction: d,
		})
		require.NoError(t, err)
		assert.Equal(t, d, cl.Direction)
	}
}

// --- Model Validation Tests ---

func TestCallDirection_IsValid(t *testing.T) {
	assert.True(t, models.CallDirectionInbound.IsValid())
	assert.True(t, models.CallDirectionOutbound.IsValid())
	assert.False(t, models.CallDirection("unknown").IsValid())
}

func TestCallStatus_IsValid(t *testing.T) {
	assert.True(t, models.CallStatusRinging.IsValid())
	assert.True(t, models.CallStatusActive.IsValid())
	assert.True(t, models.CallStatusCompleted.IsValid())
	assert.True(t, models.CallStatusFailed.IsValid())
	assert.True(t, models.CallStatusEscalated.IsValid())
	assert.False(t, models.CallStatus("unknown").IsValid())
}

// --- Fuzz Tests ---

func FuzzLogCallInput(f *testing.F) {
	f.Add("user_123", "inbound", 120, "completed", "{}")
	f.Add("", "outbound", 0, "active", "")
	f.Add("x", "invalid", -1, "", `{"key":"val"}`)
	f.Add("user_with_special_<chars>&", "inbound", 999999, "ringing", `{"a":1}`)

	db := func() *gorm.DB {
		dir := f.TempDir()
		dbPath := filepath.Join(dir, "fuzz.db")
		db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			f.Fatal(err)
		}
		if err := database.Migrate(db); err != nil {
			f.Fatal(err)
		}
		org := &models.Org{Name: "Fuzz Org", Slug: "fuzz-org", Metadata: "{}"}
		if err := db.Create(org).Error; err != nil {
			f.Fatal(err)
		}
		return db
	}()

	var orgID string
	db.Model(&models.Org{}).Select("id").First(&orgID)

	provider := NewStubProvider(db)

	f.Fuzz(func(t *testing.T, callerID, direction string, duration int, status, metadata string) {
		// Should not panic regardless of input.
		_, _ = provider.LogCall(context.Background(), LogCallInput{
			OrgID:     orgID,
			CallerID:  callerID,
			Direction: models.CallDirection(direction),
			Duration:  duration,
			Status:    models.CallStatus(status),
			Metadata:  metadata,
		})
	})
}
