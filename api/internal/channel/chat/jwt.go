package chat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SessionTokenDuration is the lifetime of a chat session JWT.
const SessionTokenDuration = 24 * time.Hour

// SessionClaims holds the payload of a chat session JWT.
type SessionClaims struct {
	SessionID string `json:"session_id"`
	OrgID     string `json:"org_id"`
	VisitorID string `json:"visitor_id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// IssueSessionToken creates an HMAC-SHA256 signed JWT for the given session.
func IssueSessionToken(secret string, claims SessionClaims) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("signing secret is required")
	}
	if claims.SessionID == "" || claims.OrgID == "" || claims.VisitorID == "" {
		return "", fmt.Errorf("session_id, org_id, and visitor_id are required")
	}

	now := time.Now().Unix()
	claims.IssuedAt = now
	claims.ExpiresAt = now + int64(SessionTokenDuration.Seconds())

	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %w", err)
	}
	payload := base64URLEncode(payloadBytes)

	signingInput := header + "." + payload
	sig := signHMAC(secret, signingInput)

	return signingInput + "." + sig, nil
}

// ValidateSessionToken verifies and decodes a chat session JWT.
func ValidateSessionToken(secret, tokenString string) (*SessionClaims, error) {
	if secret == "" {
		return nil, fmt.Errorf("signing secret is required")
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}

	// Verify signature.
	signingInput := parts[0] + "." + parts[1]
	expectedSig := signHMAC(secret, signingInput)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Decode payload.
	payloadBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	var claims SessionClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("parsing claims: %w", err)
	}

	// Check expiry.
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

// signHMAC computes HMAC-SHA256 and returns base64url-encoded signature.
func signHMAC(secret, input string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64URLEncode(mac.Sum(nil))
}

// base64URLEncode encodes bytes to base64url without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes a base64url string (with or without padding).
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
