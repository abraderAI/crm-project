package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
)

// testKeyPair holds an RSA key pair for testing.
type testKeyPair struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
	Kid     string
}

// generateTestKeyPair creates a test RSA key pair.
func generateTestKeyPair(t *testing.T) *testKeyPair {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return &testKeyPair{
		Private: key,
		Public:  &key.PublicKey,
		Kid:     "test-kid-1",
	}
}

// signTestJWT creates a signed JWT token for testing.
func signTestJWT(t *testing.T, kp *testKeyPair, claims JWTClaims) string {
	t.Helper()

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kp.Kid,
	}
	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)

	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, kp.Private, crypto.SHA256, hash[:])
	require.NoError(t, err)

	return signingInput + "." + base64URLEncode(sig)
}

// base64URLEncode encodes bytes to base64url without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// serveJWKS starts an HTTP server that serves a JWKS endpoint with the given public key.
func serveJWKS(t *testing.T, kp *testKeyPair) *httptest.Server {
	t.Helper()

	nB64 := base64URLEncode(kp.Public.N.Bytes())
	eB64 := base64URLEncode(big.NewInt(int64(kp.Public.E)).Bytes())

	jwks := map[string]interface{}{
		"keys": []map[string]string{
			{
				"kid": kp.Kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   nB64,
				"e":   eB64,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// testDB creates a fresh SQLite DB with migrations for testing.
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

// testValidClaims returns valid JWT claims for testing.
func testValidClaims(issuer string) JWTClaims {
	return JWTClaims{
		Subject:   "user_test123",
		Issuer:    issuer,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
	}
}

// testExpiredClaims returns expired JWT claims for testing.
func testExpiredClaims(issuer string) JWTClaims {
	return JWTClaims{
		Subject:   "user_expired",
		Issuer:    issuer,
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
	}
}

// testWrongIssuerClaims returns claims with a wrong issuer.
func testWrongIssuerClaims() JWTClaims {
	return JWTClaims{
		Subject:   "user_wrongissuer",
		Issuer:    "https://wrong-issuer.example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}
}

// testFutureClaims returns claims that are not yet valid.
func testFutureClaims(issuer string) JWTClaims {
	return JWTClaims{
		Subject:   "user_future",
		Issuer:    issuer,
		ExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
		NotBefore: time.Now().Add(1 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}
}

// testValidator creates a JWTValidator with pre-loaded test keys.
func testValidator(t *testing.T, kp *testKeyPair, issuerURL string) *JWTValidator {
	t.Helper()
	v := NewJWTValidator(issuerURL)
	v.SetKeys(map[string]*rsa.PublicKey{kp.Kid: kp.Public})
	return v
}

// testValidatorWithJWKS creates a validator backed by a real JWKS endpoint.
func testValidatorWithJWKS(t *testing.T, kp *testKeyPair) (*JWTValidator, string) {
	t.Helper()
	srv := serveJWKS(t, kp)
	v := NewJWTValidator(srv.URL)
	return v, srv.URL
}
