package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
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

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
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

	// Run migrations.
	require.NoError(t, database.Migrate(db))

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

// --- Phase 2 Live API Tests ---

// TestLive_Phase2_MigrationsRan verifies that after server startup with migrations,
// the health check returns 200 and the database is healthy.
func TestLive_Phase2_MigrationsRan(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	// Readyz returns 200 with healthy database.
	resp, err := http.Get(baseURL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	checks := body["checks"].(map[string]interface{})
	assert.Equal(t, "ok", checks["database"])
}

// TestLive_Phase2_HealthAfterMigrations verifies health endpoint still works
// after full model migrations.
func TestLive_Phase2_HealthAfterMigrations(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

// TestLive_Phase2_RFC7807StillWorks verifies error handling still returns RFC 7807
// after Phase 2 migrations.
func TestLive_Phase2_RFC7807StillWorks(t *testing.T) {
	baseURL, cleanup := liveServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/v1/nonexistent-phase2-path")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Equal(t, 404, problem.Status)
}

// --- Phase 3 Live API Tests ---

// liveAuthServer starts a server with mock JWKS for auth testing.
// Returns baseURL, a function to sign test JWTs, and cleanup.
type liveAuthEnv struct {
	BaseURL   string
	IssuerURL string
	DB        *gorm.DB
	SignToken func(claims auth.JWTClaims) string
	Cleanup   func()
}

func liveAuthServer(t *testing.T) *liveAuthEnv {
	t.Helper()

	// Generate test RSA key pair.
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubKey := &privKey.PublicKey
	kid := "live-test-kid"

	// Start mock JWKS server.
	nB64 := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
	eB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes())
	jwks := map[string]interface{}{
		"keys": []map[string]string{{
			"kid": kid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": nB64, "e": eB64,
		}},
	}
	jwksMux := http.NewServeMux()
	jwksMux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})
	jwksSrv := httptest.NewServer(jwksMux)

	// Database setup.
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

	testLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	router := NewRouter(Config{
		DB:          db,
		Logger:      testLogger,
		CORSOrigins: []string{"http://localhost:3000"},
		IssuerURL:   jwksSrv.URL,
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(listener) }()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())
	for i := 0; i < 50; i++ {
		if resp, err := http.Get(baseURL + "/healthz"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	signToken := func(claims auth.JWTClaims) string {
		header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
		headerJSON, _ := json.Marshal(header)
		claimsJSON, _ := json.Marshal(claims)
		hB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
		cB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
		input := hB64 + "." + cB64
		h := sha256.Sum256([]byte(input))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, h[:])
		return input + "." + base64.RawURLEncoding.EncodeToString(sig)
	}

	return &liveAuthEnv{
		BaseURL:   baseURL,
		IssuerURL: jwksSrv.URL,
		DB:        db,
		SignToken: signToken,
		Cleanup: func() {
			srv.Close()
			jwksSrv.Close()
			sqlDB.Close()
		},
	}
}

// TestLive_Phase3_AuthRequired verifies authenticated endpoints return 401
// when no auth is provided.
func TestLive_Phase3_AuthRequired(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create an org for the API key route.
	org := &models.Org{Name: "Auth Org", Slug: "auth-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	resp, err := http.Get(env.BaseURL + "/v1/orgs/" + org.ID + "/api-keys")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Equal(t, 401, problem.Status)
}

// TestLive_Phase3_ValidJWT verifies a valid JWT grants access.
func TestLive_Phase3_ValidJWT(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "JWT Org", Slug: "jwt-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_jwt_test",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestLive_Phase3_ExpiredJWT verifies an expired JWT returns 401.
func TestLive_Phase3_ExpiredJWT(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Exp Org", Slug: "exp-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_expired",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	})

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase3_MalformedJWT verifies a malformed JWT returns 401.
func TestLive_Phase3_MalformedJWT(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Mal Org", Slug: "mal-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer totally.not.valid")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase3_APIKeyAuth verifies API key authentication works end-to-end.
func TestLive_Phase3_APIKeyAuth(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "AK Org", Slug: "ak-live-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	// First, create an API key using JWT auth.
	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_ak_test",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	createBody := `{"name":"Live Test Key"}`
	createReq, err := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", strings.NewReader(createBody))
	require.NoError(t, err)
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	defer createResp.Body.Close()
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	var keyResult struct {
		Key  string `json:"key"`
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&keyResult))
	assert.True(t, strings.HasPrefix(keyResult.Key, "deft_live_"))

	// Now use the API key to authenticate.
	listReq, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
	require.NoError(t, err)
	listReq.Header.Set("X-API-Key", keyResult.Key)

	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var listBody map[string]interface{}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listBody))
	data := listBody["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 1)
}

// TestLive_Phase3_InvalidAPIKey verifies invalid API keys return 401.
func TestLive_Phase3_InvalidAPIKey(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Bad AK Org", Slug: "bad-ak-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", "deft_live_invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalid")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase3_APIKeyRevoke verifies API key revocation works.
func TestLive_Phase3_APIKeyRevoke(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "Rev Org", Slug: "rev-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_revoke_test",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// Create a key.
	createReq, err := http.NewRequest(http.MethodPost, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys",
		strings.NewReader(`{"name":"ToRevoke"}`))
	require.NoError(t, err)
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	defer createResp.Body.Close()
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	var keyResult struct {
		Key string `json:"key"`
		ID  string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&keyResult))

	// Revoke the key.
	delReq, err := http.NewRequest(http.MethodDelete, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys/"+keyResult.ID, nil)
	require.NoError(t, err)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// Revoked key should not work.
	useReq, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
	require.NoError(t, err)
	useReq.Header.Set("X-API-Key", keyResult.Key)
	useResp, err := http.DefaultClient.Do(useReq)
	require.NoError(t, err)
	defer useResp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, useResp.StatusCode)
}

// TestLive_Phase3_CORSWithAuth verifies CORS preflight works with Authorization header.
func TestLive_Phase3_CORSWithAuth(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	req, err := http.NewRequest(http.MethodOptions, env.BaseURL+"/v1/orgs/test/api-keys", nil)
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
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "X-API-Key")
}

// TestLive_Phase3_HealthStillWorks verifies health endpoints remain unprotected.
func TestLive_Phase3_HealthStillWorks(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Health should work without auth.
	resp, err := http.Get(env.BaseURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// V1 root should work without auth.
	resp2, err := http.Get(env.BaseURL + "/v1/")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}

// --- Phase 4 Live API Tests ---

// doJSON is a helper for authenticated JSON requests.
func doJSON(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// decodeBody decodes a JSON response into a map.
func decodeBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

// TestLive_Phase4_HierarchyLifecycle tests the full Org → Space → Board → Thread → Message lifecycle.
func TestLive_Phase4_HierarchyLifecycle(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_phase4",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// 1. Create Org.
	resp := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs", `{"name":"Phase4 Org","metadata":"{\"billing_tier\":\"pro\"}"}`, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	orgBody := decodeBody(t, resp)
	orgID := orgBody["id"].(string)
	orgSlug := orgBody["slug"].(string)
	assert.NotEmpty(t, orgID)
	assert.Equal(t, "phase4-org", orgSlug)

	// 2. Get Org by slug.
	resp2 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs/"+orgSlug, "", token)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// 3. List Orgs.
	resp3 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs", "", token)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	listBody := decodeBody(t, resp3)
	data := listBody["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 1)

	// 4. Update Org.
	resp4 := doJSON(t, http.MethodPatch, env.BaseURL+"/v1/orgs/"+orgID, `{"description":"updated"}`, token)
	defer resp4.Body.Close()
	assert.Equal(t, http.StatusOK, resp4.StatusCode)

	// 5. Create Space.
	resp5 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Support Space","type":"support"}`, token)
	defer resp5.Body.Close()
	assert.Equal(t, http.StatusCreated, resp5.StatusCode)
	spaceBody := decodeBody(t, resp5)
	spaceID := spaceBody["id"].(string)

	// 6. Get Space by slug.
	resp6 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs/"+orgSlug+"/spaces/support-space", "", token)
	defer resp6.Body.Close()
	assert.Equal(t, http.StatusOK, resp6.StatusCode)

	// 7. Create Board.
	resp7 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"General Board"}`, token)
	defer resp7.Body.Close()
	assert.Equal(t, http.StatusCreated, resp7.StatusCode)
	boardBody := decodeBody(t, resp7)
	boardID := boardBody["id"].(string)

	// 8. Lock Board.
	resp8 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/lock", "", token)
	defer resp8.Body.Close()
	assert.Equal(t, http.StatusOK, resp8.StatusCode)
	lockBody := decodeBody(t, resp8)
	assert.Equal(t, true, lockBody["is_locked"])

	// 9. Try creating thread on locked board → should fail 409.
	resp9 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads", `{"title":"Blocked"}`, token)
	defer resp9.Body.Close()
	assert.Equal(t, http.StatusConflict, resp9.StatusCode)

	// 10. Unlock Board.
	resp10 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/unlock", "", token)
	defer resp10.Body.Close()
	assert.Equal(t, http.StatusOK, resp10.StatusCode)

	// 11. Create Thread.
	resp11 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Bug Report","body":"Steps to reproduce...","metadata":"{\"status\":\"open\",\"priority\":\"high\"}"}`, token)
	defer resp11.Body.Close()
	assert.Equal(t, http.StatusCreated, resp11.StatusCode)
	threadBody := decodeBody(t, resp11)
	threadID := threadBody["id"].(string)

	// 12. Pin Thread.
	resp12 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/pin", "", token)
	defer resp12.Body.Close()
	assert.Equal(t, http.StatusOK, resp12.StatusCode)
	pinBody := decodeBody(t, resp12)
	assert.Equal(t, true, pinBody["is_pinned"])

	// 13. Create Message.
	resp13 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages",
		`{"body":"I can reproduce this.","type":"comment"}`, token)
	defer resp13.Body.Close()
	assert.Equal(t, http.StatusCreated, resp13.StatusCode)
	msgBody := decodeBody(t, resp13)
	msgID := msgBody["id"].(string)

	// 14. Update Message (author-only).
	resp14 := doJSON(t, http.MethodPatch, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages/"+msgID,
		`{"body":"Updated: I can still reproduce this."}`, token)
	defer resp14.Body.Close()
	assert.Equal(t, http.StatusOK, resp14.StatusCode)

	// 15. List Messages.
	resp15 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages", "", token)
	defer resp15.Body.Close()
	assert.Equal(t, http.StatusOK, resp15.StatusCode)

	// 16. Lock Thread.
	resp16 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/lock", "", token)
	defer resp16.Body.Close()
	assert.Equal(t, http.StatusOK, resp16.StatusCode)

	// 17. Try creating message on locked thread → should fail 409.
	resp17 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages",
		`{"body":"Should fail"}`, token)
	defer resp17.Body.Close()
	assert.Equal(t, http.StatusConflict, resp17.StatusCode)

	// 18. Delete Message.
	resp18 := doJSON(t, http.MethodDelete, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages/"+msgID, "", token)
	defer resp18.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp18.StatusCode)

	// 19. Unlock and Delete Thread.
	resp19 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/unlock", "", token)
	defer resp19.Body.Close()
	assert.Equal(t, http.StatusOK, resp19.StatusCode)

	resp20 := doJSON(t, http.MethodDelete, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID, "", token)
	defer resp20.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp20.StatusCode)

	// 20. Delete Board.
	resp21 := doJSON(t, http.MethodDelete, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID, "", token)
	defer resp21.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp21.StatusCode)

	// 21. Delete Space.
	resp22 := doJSON(t, http.MethodDelete, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID, "", token)
	defer resp22.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp22.StatusCode)

	// 22. Delete Org.
	resp23 := doJSON(t, http.MethodDelete, env.BaseURL+"/v1/orgs/"+orgID, "", token)
	defer resp23.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp23.StatusCode)
}

// TestLive_Phase4_OrgMembership tests org membership management.
func TestLive_Phase4_OrgMembership(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_member",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// Create org.
	resp := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs", `{"name":"Member Org"}`, token)
	defer resp.Body.Close()
	orgBody := decodeBody(t, resp)
	orgID := orgBody["id"].(string)

	// Add member.
	resp2 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/members",
		`{"user_id":"newuser","role":"viewer"}`, token)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	// List members.
	resp3 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs/"+orgID+"/members", "", token)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
}

// TestLive_Phase4_MetadataFilter tests thread metadata filtering.
func TestLive_Phase4_MetadataFilter(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_filter",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// Create hierarchy.
	resp := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs", `{"name":"Filter Org"}`, token)
	defer resp.Body.Close()
	orgBody := decodeBody(t, resp)
	orgID := orgBody["id"].(string)

	resp2 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Filter Space"}`, token)
	defer resp2.Body.Close()
	spaceID := decodeBody(t, resp2)["id"].(string)

	resp3 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Filter Board"}`, token)
	defer resp3.Body.Close()
	boardID := decodeBody(t, resp3)["id"].(string)

	base := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads"

	// Create threads with different metadata.
	resp4 := doJSON(t, http.MethodPost, base, `{"title":"Open Bug","metadata":"{\"status\":\"open\"}"}`, token)
	defer resp4.Body.Close()
	assert.Equal(t, http.StatusCreated, resp4.StatusCode)

	resp5 := doJSON(t, http.MethodPost, base, `{"title":"Closed Bug","metadata":"{\"status\":\"closed\"}"}`, token)
	defer resp5.Body.Close()
	assert.Equal(t, http.StatusCreated, resp5.StatusCode)

	// Filter by status=open.
	req, err := http.NewRequest(http.MethodGet, base+"?metadata[status]=open", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp6, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp6.Body.Close()
	assert.Equal(t, http.StatusOK, resp6.StatusCode)
	filterBody := decodeBody(t, resp6)
	threads := filterBody["data"].([]interface{})
	assert.Len(t, threads, 1)
}

// TestLive_Phase4_SlugURLs tests accessing resources by slug.
func TestLive_Phase4_SlugURLs(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_slug",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// Create hierarchy.
	resp := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs", `{"name":"Slug Org"}`, token)
	defer resp.Body.Close()
	orgID := decodeBody(t, resp)["id"].(string)

	resp2 := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Slug Space"}`, token)
	defer resp2.Body.Close()

	// Access space by slug.
	resp3 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs/slug-org/spaces/slug-space", "", token)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
}

// TestLive_Phase4_Pagination tests cursor-based pagination.
func TestLive_Phase4_Pagination(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_page",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	orgID := ""
	// Create 3 orgs.
	for i := 0; i < 3; i++ {
		resp := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs",
			fmt.Sprintf(`{"name":"Page Org %d"}`, i), token)
		defer resp.Body.Close()
		if i == 0 {
			orgID = decodeBody(t, resp)["id"].(string)
			_ = orgID
		}
	}

	// List with limit=2.
	resp2 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs?limit=2", "", token)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	pageBody := decodeBody(t, resp2)
	pageInfo := pageBody["page_info"].(map[string]interface{})
	assert.Equal(t, true, pageInfo["has_more"])
	assert.NotEmpty(t, pageInfo["next_cursor"])

	// Use cursor for next page.
	cursor := pageInfo["next_cursor"].(string)
	resp3 := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs?limit=2&cursor="+cursor, "", token)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
}

// TestLive_Phase4_NotFound tests 404 responses for missing resources.
func TestLive_Phase4_NotFound(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_404",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	resp := doJSON(t, http.MethodGet, env.BaseURL+"/v1/orgs/nonexistent-org", "", token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase4_ValidationErrors tests validation error responses.
func TestLive_Phase4_ValidationErrors(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	token := env.SignToken(auth.JWTClaims{
		Subject:   "user_val",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	// Create org with empty name → validation error.
	resp := doJSON(t, http.MethodPost, env.BaseURL+"/v1/orgs", `{"name":""}`, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

// TestLive_Phase3_RFC7807OnAuthFailure verifies all auth failures return RFC 7807.
func TestLive_Phase3_RFC7807OnAuthFailure(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := &models.Org{Name: "RFC Org", Slug: "rfc-org", Metadata: "{}"}
	require.NoError(t, env.DB.Create(org).Error)

	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"no auth", "", ""},
		{"empty bearer", "Authorization", "Bearer "},
		{"malformed bearer", "Authorization", "Bearer not.a.jwt"},
		{"bad api key", "X-API-Key", "deft_live_badkey"},
		{"wrong prefix", "X-API-Key", "wrong_prefix_key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/v1/orgs/"+org.ID+"/api-keys", nil)
			require.NoError(t, err)
			if tt.header != "" {
				req.Header.Set(tt.header, tt.value)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
			assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

			var problem apierrors.ProblemDetail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
			assert.Equal(t, 401, problem.Status)
		})
	}
}
