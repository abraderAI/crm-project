package admin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

const (
	// impersonationSecret is used for HMAC signing impersonation tokens.
	// In production, this would come from config/env.
	impersonationSecret = "deft-impersonation-hmac-secret-v1"
	// maxImpersonationMinutes is the maximum allowed impersonation duration.
	maxImpersonationMinutes = 120
	// defaultImpersonationMinutes is the default impersonation duration.
	defaultImpersonationMinutes = 30
)

// ImpersonationToken represents the claims inside an impersonation token.
type ImpersonationToken struct {
	ImpersonatorID string    `json:"impersonator_id"`
	TargetUserID   string    `json:"target_user_id"`
	Reason         string    `json:"reason"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// ImpersonateUser creates a time-limited impersonation token. Returns an error
// if the target user is a platform admin.
func (s *Service) ImpersonateUser(ctx context.Context, impersonatorID, targetUserID, reason string, durationMinutes int) (*ImpersonationToken, string, error) {
	if targetUserID == "" {
		return nil, "", fmt.Errorf("target user_id is required")
	}
	if impersonatorID == "" {
		return nil, "", fmt.Errorf("impersonator_id is required")
	}
	if impersonatorID == targetUserID {
		return nil, "", fmt.Errorf("cannot impersonate yourself")
	}

	// Cannot impersonate another platform admin.
	isAdmin, err := s.IsPlatformAdmin(ctx, targetUserID)
	if err != nil {
		return nil, "", fmt.Errorf("checking admin status: %w", err)
	}
	if isAdmin {
		return nil, "", fmt.Errorf("cannot impersonate a platform admin")
	}

	// Clamp duration.
	if durationMinutes <= 0 {
		durationMinutes = defaultImpersonationMinutes
	}
	if durationMinutes > maxImpersonationMinutes {
		durationMinutes = maxImpersonationMinutes
	}

	token := &ImpersonationToken{
		ImpersonatorID: impersonatorID,
		TargetUserID:   targetUserID,
		Reason:         reason,
		ExpiresAt:      time.Now().Add(time.Duration(durationMinutes) * time.Minute),
	}

	signed, err := signImpersonationToken(token)
	if err != nil {
		return nil, "", fmt.Errorf("signing token: %w", err)
	}

	return token, signed, nil
}

// ValidateImpersonationToken validates and decodes an impersonation token string.
func ValidateImpersonationToken(tokenStr string) (*ImpersonationToken, error) {
	return verifyImpersonationToken(tokenStr)
}

// signImpersonationToken creates an HMAC-signed base64 token.
func signImpersonationToken(token *ImpersonationToken) (string, error) {
	payload, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("marshaling token: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(impersonationSecret))
	mac.Write(payload)
	sig := mac.Sum(nil)

	// Format: base64(payload).base64(signature)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payloadB64 + "." + sigB64, nil
}

// verifyImpersonationToken verifies the HMAC signature and expiry of a token.
func verifyImpersonationToken(tokenStr string) (*ImpersonationToken, error) {
	// Split into payload and signature.
	dotIdx := -1
	for i := len(tokenStr) - 1; i >= 0; i-- {
		if tokenStr[i] == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx < 0 {
		return nil, fmt.Errorf("malformed impersonation token")
	}

	payloadB64 := tokenStr[:dotIdx]
	sigB64 := tokenStr[dotIdx+1:]

	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decoding token payload: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("decoding token signature: %w", err)
	}

	// Verify HMAC.
	mac := hmac.New(sha256.New, []byte(impersonationSecret))
	mac.Write(payload)
	expectedSig := mac.Sum(nil)
	if !hmac.Equal(sig, expectedSig) {
		return nil, fmt.Errorf("invalid token signature")
	}

	var token ImpersonationToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	// Check expiry.
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("impersonation token has expired")
	}

	return &token, nil
}

// ImpersonateHandler handles POST /v1/admin/users/{user_id}/impersonate.
func (h *Handler) ImpersonateHandler(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "user_id")
	if targetUserID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	var body struct {
		Reason          string `json:"reason"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if body.Reason == "" {
		apierrors.ValidationError(w, "reason is required", nil)
		return
	}

	uc := auth.GetUserContext(r.Context())
	impersonatorID := ""
	if uc != nil {
		impersonatorID = uc.UserID
	}

	token, signed, err := h.service.ImpersonateUser(r.Context(), impersonatorID, targetUserID, body.Reason, body.DurationMinutes)
	if err != nil {
		if isImpersonationValidationErr(err) {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		apierrors.InternalError(w, "failed to create impersonation token")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "impersonate", "user", targetUserID, nil,
		map[string]string{
			"impersonator_id": impersonatorID,
			"target_user_id":  targetUserID,
			"reason":          body.Reason,
			"expires_at":      token.ExpiresAt.Format(time.RFC3339),
		})

	response.JSON(w, http.StatusOK, map[string]any{
		"token":      signed,
		"expires_at": token.ExpiresAt.Format(time.RFC3339),
		"target_id":  targetUserID,
	})
}

// isImpersonationValidationErr checks if an error is a validation error.
func isImpersonationValidationErr(err error) bool {
	msg := err.Error()
	return msg == "cannot impersonate a platform admin" ||
		msg == "cannot impersonate yourself" ||
		msg == "target user_id is required" ||
		msg == "impersonator_id is required"
}
