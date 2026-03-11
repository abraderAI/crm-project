package vote

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// WeightConfig maps role and billing tier combinations to vote weights.
type WeightConfig struct {
	// RoleWeights maps role names to their base weight.
	RoleWeights map[models.Role]int
	// TierBonuses maps billing tier names to bonus weight added on top of role weight.
	TierBonuses map[string]int
	// DefaultWeight is used when no specific mapping is found.
	DefaultWeight int
}

// DefaultWeightConfig returns the default vote weight configuration.
func DefaultWeightConfig() *WeightConfig {
	return &WeightConfig{
		RoleWeights: map[models.Role]int{
			models.RoleViewer:      1,
			models.RoleCommenter:   1,
			models.RoleContributor: 2,
			models.RoleModerator:   3,
			models.RoleAdmin:       4,
			models.RoleOwner:       5,
		},
		TierBonuses: map[string]int{
			"free":       0,
			"pro":        1,
			"enterprise": 2,
		},
		DefaultWeight: 1,
	}
}

// CalculateWeight determines the vote weight for a given role and billing tier.
func (wc *WeightConfig) CalculateWeight(role models.Role, billingTier string) int {
	weight := wc.DefaultWeight
	if w, ok := wc.RoleWeights[role]; ok {
		weight = w
	}
	if bonus, ok := wc.TierBonuses[billingTier]; ok {
		weight += bonus
	}
	return weight
}

// VoteResult holds the outcome of a toggle vote operation.
type VoteResult struct {
	Voted     bool `json:"voted"`
	VoteScore int  `json:"vote_score"`
	Weight    int  `json:"weight"`
}

// Service provides business logic for Vote operations.
type Service struct {
	repo         *Repository
	weightConfig *WeightConfig
}

// NewService creates a new Vote service with the given weight configuration.
// If weightConfig is nil, DefaultWeightConfig is used.
func NewService(repo *Repository, weightConfig *WeightConfig) *Service {
	if weightConfig == nil {
		weightConfig = DefaultWeightConfig()
	}
	return &Service{repo: repo, weightConfig: weightConfig}
}

// Toggle creates or removes a vote for the user on the thread.
// If the user has already voted, the vote is removed (toggle off).
// If the user has not voted, a new vote is created (toggle on).
// Returns the result including updated VoteScore.
func (s *Service) Toggle(ctx context.Context, threadID, userID string, role models.Role, billingTier string) (*VoteResult, error) {
	// Verify thread exists.
	thread, err := s.repo.FindThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, fmt.Errorf("thread not found")
	}

	existing, err := s.repo.FindByUserAndThread(ctx, userID, threadID)
	if err != nil {
		return nil, err
	}

	weight := s.weightConfig.CalculateWeight(role, billingTier)
	result := &VoteResult{Weight: weight}

	if existing != nil {
		// Remove existing vote (toggle off).
		if err := s.repo.Delete(ctx, existing.ID); err != nil {
			return nil, err
		}
		result.Voted = false
	} else {
		// Create new vote (toggle on).
		v := &models.Vote{
			ThreadID: threadID,
			UserID:   userID,
			Weight:   weight,
		}
		if err := s.repo.Create(ctx, v); err != nil {
			return nil, err
		}
		result.Voted = true
	}

	// Recalculate and update the thread's VoteScore.
	score, err := s.repo.RecalculateThreadScore(ctx, threadID)
	if err != nil {
		return nil, err
	}
	result.VoteScore = score

	return result, nil
}

// GetWeightConfig returns the current weight configuration.
func (s *Service) GetWeightConfig() *WeightConfig {
	return s.weightConfig
}
