package admin

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// IntegrationStatus represents the health status of an external integration.
type IntegrationStatus struct {
	Clerk     string `json:"clerk"`
	Resend    string `json:"resend"`
	FlexPoint string `json:"flexpoint"`
}

// GetIntegrationStatus performs lightweight health checks on all integrations.
func (s *Service) GetIntegrationStatus(_ context.Context) *IntegrationStatus {
	status := &IntegrationStatus{
		Clerk:     "unconfigured",
		Resend:    "unconfigured",
		FlexPoint: "unconfigured",
	}

	// Clerk: check if API key is configured.
	if key := os.Getenv("CLERK_SECRET_KEY"); key != "" {
		status.Clerk = "ok"
	}

	// Resend: check if API key is configured.
	if key := os.Getenv("RESEND_API_KEY"); key != "" {
		status.Resend = "ok"
	}

	// FlexPoint: check if API key is configured.
	if key := os.Getenv("FLEXPOINT_API_KEY"); key != "" {
		status.FlexPoint = "ok"
	}

	return status
}

// GetIntegrationHealth handles GET /v1/admin/integrations/status.
func (h *Handler) GetIntegrationHealth(w http.ResponseWriter, r *http.Request) {
	status := h.service.GetIntegrationStatus(r.Context())
	response.JSON(w, http.StatusOK, status)
}

// ListAllWebhookDeliveries returns platform-wide webhook deliveries (no org scope filter).
func (s *Service) ListAllWebhookDeliveries(ctx context.Context, params pagination.Params) ([]models.WebhookDelivery, *pagination.PageInfo, error) {
	var deliveries []models.WebhookDelivery
	query := s.db.WithContext(ctx).Order("id DESC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&deliveries).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []models.WebhookDelivery{}, &pagination.PageInfo{}, nil
		}
		return nil, nil, fmt.Errorf("listing webhook deliveries: %w", err)
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

// ListWebhookDeliveries handles GET /v1/admin/webhooks/deliveries.
func (h *Handler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r)

	deliveries, pageInfo, err := h.service.ListAllWebhookDeliveries(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to list webhook deliveries")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     deliveries,
		PageInfo: pageInfo,
	})
}
