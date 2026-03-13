package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Platform-wide Audit Log ---

// AuditListParams extends pagination with platform-wide audit filters.
type AuditListParams struct {
	pagination.Params
	OrgID      string
	UserID     string
	Action     string
	EntityType string
	IPAddress  string
	After      *time.Time
	Before     *time.Time
}

// ListAuditLogs returns a platform-wide filtered, paginated audit log.
func (s *Service) ListAuditLogs(ctx context.Context, params AuditListParams) ([]models.AuditLog, *pagination.PageInfo, error) {
	var logs []models.AuditLog
	query := s.db.WithContext(ctx).Order("id DESC")

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}
	if params.EntityType != "" {
		query = query.Where("entity_type = ?", params.EntityType)
	}
	if params.IPAddress != "" {
		query = query.Where("ip_address = ?", params.IPAddress)
	}
	if params.After != nil {
		query = query.Where("created_at >= ?", *params.After)
	}
	if params.Before != nil {
		query = query.Where("created_at <= ?", *params.Before)
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
