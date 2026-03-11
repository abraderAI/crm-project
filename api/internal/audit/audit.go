// Package audit provides audit logging for every mutation and a query API.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/middleware"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Service provides audit log read/write operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new audit service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Log writes an audit log entry asynchronously.
func (s *Service) Log(ctx context.Context, entry models.AuditLog) {
	// Set request ID and IP from context if available.
	if entry.RequestID == "" {
		entry.RequestID = middleware.GetRequestID(ctx)
	}
	// Async write to avoid blocking the request path.
	go func() {
		_ = s.db.Create(&entry).Error
	}()
}

// LogSync writes an audit log entry synchronously (for tests).
func (s *Service) LogSync(ctx context.Context, entry models.AuditLog) error {
	if entry.RequestID == "" {
		entry.RequestID = middleware.GetRequestID(ctx)
	}
	return s.db.WithContext(ctx).Create(&entry).Error
}

// ListParams extends pagination with audit-specific filters.
type ListParams struct {
	pagination.Params
	EntityType string
	EntityID   string
	Action     string
	UserID     string
}

// List returns a paginated, filtered list of audit log entries for an org.
func (s *Service) List(ctx context.Context, params ListParams) ([]models.AuditLog, *pagination.PageInfo, error) {
	var logs []models.AuditLog
	query := s.db.WithContext(ctx).Order("id DESC")

	if params.EntityType != "" {
		query = query.Where("entity_type = ?", params.EntityType)
	}
	if params.EntityID != "" {
		query = query.Where("entity_id = ?", params.EntityID)
	}
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}
	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&logs).Error; err != nil {
		return nil, nil, fmt.Errorf("listing audit logs: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(logs) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(logs[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		logs = logs[:params.Limit]
	}

	return logs, pageInfo, nil
}

// Handler provides HTTP handlers for the audit log API.
type Handler struct {
	service *Service
}

// NewHandler creates a new audit handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /v1/orgs/{org}/audit-log.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "org") // available for future RBAC scoping

	params := ListParams{
		Params:     pagination.Parse(r),
		EntityType: r.URL.Query().Get("entity_type"),
		EntityID:   r.URL.Query().Get("entity_id"),
		Action:     r.URL.Query().Get("action"),
		UserID:     r.URL.Query().Get("user_id"),
	}

	logs, pageInfo, err := h.service.List(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to list audit logs")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     logs,
		PageInfo: pageInfo,
	})
}

// CreateAuditEntry is a helper to build and log an audit entry from a handler context.
func CreateAuditEntry(ctx context.Context, svc *Service, action models.AuditAction, entityType, entityID string, before, after any) {
	uc := auth.GetUserContext(ctx)
	userID := ""
	if uc != nil {
		userID = uc.UserID
	}

	var beforeJSON, afterJSON string
	if before != nil {
		b, _ := json.Marshal(before)
		beforeJSON = string(b)
	}
	if after != nil {
		b, _ := json.Marshal(after)
		afterJSON = string(b)
	}

	svc.Log(ctx, models.AuditLog{
		UserID:      userID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		BeforeState: beforeJSON,
		AfterState:  afterJSON,
		RequestID:   middleware.GetRequestID(ctx),
	})
}
