package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
)

// Service provides business logic for CRM pipeline operations.
type Service struct {
	db       *gorm.DB
	eventBus *event.Bus
}

// NewService creates a new pipeline service.
func NewService(db *gorm.DB, eventBus *event.Bus) *Service {
	return &Service{db: db, eventBus: eventBus}
}

// TransitionInput holds data for a stage transition request.
type TransitionInput struct {
	Stage Stage `json:"stage"`
}

// TransitionResult holds the outcome of a stage transition.
type TransitionResult struct {
	ThreadID      string `json:"thread_id"`
	PreviousStage string `json:"previous_stage"`
	NewStage      string `json:"new_stage"`
	OrgID         string `json:"org_id"`
}

// TransitionStage validates and applies a pipeline stage transition to a thread.
func (s *Service) TransitionStage(ctx context.Context, threadID string, newStage Stage, userID string) (*TransitionResult, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}
	if newStage == "" {
		return nil, fmt.Errorf("stage is required")
	}

	// Look up the thread.
	var thread models.Thread
	if err := s.db.WithContext(ctx).First(&thread, "id = ?", threadID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("thread not found")
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}

	// Resolve pipeline config from the org.
	orgID, err := s.resolveOrgID(ctx, thread.BoardID)
	if err != nil {
		return nil, fmt.Errorf("resolving org: %w", err)
	}

	stages := s.resolveStages(ctx, orgID)

	if !IsValidStage(stages, newStage) {
		return nil, fmt.Errorf("invalid stage: %s", newStage)
	}

	// Get current stage from thread metadata.
	currentStage := extractStage(thread.Metadata)

	// Validate transition.
	if err := ValidateTransition(stages, Stage(currentStage), newStage); err != nil {
		return nil, err
	}

	// Update thread metadata with new stage.
	stageUpdate := map[string]any{"stage": string(newStage)}
	updateJSON, err := json.Marshal(stageUpdate)
	if err != nil {
		return nil, fmt.Errorf("marshaling stage update: %w", err)
	}

	merged, err := metadata.DeepMerge(thread.Metadata, string(updateJSON))
	if err != nil {
		return nil, fmt.Errorf("merging metadata: %w", err)
	}
	thread.Metadata = merged

	if err := s.db.WithContext(ctx).Save(&thread).Error; err != nil {
		return nil, fmt.Errorf("saving thread: %w", err)
	}

	result := &TransitionResult{
		ThreadID:      thread.ID,
		PreviousStage: currentStage,
		NewStage:      string(newStage),
		OrgID:         orgID,
	}

	// Publish event.
	if s.eventBus != nil {
		payload, _ := json.Marshal(result)
		s.eventBus.Publish(event.Event{
			Type:       event.PipelineStageChanged,
			EntityType: "thread",
			EntityID:   thread.ID,
			OrgID:      orgID,
			UserID:     userID,
			Payload:    string(payload),
		})
	}

	return result, nil
}

// GetStages returns the pipeline stages configured for an org.
func (s *Service) GetStages(ctx context.Context, orgID string) []StageInfo {
	return s.resolveStages(ctx, orgID)
}

// resolveStages loads pipeline config from org metadata or returns defaults.
func (s *Service) resolveStages(ctx context.Context, orgID string) []StageInfo {
	if orgID == "" {
		return DefaultStages()
	}

	var org models.Org
	if err := s.db.WithContext(ctx).First(&org, "id = ?", orgID).Error; err != nil {
		return DefaultStages()
	}

	cfg := ParseConfigFromMetadata(org.Metadata)
	if cfg != nil {
		return cfg.Stages
	}
	return DefaultStages()
}

// resolveOrgID resolves the org ID from a board ID by traversing the hierarchy.
func (s *Service) resolveOrgID(ctx context.Context, boardID string) (string, error) {
	var board models.Board
	if err := s.db.WithContext(ctx).First(&board, "id = ?", boardID).Error; err != nil {
		return "", fmt.Errorf("finding board: %w", err)
	}
	var space models.Space
	if err := s.db.WithContext(ctx).First(&space, "id = ?", board.SpaceID).Error; err != nil {
		return "", fmt.Errorf("finding space: %w", err)
	}
	return space.OrgID, nil
}

// extractStage extracts the stage value from thread metadata JSON.
func extractStage(metadataJSON string) string {
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
