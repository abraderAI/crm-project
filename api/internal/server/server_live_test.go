package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// liveServer starts a real HTTP server on a random port and returns its base URL.
func liveServer(t *testing.T) (string, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign keys.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	testLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	router := NewRouter(Config{
		DB:          db,
		Logger:      testLogger,
		CORSOrigins: []string{"http://localhost:3000"},
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(listener) }()

	// Wait for server to start.
	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())
	for i := 0; i < 50; i++ {
		if resp, err := http.Get(baseURL + "/healthz"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cleanup := func() {
		srv.Close()
		sqlDB.Close()
	}

	return baseURL, cleanup
}

func TestLive_Healthz(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestLive_Readyz(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	checks := body["checks"].(map[string]interface{})
	assert.Equal(t, "ok", checks["database"])
}

func TestLive_NotFound_RFC7807(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Equal(t, "Not Found", problem.Title)
	assert.Equal(t, 404, problem.Status)
}

func TestLive_V1Root(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/v1/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "v1", body["version"])
}

func TestLive_CORS_Preflight(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodOptions, baseURL+"/v1/", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
}

func TestLive_CORS_DisallowedOrigin(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://evil.com")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestLive_RequestID_CustomHeader(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "my-test-id-123")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "my-test-id-123", resp.Header.Get("X-Request-ID"))
}

func TestLive_ContentType_Rejection(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/", strings.NewReader(`{"test":"data"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestLive_V1_NonexistentEndpoint(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/v1/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestLive_CORSHeaders_OnNormalRequest(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Expose-Headers"), "X-Request-ID")
}

func TestLive_RequestID_Generated(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp1, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp1.Body.Close()

	resp2, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp2.Body.Close()

	id1 := resp1.Header.Get("X-Request-ID")
	id2 := resp2.Header.Get("X-Request-ID")
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "each request should get a unique ID")
}

func TestLive_Healthz_ResponseBody(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"status":"ok"`)
}
