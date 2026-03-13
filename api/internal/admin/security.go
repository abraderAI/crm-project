package admin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// loginDebounceCache tracks the last login event per user to debounce writes.
var loginDebounceCache = struct {
	sync.Mutex
	seen map[string]time.Time
}{seen: make(map[string]time.Time)}

// LoginEventRecorder middleware records login events debounced to once per user per hour.
func LoginEventRecorder(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uc := auth.GetUserContext(r.Context())
			if uc != nil && uc.UserID != "" {
				userID := uc.UserID
				now := time.Now()
				hourKey := userID + ":" + now.Format("2006-01-02-15")

				loginDebounceCache.Lock()
				lastSeen, exists := loginDebounceCache.seen[hourKey]
				shouldRecord := !exists || now.Sub(lastSeen) > time.Hour
				if shouldRecord {
					loginDebounceCache.seen[hourKey] = now
				}
				loginDebounceCache.Unlock()

				if shouldRecord {
					ip := extractIP(r)
					ua := r.UserAgent()
					go func() {
						_ = db.Create(&models.LoginEvent{
							UserID:    userID,
							IPAddress: ip,
							UserAgent: ua,
						}).Error
					}()
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// FailedAuthRecorder middleware tracks 401 responses per IP/user.
func FailedAuthRecorder(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			if rw.statusCode == http.StatusUnauthorized {
				ip := extractIP(r)
				hour := time.Now().UTC().Format("2006-01-02-15")
				userID := ""
				// Try to extract user identity from the request.
				if token := r.Header.Get("Authorization"); token != "" {
					// Just use a hash of the token prefix for tracking.
					if len(token) > 20 {
						userID = "bearer:" + token[7:17]
					}
				}
				if key := r.Header.Get("X-API-Key"); key != "" {
					if len(key) > 10 {
						userID = "apikey:" + key[:10]
					}
				}

				go func() {
					_ = db.Clauses(clause.OnConflict{
						Columns:   []clause.Column{{Name: "ip_address"}, {Name: "user_id"}, {Name: "hour"}},
						DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("count + 1")}),
					}).Create(&models.FailedAuth{
						IPAddress: ip,
						UserID:    userID,
						Hour:      hour,
						Count:     1,
					}).Error
				}()
			}
		})
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// extractIP extracts the client IP from the request.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr.
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// RecentLoginEntry represents a login event for the API response.
type RecentLoginEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// GetRecentLogins returns recent login events.
func (s *Service) GetRecentLogins(ctx context.Context, params pagination.Params) ([]RecentLoginEntry, *pagination.PageInfo, error) {
	var events []models.LoginEvent
	query := s.db.WithContext(ctx).Order("created_at DESC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&events).Error; err != nil {
		return nil, nil, fmt.Errorf("querying login events: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(events) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(events[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		events = events[:params.Limit]
	}

	entries := make([]RecentLoginEntry, len(events))
	for i, e := range events {
		entries[i] = RecentLoginEntry{
			ID:        e.ID,
			UserID:    e.UserID,
			IPAddress: e.IPAddress,
			UserAgent: e.UserAgent,
			CreatedAt: e.CreatedAt,
		}
	}

	return entries, pageInfo, nil
}

// FailedAuthEntry represents a failed auth pattern for the API response.
type FailedAuthEntry struct {
	IPAddress string `json:"ip_address"`
	UserID    string `json:"user_id"`
	Hour      string `json:"hour"`
	Count     int64  `json:"count"`
}

// GetFailedAuths returns failed authentication attempts, surfacing potential brute-force patterns.
func (s *Service) GetFailedAuths(ctx context.Context, period string) ([]FailedAuthEntry, error) {
	cutoff := time.Now().UTC()
	switch period {
	case "24h":
		cutoff = cutoff.Add(-24 * time.Hour)
	case "7d":
		cutoff = cutoff.AddDate(0, 0, -7)
	default:
		cutoff = cutoff.Add(-24 * time.Hour)
	}
	cutoffHour := cutoff.Format("2006-01-02-15")

	var results []FailedAuthEntry
	err := s.db.WithContext(ctx).
		Model(&models.FailedAuth{}).
		Where("hour >= ?", cutoffHour).
		Order("count DESC").
		Limit(100).
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("querying failed auths: %w", err)
	}

	return results, nil
}

// GetRecentLoginsHandler handles GET /v1/admin/security/recent-logins.
func (h *Handler) GetRecentLoginsHandler(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r)
	entries, pageInfo, err := h.service.GetRecentLogins(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to get recent logins")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     entries,
		PageInfo: pageInfo,
	})
}

// GetFailedAuthsHandler handles GET /v1/admin/security/failed-auths.
func (h *Handler) GetFailedAuthsHandler(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	entries, err := h.service.GetFailedAuths(r.Context(), period)
	if err != nil {
		apierrors.InternalError(w, "failed to get failed auth data")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"period": period,
		"data":   entries,
	})
}

// ResetLoginDebounceCache clears the debounce cache (for testing).
func ResetLoginDebounceCache() {
	loginDebounceCache.Lock()
	loginDebounceCache.seen = make(map[string]time.Time)
	loginDebounceCache.Unlock()
}
