package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

const (
	apiKeyPrefix    = "deft_live_"
	apiKeyRandBytes = 32
)

// Common API key errors.
var (
	ErrAPIKeyInvalid = errors.New("invalid API key")
	ErrAPIKeyExpired = errors.New("API key has expired")
)

// APIKeyService handles API key CRUD and validation.
type APIKeyService struct {
	db *gorm.DB
}

// NewAPIKeyService creates a new API key service.
func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// CreateKeyResult contains the generated API key details.
type CreateKeyResult struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"` // Only returned on creation.
	KeyPrefix string     `json:"key_prefix"`
	OrgID     string     `json:"org_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateKey generates a new API key for an org.
func (s *APIKeyService) CreateKey(orgID, name string, expiresAt *time.Time) (*CreateKeyResult, error) {
	rawKey, err := generateRawKey()
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	hash := hashKey(rawKey)
	prefix := rawKey[:len(apiKeyPrefix)+8] // prefix + first 8 hex chars

	apiKey := &models.APIKey{
		OrgID:       orgID,
		Name:        name,
		KeyHash:     hash,
		KeyPrefix:   prefix,
		Permissions: "{}",
		ExpiresAt:   expiresAt,
	}
	if err := s.db.Create(apiKey).Error; err != nil {
		return nil, fmt.Errorf("storing API key: %w", err)
	}

	return &CreateKeyResult{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       rawKey,
		KeyPrefix: apiKey.KeyPrefix,
		OrgID:     apiKey.OrgID,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

// ListKeys returns all active API keys for an org (without hashes).
func (s *APIKeyService) ListKeys(orgID string) ([]models.APIKey, error) {
	var keys []models.APIKey
	if err := s.db.Where("org_id = ?", orgID).Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	return keys, nil
}

// RevokeKey soft-deletes an API key.
func (s *APIKeyService) RevokeKey(orgID, keyID string) error {
	result := s.db.Where("org_id = ? AND id = ?", orgID, keyID).Delete(&models.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("revoking API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ValidateKey checks an API key string and returns the associated key record.
func (s *APIKeyService) ValidateKey(rawKey string) (*models.APIKey, error) {
	if len(rawKey) < len(apiKeyPrefix) || rawKey[:len(apiKeyPrefix)] != apiKeyPrefix {
		return nil, ErrAPIKeyInvalid
	}

	hash := hashKey(rawKey)
	var key models.APIKey
	if err := s.db.Where("key_hash = ?", hash).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyInvalid
		}
		return nil, fmt.Errorf("looking up API key: %w", err)
	}

	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}

	// Update last used timestamp.
	now := time.Now()
	_ = s.db.Model(&key).Update("last_used_at", now).Error

	return &key, nil
}

// generateRawKey creates a new raw API key string.
func generateRawKey() (string, error) {
	b := make([]byte, apiKeyRandBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return apiKeyPrefix + hex.EncodeToString(b), nil
}

// hashKey computes the SHA-256 hash of an API key.
func hashKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

// --- HTTP Handlers ---

// APIKeyHandler provides HTTP handlers for API key management.
type APIKeyHandler struct {
	service *APIKeyService
}

// NewAPIKeyHandler creates a new API key handler.
func NewAPIKeyHandler(service *APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// createKeyRequest is the request body for creating an API key.
type createKeyRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Create handles POST /v1/orgs/{org}/api-keys.
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if orgID == "" {
		apierrors.BadRequest(w, "org ID is required")
		return
	}

	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" {
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "name", Message: "name is required"},
		})
		return
	}

	result, err := h.service.CreateKey(orgID, req.Name, req.ExpiresAt)
	if err != nil {
		apierrors.InternalError(w, "failed to create API key")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

// List handles GET /v1/orgs/{org}/api-keys.
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if orgID == "" {
		apierrors.BadRequest(w, "org ID is required")
		return
	}

	keys, err := h.service.ListKeys(orgID)
	if err != nil {
		apierrors.InternalError(w, "failed to list API keys")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": keys,
	})
}

// Revoke handles DELETE /v1/orgs/{org}/api-keys/{id}.
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	keyID := chi.URLParam(r, "id")
	if orgID == "" || keyID == "" {
		apierrors.BadRequest(w, "org ID and key ID are required")
		return
	}

	if err := h.service.RevokeKey(orgID, keyID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apierrors.NotFound(w, "API key not found")
			return
		}
		apierrors.InternalError(w, "failed to revoke API key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
