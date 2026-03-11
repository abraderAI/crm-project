package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for LLM enrichment operations.
type Handler struct {
	provider LLMProvider
	db       *gorm.DB
	eventBus *event.Bus
}

// NewHandler creates a new LLM enrichment handler.
func NewHandler(provider LLMProvider, db *gorm.DB, eventBus *event.Bus) *Handler {
	return &Handler{provider: provider, db: db, eventBus: eventBus}
}

// Enrich handles POST .../threads/{thread}/enrich.
func (h *Handler) Enrich(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")
	if threadID == "" {
		apierrors.BadRequest(w, "thread identifier is required")
		return
	}

	result, err := h.enrichThread(r.Context(), threadID)
	if err != nil {
		if err.Error() == "thread not found" {
			apierrors.NotFound(w, "thread not found")
			return
		}
		apierrors.InternalError(w, fmt.Sprintf("enrichment failed: %s", err.Error()))
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// enrichThread performs the enrichment and stores results in thread metadata.
func (h *Handler) enrichThread(ctx context.Context, threadID string) (*EnrichResult, error) {
	var thread models.Thread
	if err := h.db.WithContext(ctx).First(&thread, "id = ?", threadID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("thread not found")
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}

	// Extract current stage from metadata.
	stage := extractStageFromMeta(thread.Metadata)

	result := &EnrichResult{ThreadID: thread.ID}

	// Get summary.
	summary, err := h.provider.Summarize(ctx, SummarizeInput{
		ThreadID: thread.ID,
		Title:    thread.Title,
		Body:     thread.Body,
		Metadata: thread.Metadata,
	})
	if err == nil {
		result.Summary = summary
	}

	// Get next action suggestion.
	suggestion, err := h.provider.SuggestNextAction(ctx, SuggestInput{
		ThreadID: thread.ID,
		Title:    thread.Title,
		Body:     thread.Body,
		Stage:    stage,
		Metadata: thread.Metadata,
	})
	if err == nil {
		result.Suggestion = suggestion
	}

	// Store results in thread metadata.
	updates := make(map[string]any)
	if result.Summary != nil {
		updates["llm_summary"] = result.Summary.Text
	}
	if result.Suggestion != nil {
		updates["llm_next_action"] = result.Suggestion.Action
	}

	if len(updates) > 0 {
		updateJSON, err := json.Marshal(updates)
		if err != nil {
			return nil, fmt.Errorf("marshaling updates: %w", err)
		}
		merged, err := metadata.DeepMerge(thread.Metadata, string(updateJSON))
		if err != nil {
			return nil, fmt.Errorf("merging metadata: %w", err)
		}
		thread.Metadata = merged
		if err := h.db.WithContext(ctx).Save(&thread).Error; err != nil {
			return nil, fmt.Errorf("saving thread: %w", err)
		}
	}

	// Publish enrichment event.
	if h.eventBus != nil {
		payload, _ := json.Marshal(result)
		h.eventBus.Publish(event.Event{
			Type:       event.LeadEnriched,
			EntityType: "thread",
			EntityID:   thread.ID,
			Payload:    string(payload),
		})
	}

	return result, nil
}

// extractStageFromMeta extracts the stage from thread metadata JSON.
func extractStageFromMeta(metadataJSON string) string {
	if metadataJSON == "" || metadataJSON == "{}" {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return ""
	}
	if stage, ok := meta["stage"].(string); ok {
		return stage
	}
	return ""
}
