package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// PlatformStats holds DB-derived platform metrics.
type PlatformStats struct {
	Orgs                 CountStats `json:"orgs"`
	Users                CountStats `json:"users"`
	Threads              CountStats `json:"threads"`
	Messages             CountStats `json:"messages"`
	DBSizeBytes          int64      `json:"db_size_bytes"`
	ApiUptimePct         float64    `json:"api_uptime_pct"`
	FailedWebhooks24h    int64      `json:"failed_webhooks_24h"`
	PendingNotifications int64      `json:"pending_notifications"`
}

// CountStats holds total and recent counts.
type CountStats struct {
	Total   int64 `json:"total"`
	Last7d  int64 `json:"last_7d"`
	Last30d int64 `json:"last_30d"`
}

// GetPlatformStats computes platform-wide statistics from the database.
func (s *Service) GetPlatformStats(ctx context.Context) (*PlatformStats, error) {
	stats := &PlatformStats{}
	now := time.Now()
	d7 := now.AddDate(0, 0, -7)
	d30 := now.AddDate(0, 0, -30)

	// Org counts.
	if err := s.db.WithContext(ctx).Model(&models.Org{}).Count(&stats.Orgs.Total).Error; err != nil {
		return nil, fmt.Errorf("counting orgs: %w", err)
	}
	s.db.WithContext(ctx).Model(&models.Org{}).Where("created_at >= ?", d7).Count(&stats.Orgs.Last7d)
	s.db.WithContext(ctx).Model(&models.Org{}).Where("created_at >= ?", d30).Count(&stats.Orgs.Last30d)

	// User counts (from user_shadows).
	if err := s.db.WithContext(ctx).Model(&models.UserShadow{}).Count(&stats.Users.Total).Error; err != nil {
		return nil, fmt.Errorf("counting users: %w", err)
	}
	s.db.WithContext(ctx).Model(&models.UserShadow{}).Where("last_seen_at >= ?", d7).Count(&stats.Users.Last7d)
	s.db.WithContext(ctx).Model(&models.UserShadow{}).Where("last_seen_at >= ?", d30).Count(&stats.Users.Last30d)

	// Thread counts.
	if err := s.db.WithContext(ctx).Model(&models.Thread{}).Count(&stats.Threads.Total).Error; err != nil {
		return nil, fmt.Errorf("counting threads: %w", err)
	}
	s.db.WithContext(ctx).Model(&models.Thread{}).Where("created_at >= ?", d7).Count(&stats.Threads.Last7d)
	s.db.WithContext(ctx).Model(&models.Thread{}).Where("created_at >= ?", d30).Count(&stats.Threads.Last30d)

	// Message counts.
	if err := s.db.WithContext(ctx).Model(&models.Message{}).Count(&stats.Messages.Total).Error; err != nil {
		return nil, fmt.Errorf("counting messages: %w", err)
	}
	s.db.WithContext(ctx).Model(&models.Message{}).Where("created_at >= ?", d7).Count(&stats.Messages.Last7d)
	s.db.WithContext(ctx).Model(&models.Message{}).Where("created_at >= ?", d30).Count(&stats.Messages.Last30d)

	// DB file size (SQLite PRAGMA).
	var pageCount, pageSize int64
	s.db.WithContext(ctx).Raw("PRAGMA page_count").Scan(&pageCount)
	s.db.WithContext(ctx).Raw("PRAGMA page_size").Scan(&pageSize)
	stats.DBSizeBytes = pageCount * pageSize

	// Failed webhook deliveries (last 24h).
	d24h := now.Add(-24 * time.Hour)
	s.db.WithContext(ctx).Model(&models.WebhookDelivery{}).
		Where("(status_code < 200 OR status_code >= 300) AND created_at >= ?", d24h).
		Count(&stats.FailedWebhooks24h)

	// Pending (unread) notifications.
	s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("is_read = ?", false).
		Count(&stats.PendingNotifications)

	// API uptime: derive from webhook delivery success rate (24h), fallback 100.
	var totalDeliveries, failedDeliveries int64
	s.db.WithContext(ctx).Model(&models.WebhookDelivery{}).
		Where("created_at >= ?", d24h).
		Count(&totalDeliveries)
	failedDeliveries = stats.FailedWebhooks24h
	if totalDeliveries > 0 {
		stats.ApiUptimePct = 100.0 * float64(totalDeliveries-failedDeliveries) / float64(totalDeliveries)
	} else {
		stats.ApiUptimePct = 100.0
	}

	return stats, nil
}

// GetStats handles GET /v1/admin/stats.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetPlatformStats(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to compute platform stats")
		return
	}
	response.JSON(w, http.StatusOK, stats)
}
