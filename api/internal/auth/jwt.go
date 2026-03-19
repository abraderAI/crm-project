package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Common JWT validation errors.
var (
	ErrTokenMissing     = errors.New("authorization token is required")
	ErrTokenMalformed   = errors.New("token is malformed")
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenNotYet      = errors.New("token is not yet valid")
	ErrTokenIssuer      = errors.New("invalid token issuer")
	ErrTokenSignature   = errors.New("invalid token signature")
	ErrTokenKeyNotFound = errors.New("signing key not found")
)

// JWTClaims represents the validated JWT claims.
type JWTClaims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
	// User identity claims populated by Clerk.
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// jwksKey represents a single key from JWKS response.
type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwksResponse represents the JWKS endpoint response.
type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// JWTValidator validates Clerk JWTs using JWKS.
type JWTValidator struct {
	issuerURL  string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	cacheTTL  time.Duration
}

// NewJWTValidator creates a JWT validator for the given Clerk issuer URL.
func NewJWTValidator(issuerURL string) *JWTValidator {
	return &JWTValidator{
		issuerURL:  strings.TrimRight(issuerURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
		cacheTTL:   1 * time.Hour,
	}
}

// SetKeys allows directly setting public keys (used for testing).
func (v *JWTValidator) SetKeys(keys map[string]*rsa.PublicKey) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = keys
	v.fetchedAt = time.Now()
}

// Validate parses and validates a JWT token string, returning claims on success.
func (v *JWTValidator) Validate(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrTokenMalformed
	}

	// Decode header.
	headerBytes, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrTokenMalformed
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrTokenMalformed, header.Alg)
	}

	// Decode payload.
	payloadBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrTokenMalformed
	}

	// Validate time claims.
	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return nil, ErrTokenNotYet
	}

	// Validate issuer.
	if v.issuerURL != "" && claims.Issuer != v.issuerURL {
		return nil, ErrTokenIssuer
	}

	// Get signing key.
	key, err := v.getKey(header.Kid)
	if err != nil {
		return nil, err
	}

	// Verify signature.
	if err := verifyRS256(parts[0]+"."+parts[1], parts[2], key); err != nil {
		return nil, ErrTokenSignature
	}

	return &claims, nil
}

// getKey retrieves the RSA public key for the given kid, fetching JWKS if needed.
func (v *JWTValidator) getKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	stale := time.Since(v.fetchedAt) > v.cacheTTL
	v.mu.RUnlock()

	if ok && !stale {
		return key, nil
	}

	// Fetch fresh JWKS.
	if err := v.fetchJWKS(); err != nil {
		// If fetch fails but we have a cached key, use it.
		if ok {
			return key, nil
		}
		return nil, fmt.Errorf("fetching JWKS: %w", err)
	}

	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, ErrTokenKeyNotFound
	}
	return key, nil
}

// fetchJWKS fetches the JWKS from the issuer's well-known endpoint.
func (v *JWTValidator) fetchJWKS() error {
	url := v.issuerURL + "/.well-known/jwks.json"
	resp, err := v.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request to JWKS endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return fmt.Errorf("reading JWKS response: %w", err)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("parsing JWKS response: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}
		pub, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		newKeys[k.Kid] = pub
	}

	v.mu.Lock()
	v.keys = newKeys
	v.fetchedAt = time.Now()
	v.mu.Unlock()

	return nil
}

// parseRSAPublicKey constructs an RSA public key from base64url-encoded n and e.
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64URLDecode(nStr)
	if err != nil {
		return nil, fmt.Errorf("decoding modulus: %w", err)
	}
	eBytes, err := base64URLDecode(eStr)
	if err != nil {
		return nil, fmt.Errorf("decoding exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() {
		return nil, fmt.Errorf("exponent too large")
	}

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// base64URLDecode decodes a base64url-encoded string (no padding).
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed.
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

// verifyRS256 verifies an RS256 signature.
func verifyRS256(signingInput, signatureB64 string, key *rsa.PublicKey) error {
	sigBytes, err := base64URLDecode(signatureB64)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}

	// Compute SHA-256 hash of signing input.
	hash := sha256.Sum256([]byte(signingInput))

	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sigBytes)
}
