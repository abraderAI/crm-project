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

// LLMUsageEntry represents a single LLM usage log entry for the API response.
type LLMUsageEntry struct {
	ID           string    `json:"id"`
	Endpoint     string    `json:"endpoint"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	DurationMs   int64     `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetLLMUsage returns recent LLM enrichment call logs.
func (s *Service) GetLLMUsage(ctx context.Context, limit int) ([]LLMUsageEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var logs []models.LLMUsageLog
	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("querying LLM usage: %w", err)
	}

	entries := make([]LLMUsageEntry, len(logs))
	for i, l := range logs {
		entries[i] = LLMUsageEntry{
			ID:           l.ID,
			Endpoint:     l.Endpoint,
			Model:        l.Model,
			InputTokens:  l.InputTokens,
			OutputTokens: l.OutputTokens,
			DurationMs:   l.DurationMs,
			CreatedAt:    l.CreatedAt,
		}
	}

	return entries, nil
}

// GetLLMUsageHandler handles GET /v1/admin/llm-usage.
func (h *Handler) GetLLMUsageHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.GetLLMUsage(r.Context(), 50)
	if err != nil {
		apierrors.InternalError(w, "failed to get LLM usage")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":    entries,
		"message": "LLM usage tracking is available; token counts require LLMProvider support",
	})
}
