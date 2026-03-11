// Package webhook provides webhook subscription management and delivery.
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// MaxRetries is the maximum number of delivery retry attempts.
const MaxRetries = 3

// Service manages webhook subscriptions and deliveries.
type Service struct {
	db     *gorm.DB
	client *http.Client
}

// NewService creates a new webhook service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateInput holds the data needed to create a webhook subscription.
type CreateInput struct {
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`
	EventFilter []string `json:"event_filter"`
	ScopeType   string   `json:"scope_type"`
	ScopeID     string   `json:"scope_id"`
}

// Create creates a new webhook subscription.
func (s *Service) Create(ctx context.Context, orgID string, input CreateInput) (*models.WebhookSubscription, error) {
	if input.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if !isValidWebhookURL(input.URL) {
		return nil, fmt.Errorf("invalid webhook URL")
	}
	if input.Secret == "" {
		return nil, fmt.Errorf("secret is required")
	}

	filterJSON := "[]"
	if len(input.EventFilter) > 0 {
		b, err := json.Marshal(input.EventFilter)
		if err != nil {
			return nil, fmt.Errorf("encoding event filter: %w", err)
		}
		filterJSON = string(b)
	}

	sub := &models.WebhookSubscription{
		OrgID:       orgID,
		ScopeType:   input.ScopeType,
		ScopeID:     input.ScopeID,
		URL:         input.URL,
		Secret:      input.Secret,
		EventFilter: filterJSON,
		IsActive:    true,
	}
	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, fmt.Errorf("creating subscription: %w", err)
	}
	return sub, nil
}

// List returns webhook subscriptions for an org.
func (s *Service) List(ctx context.Context, orgID string, params pagination.Params) ([]models.WebhookSubscription, *pagination.PageInfo, error) {
	var subs []models.WebhookSubscription
	query := s.db.WithContext(ctx).Where("org_id = ?", orgID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&subs).Error; err != nil {
		return nil, nil, fmt.Errorf("listing subscriptions: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(subs) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(subs[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		subs = subs[:params.Limit]
	}

	return subs, pageInfo, nil
}

// Get retrieves a single subscription.
func (s *Service) Get(ctx context.Context, id string) (*models.WebhookSubscription, error) {
	var sub models.WebhookSubscription
	if err := s.db.WithContext(ctx).First(&sub, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("finding subscription: %w", err)
	}
	return &sub, nil
}

// Delete soft-deletes a webhook subscription.
func (s *Service) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.WebhookSubscription{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting subscription: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

// ListDeliveries returns deliveries for a subscription.
func (s *Service) ListDeliveries(ctx context.Context, subID string, params pagination.Params) ([]models.WebhookDelivery, *pagination.PageInfo, error) {
	var deliveries []models.WebhookDelivery
	query := s.db.WithContext(ctx).Where("subscription_id = ?", subID).Order("id DESC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&deliveries).Error; err != nil {
		return nil, nil, fmt.Errorf("listing deliveries: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(deliveries) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(deliveries[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		deliveries = deliveries[:params.Limit]
	}

	return deliveries, pageInfo, nil
}

// Replay re-delivers a webhook delivery.
func (s *Service) Replay(ctx context.Context, deliveryID string) (*models.WebhookDelivery, error) {
	var delivery models.WebhookDelivery
	if err := s.db.WithContext(ctx).First(&delivery, "id = ?", deliveryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("finding delivery: %w", err)
	}

	var sub models.WebhookSubscription
	if err := s.db.WithContext(ctx).First(&sub, "id = ?", delivery.SubscriptionID).Error; err != nil {
		return nil, fmt.Errorf("finding subscription: %w", err)
	}

	// Re-deliver.
	statusCode, respBody := s.deliver(sub.URL, sub.Secret, delivery.EventType, delivery.Payload)
	delivery.StatusCode = statusCode
	delivery.ResponseBody = respBody
	delivery.Attempts++
	if err := s.db.WithContext(ctx).Save(&delivery).Error; err != nil {
		return nil, fmt.Errorf("updating delivery: %w", err)
	}

	return &delivery, nil
}

// HandleEvent processes an event from the event bus and delivers to matching subscriptions.
func (s *Service) HandleEvent(evt event.Event) {
	ctx := context.Background()
	var subs []models.WebhookSubscription
	if err := s.db.WithContext(ctx).Where("org_id = ? AND is_active = ?", evt.OrgID, true).Find(&subs).Error; err != nil {
		return
	}

	for _, sub := range subs {
		if !matchesFilter(sub.EventFilter, string(evt.Type)) {
			continue
		}
		go s.deliverWithRetry(ctx, sub, evt)
	}
}

// deliverWithRetry attempts delivery with exponential backoff.
func (s *Service) deliverWithRetry(ctx context.Context, sub models.WebhookSubscription, evt event.Event) {
	payload, _ := json.Marshal(evt)
	payloadStr := string(payload)

	delivery := models.WebhookDelivery{
		SubscriptionID: sub.ID,
		EventType:      string(evt.Type),
		Payload:        payloadStr,
	}

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		statusCode, respBody := s.deliver(sub.URL, sub.Secret, string(evt.Type), payloadStr)
		delivery.StatusCode = statusCode
		delivery.ResponseBody = respBody
		delivery.Attempts = attempt

		if statusCode >= 200 && statusCode < 300 {
			break
		}

		if attempt < MaxRetries {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			nextRetry := time.Now().Add(backoff)
			delivery.NextRetryAt = &nextRetry
			time.Sleep(backoff)
		}
	}

	_ = s.db.WithContext(ctx).Create(&delivery).Error
}

// deliver makes the HTTP POST to the webhook URL with HMAC-SHA256 signature.
func (s *Service) deliver(url, secret, eventType, payload string) (int, string) {
	sig := SignPayload(secret, payload)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return 0, fmt.Sprintf("request error: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", sig)
	req.Header.Set("X-Webhook-Event", eventType)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Sprintf("delivery error: %s", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(body)
}

// SignPayload computes HMAC-SHA256 signature of the payload using the secret.
func SignPayload(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks that the provided signature matches the expected HMAC-SHA256.
func VerifySignature(secret, payload, signature string) bool {
	expected := SignPayload(secret, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// matchesFilter checks if an event type matches the subscription's event filter.
func matchesFilter(filterJSON, eventType string) bool {
	if filterJSON == "" || filterJSON == "[]" {
		return true // Empty filter matches all events.
	}
	var filters []string
	if err := json.Unmarshal([]byte(filterJSON), &filters); err != nil {
		return true // On parse error, allow delivery.
	}
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f == eventType {
			return true
		}
	}
	return false
}

// isValidWebhookURL performs basic webhook URL validation.
func isValidWebhookURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
