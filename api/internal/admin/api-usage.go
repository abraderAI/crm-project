package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// APIUsageCounter middleware records per-endpoint request counts asynchronously.
func APIUsageCounter(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Record the request asynchronously.
			endpoint := r.URL.Path
			method := r.Method
			hour := time.Now().UTC().Format("2006-01-02-15")

			// Synchronous write avoids SQLite contention from concurrent goroutines.
			_ = db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "endpoint"}, {Name: "method"}, {Name: "hour"}},
				DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("count + 1")}),
			}).Create(&models.APIUsageStat{
				Endpoint: endpoint,
				Method:   method,
				Hour:     hour,
				Count:    1,
			}).Error

			next.ServeHTTP(w, r)
		})
	}
}

// APIUsageResponse represents a single endpoint's usage stats.
type APIUsageResponse struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Count    int64  `json:"count"`
}

// GetAPIUsage returns per-endpoint request counts for the given period.
func (s *Service) GetAPIUsage(ctx context.Context, period string) ([]APIUsageResponse, error) {
	cutoff := time.Now().UTC()
	switch period {
	case "24h":
		cutoff = cutoff.Add(-24 * time.Hour)
	case "7d":
		cutoff = cutoff.AddDate(0, 0, -7)
	case "30d":
		cutoff = cutoff.AddDate(0, 0, -30)
	default:
		cutoff = cutoff.Add(-24 * time.Hour)
	}

	cutoffHour := cutoff.Format("2006-01-02-15")

	var results []APIUsageResponse
	err := s.db.WithContext(ctx).
		Model(&models.APIUsageStat{}).
		Select("endpoint, method, SUM(count) as count").
		Where("hour >= ?", cutoffHour).
		Group("endpoint, method").
		Order("count DESC").
		Limit(100).
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("querying API usage: %w", err)
	}

	return results, nil
}

// GetAPIUsageHandler handles GET /v1/admin/api-usage.
func (h *Handler) GetAPIUsageHandler(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	results, err := h.service.GetAPIUsage(r.Context(), period)
	if err != nil {
		apierrors.InternalError(w, "failed to get API usage stats")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"period": period,
		"data":   results,
	})
}
