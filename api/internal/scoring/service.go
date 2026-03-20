package scoring

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
)

// Service provides business logic for lead scoring.
type Service struct {
	db       *gorm.DB
	eventBus *event.Bus
}

// NewService creates a new scoring service.
func NewService(db *gorm.DB, eventBus *event.Bus) *Service {
	return &Service{db: db, eventBus: eventBus}
}

// ScoreThread evaluates all scoring rules against a thread and updates its lead_score.
func (s *Service) ScoreThread(ctx context.Context, threadID string) (*ScoreBreakdown, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}

	var thread models.Thread
	if err := s.db.WithContext(ctx).First(&thread, "id = ?", threadID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("thread not found")
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}

	// Resolve org for custom rules.
	rules := s.resolveRules(ctx, thread.BoardID)

	breakdown := Evaluate(rules, thread.Metadata)

	// Persist the score using a transaction so we always merge against the latest
	// metadata, preventing a read-modify-write race with concurrent stage transitions.
	scoreUpdate := map[string]any{"lead_score": breakdown.TotalScore}
	updateJSON, err := json.Marshal(scoreUpdate)
	if err != nil {
		return nil, fmt.Errorf("marshaling score update: %w", err)
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fresh models.Thread
		if err := tx.First(&fresh, "id = ?", threadID).Error; err != nil {
			return fmt.Errorf("re-reading thread: %w", err)
		}
		merged, err := metadata.DeepMerge(fresh.Metadata, string(updateJSON))
		if err != nil {
			return fmt.Errorf("merging metadata: %w", err)
		}
		return tx.Model(&fresh).Update("metadata", merged).Error
	}); err != nil {
		return nil, fmt.Errorf("saving thread: %w", err)
	}

	// Publish score updated event.
	if s.eventBus != nil {
		payload, _ := json.Marshal(breakdown)
		s.eventBus.Publish(event.Event{
			Type:       event.LeadScoreUpdated,
			EntityType: "thread",
			EntityID:   thread.ID,
			Payload:    string(payload),
		})
	}

	return breakdown, nil
}

// GetScore returns the current score breakdown for a thread without updating.
func (s *Service) GetScore(ctx context.Context, threadID string) (*ScoreBreakdown, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}

	var thread models.Thread
	if err := s.db.WithContext(ctx).First(&thread, "id = ?", threadID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("thread not found")
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}

	rules := s.resolveRules(ctx, thread.BoardID)
	return Evaluate(rules, thread.Metadata), nil
}

// HandleStageChanged is an event handler that recalculates score on pipeline stage changes.
func (s *Service) HandleStageChanged(evt event.Event) {
	if evt.EntityType != "thread" || evt.EntityID == "" {
		return
	}
	// Use background context since events are processed async.
	_, _ = s.ScoreThread(context.Background(), evt.EntityID)
}

// resolveRules loads scoring rules from org metadata or returns defaults.
func (s *Service) resolveRules(ctx context.Context, boardID string) []ScoringRule {
	orgID := s.resolveOrgID(ctx, boardID)
	if orgID == "" {
		return DefaultRules()
	}

	var org models.Org
	if err := s.db.WithContext(ctx).First(&org, "id = ?", orgID).Error; err != nil {
		return DefaultRules()
	}

	custom := ParseRulesFromMetadata(org.Metadata)
	if custom != nil {
		return custom
	}
	return DefaultRules()
}

// resolveOrgID resolves the org ID from a board ID by traversing the hierarchy.
func (s *Service) resolveOrgID(ctx context.Context, boardID string) string {
	var board models.Board
	if err := s.db.WithContext(ctx).First(&board, "id = ?", boardID).Error; err != nil {
		return ""
	}
	var space models.Space
	if err := s.db.WithContext(ctx).First(&space, "id = ?", board.SpaceID).Error; err != nil {
		return ""
	}
	return space.OrgID
}
