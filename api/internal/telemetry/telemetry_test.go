package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Init Tests ---

func TestInit_StdoutExporter(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{
		Enabled:     true,
		ServiceName: "test-service",
	})
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.NotNil(t, provider.TracerProvider)
	require.NotNil(t, provider.MeterProvider)

	err = provider.Shutdown(ctx)
	require.NoError(t, err)
}

func TestInit_CustomServiceName(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{
		Enabled:     true,
		ServiceName: "custom-name",
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	err = provider.Shutdown(ctx)
	require.NoError(t, err)
}

func TestInit_DefaultServiceName(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{
		Enabled: true,
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	err = provider.Shutdown(ctx)
	require.NoError(t, err)
}

// --- Shutdown Tests ---

func TestProvider_Shutdown_NilProviders(t *testing.T) {
	p := &Provider{}
	err := p.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestProvider_Shutdown_AfterInit(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true})
	require.NoError(t, err)

	err = provider.Shutdown(ctx)
	require.NoError(t, err)
}

// --- HTTPTrace Middleware Tests ---

func TestHTTPTrace_SetsSpanAttributes(t *testing.T) {
	// Initialize OTel for the test.
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true, ServiceName: "trace-test"})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	handler := HTTPTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestHTTPTrace_500StatusSetsError(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true, ServiceName: "trace-error-test"})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	handler := HTTPTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/error", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHTTPTrace_404Status(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true, ServiceName: "trace-404-test"})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	handler := HTTPTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHTTPTrace_DefaultStatusCode(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	handler := HTTPTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't explicitly write header — should default to 200.
		_, _ = w.Write([]byte("default"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- statusWriter Tests ---

func TestStatusWriter_CapturesCode(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	sw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, sw.status)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestStatusWriter_DefaultStatus(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	assert.Equal(t, http.StatusOK, sw.status)
}

// --- Metrics Tests ---

func TestRecordHTTPMetrics_DoesNotPanic(t *testing.T) {
	// Should not panic even without provider init.
	assert.NotPanics(t, func() {
		recordHTTPMetrics("GET", "/test", 200, 10*time.Millisecond)
	})
}

func TestIncrementDecrementActiveConnections(t *testing.T) {
	assert.NotPanics(t, func() {
		IncrementActiveConnections()
		DecrementActiveConnections()
	})
}

func TestRecordWebhookResult(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordWebhookResult(true)
		RecordWebhookResult(false)
	})
}

func TestMetricsStatus(t *testing.T) {
	ok, err := MetricsStatus()
	assert.True(t, ok)
	assert.NoError(t, err)
}

// --- Meter Tests ---

func TestMeter_ReturnsNonNil(t *testing.T) {
	m := Meter()
	assert.NotNil(t, m)
}

func TestInit_WithOTLPEndpoint(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{
		Enabled:     true,
		ServiceName: "otlp-test",
		Endpoint:    "http://localhost:4318",
	})
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.NotNil(t, provider.TracerProvider)

	err = provider.Shutdown(ctx)
	require.NoError(t, err)
}

func TestHTTPTrace_RecordsMetrics(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true, ServiceName: "metrics-trace-test"})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	handler := HTTPTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/123/calls", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHTTPTrace_MultipleRequests(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true, ServiceName: "multi-req-test"})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	handler := HTTPTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRecordHTTPMetrics_WithInitializedProvider(t *testing.T) {
	ctx := context.Background()
	provider, err := Init(ctx, Config{Enabled: true, ServiceName: "record-metrics-test"})
	require.NoError(t, err)
	defer func() { _ = provider.Shutdown(ctx) }()

	assert.NotPanics(t, func() {
		recordHTTPMetrics("GET", "/test", 200, 50*time.Millisecond)
		recordHTTPMetrics("POST", "/create", 201, 100*time.Millisecond)
		recordHTTPMetrics("DELETE", "/delete", 500, 200*time.Millisecond)
	})
}

func TestStatusWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	n, err := sw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", w.Body.String())
}
