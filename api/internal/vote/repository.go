// Package vote provides the Vote domain (handler → service → repository).
package vote

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Repository handles database operations for Votes.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Vote repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// FindByUserAndThread returns a vote if the user has already voted on the thread.
func (r *Repository) FindByUserAndThread(ctx context.Context, userID, threadID string) (*models.Vote, error) {
	var vote models.Vote
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		First(&vote).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding vote: %w", err)
	}
	return &vote, nil
}

// Create inserts a new Vote.
func (r *Repository) Create(ctx context.Context, vote *models.Vote) error {
	if err := r.db.WithContext(ctx).Create(vote).Error; err != nil {
		return fmt.Errorf("creating vote: %w", err)
	}
	return nil
}

// Delete removes a Vote by ID (hard delete, not soft delete).
// Hard delete is required because the unique constraint on (thread_id, user_id)
// would be violated by soft-deleted records on re-vote.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&models.Vote{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting vote: %w", result.Error)
	}
	return nil
}

// RecalculateThreadScore sums all vote weights for a thread and updates Thread.VoteScore atomically.
func (r *Repository) RecalculateThreadScore(ctx context.Context, threadID string) (int, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&models.Vote{}).
		Where("thread_id = ?", threadID).
		Select("COALESCE(SUM(weight), 0)").
		Row().Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("calculating vote score: %w", err)
	}

	score := int(total)
	err = r.db.WithContext(ctx).
		Model(&models.Thread{}).
		Where("id = ?", threadID).
		Update("vote_score", score).Error
	if err != nil {
		return 0, fmt.Errorf("updating thread vote score: %w", err)
	}

	return score, nil
}

// FindThread retrieves a thread by ID.
func (r *Repository) FindThread(ctx context.Context, threadID string) (*models.Thread, error) {
	var thread models.Thread
	if err := r.db.WithContext(ctx).First(&thread, "id = ?", threadID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}
	return &thread, nil
}
