package moderation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Service provides business logic for moderation operations.
type Service struct {
	repo *Repository
}

// NewService creates a new Moderation service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// FlagInput holds the data needed to create a flag.
type FlagInput struct {
	ThreadID string `json:"thread_id"`
	Reason   string `json:"reason"`
}

// CreateFlag creates a new moderation flag on a thread.
func (s *Service) CreateFlag(ctx context.Context, userID string, input FlagInput) (*models.Flag, error) {
	if input.ThreadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}
	if input.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	// Verify thread exists.
	thread, err := s.repo.FindThreadByID(ctx, input.ThreadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, fmt.Errorf("thread not found")
	}

	flag := &models.Flag{
		ThreadID: input.ThreadID,
		UserID:   userID,
		Reason:   input.Reason,
		Status:   models.FlagStatusOpen,
	}
	if err := s.repo.CreateFlag(ctx, flag); err != nil {
		return nil, err
	}

	// Audit log.
	s.audit(ctx, userID, models.AuditActionCreate, "flag", flag.ID, "", mustJSON(flag))

	return flag, nil
}

// ListOrgFlags returns the moderation flag queue for an org.
func (s *Service) ListOrgFlags(ctx context.Context, orgID string, params pagination.Params) ([]models.Flag, *pagination.PageInfo, error) {
	return s.repo.ListOrgFlags(ctx, orgID, models.FlagStatusOpen, params)
}

// ResolveFlag marks a flag as resolved.
func (s *Service) ResolveFlag(ctx context.Context, flagID, moderatorID string) (*models.Flag, error) {
	return s.updateFlagStatus(ctx, flagID, moderatorID, models.FlagStatusResolved)
}

// DismissFlag marks a flag as dismissed.
func (s *Service) DismissFlag(ctx context.Context, flagID, moderatorID string) (*models.Flag, error) {
	return s.updateFlagStatus(ctx, flagID, moderatorID, models.FlagStatusDismissed)
}

func (s *Service) updateFlagStatus(ctx context.Context, flagID, moderatorID string, status models.FlagStatus) (*models.Flag, error) {
	flag, err := s.repo.FindFlagByID(ctx, flagID)
	if err != nil {
		return nil, err
	}
	if flag == nil {
		return nil, fmt.Errorf("flag not found")
	}
	if flag.Status != models.FlagStatusOpen {
		return nil, fmt.Errorf("flag is already %s", flag.Status)
	}

	before := mustJSON(flag)
	flag.Status = status
	flag.ResolvedBy = moderatorID
	if err := s.repo.UpdateFlag(ctx, flag); err != nil {
		return nil, err
	}

	s.audit(ctx, moderatorID, models.AuditActionUpdate, "flag", flag.ID, before, mustJSON(flag))

	return flag, nil
}

// MoveInput holds the data for a thread move operation.
type MoveInput struct {
	TargetBoardID string `json:"target_board_id"`
}

// MoveThread moves a thread from its current board to a different board.
func (s *Service) MoveThread(ctx context.Context, threadID, moderatorID string, input MoveInput) (*models.Thread, error) {
	if input.TargetBoardID == "" {
		return nil, fmt.Errorf("target_board_id is required")
	}

	thread, err := s.repo.FindThreadByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, fmt.Errorf("thread not found")
	}

	if thread.BoardID == input.TargetBoardID {
		return nil, fmt.Errorf("thread is already in the target board")
	}

	// Verify target board exists.
	targetBoard, err := s.repo.FindBoardByID(ctx, input.TargetBoardID)
	if err != nil {
		return nil, err
	}
	if targetBoard == nil {
		return nil, fmt.Errorf("target board not found")
	}

	before := mustJSON(map[string]string{"board_id": thread.BoardID})
	thread.BoardID = input.TargetBoardID
	if err := s.repo.UpdateThread(ctx, thread); err != nil {
		return nil, err
	}

	after := mustJSON(map[string]string{"board_id": thread.BoardID})
	s.audit(ctx, moderatorID, models.AuditActionUpdate, "thread", thread.ID, before, after)

	return thread, nil
}

// MergeInput holds the data for a thread merge operation.
type MergeInput struct {
	TargetThreadID string `json:"target_thread_id"`
}

// MergeThread merges a source thread into a target thread.
// All messages from the source are moved to the target, then the source is soft-deleted.
func (s *Service) MergeThread(ctx context.Context, sourceThreadID, moderatorID string, input MergeInput) (*models.Thread, error) {
	if input.TargetThreadID == "" {
		return nil, fmt.Errorf("target_thread_id is required")
	}
	if sourceThreadID == input.TargetThreadID {
		return nil, fmt.Errorf("cannot merge a thread into itself")
	}

	source, err := s.repo.FindThreadByID(ctx, sourceThreadID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("source thread not found")
	}

	target, err := s.repo.FindThreadByID(ctx, input.TargetThreadID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("target thread not found")
	}

	// Move all messages from source to target.
	if err := s.repo.MoveMessages(ctx, sourceThreadID, input.TargetThreadID); err != nil {
		return nil, err
	}

	// Soft-delete the source thread.
	if err := s.repo.SoftDeleteThread(ctx, sourceThreadID); err != nil {
		return nil, err
	}

	s.audit(ctx, moderatorID, models.AuditActionUpdate, "thread", input.TargetThreadID,
		mustJSON(map[string]string{"merged_from": sourceThreadID}),
		mustJSON(map[string]string{"merged_from": sourceThreadID, "status": "merged"}))

	return target, nil
}

// HideThread sets the IsHidden flag on a thread.
func (s *Service) HideThread(ctx context.Context, threadID, moderatorID string) (*models.Thread, error) {
	return s.setHidden(ctx, threadID, moderatorID, true)
}

// UnhideThread clears the IsHidden flag on a thread.
func (s *Service) UnhideThread(ctx context.Context, threadID, moderatorID string) (*models.Thread, error) {
	return s.setHidden(ctx, threadID, moderatorID, false)
}

func (s *Service) setHidden(ctx context.Context, threadID, moderatorID string, hidden bool) (*models.Thread, error) {
	thread, err := s.repo.FindThreadByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, fmt.Errorf("thread not found")
	}

	before := mustJSON(map[string]bool{"is_hidden": thread.IsHidden})
	thread.IsHidden = hidden
	if err := s.repo.UpdateThread(ctx, thread); err != nil {
		return nil, err
	}

	action := "hide"
	if !hidden {
		action = "unhide"
	}
	after := mustJSON(map[string]any{"is_hidden": hidden, "action": action})
	s.audit(ctx, moderatorID, models.AuditActionUpdate, "thread", thread.ID, before, after)

	return thread, nil
}

// audit writes an audit log entry, silently ignoring errors.
func (s *Service) audit(ctx context.Context, userID string, action models.AuditAction, entityType, entityID, before, after string) {
	entry := &models.AuditLog{
		UserID:      userID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		BeforeState: before,
		AfterState:  after,
	}
	_ = s.repo.CreateAuditLog(ctx, entry)
}

// mustJSON marshals v to JSON string, returning "{}" on error.
func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
