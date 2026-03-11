package auth

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// RBACEngine resolves effective roles and checks permissions using the RBAC policy.
type RBACEngine struct {
	policy *config.RBACPolicy
	db     *gorm.DB
}

// NewRBACEngine creates a new RBAC engine.
func NewRBACEngine(policy *config.RBACPolicy, db *gorm.DB) *RBACEngine {
	return &RBACEngine{
		policy: policy,
		db:     db,
	}
}

// ResolveRole returns the effective role for a user on a given entity.
// It follows the configured resolution order: board → space → org.
// Returns empty string if the user has no membership.
func (e *RBACEngine) ResolveRole(ctx context.Context, userID, entityType, entityID string) (models.Role, error) {
	switch entityType {
	case "board":
		return e.resolveForBoard(ctx, userID, entityID)
	case "space":
		return e.resolveForSpace(ctx, userID, entityID)
	case "org":
		return e.resolveForOrg(ctx, userID, entityID)
	default:
		return "", fmt.Errorf("unknown entity type: %s", entityType)
	}
}

// resolveForBoard checks board → space → org membership.
func (e *RBACEngine) resolveForBoard(ctx context.Context, userID, boardID string) (models.Role, error) {
	// Check board membership.
	var boardMembership models.BoardMembership
	err := e.db.WithContext(ctx).
		Where("board_id = ? AND user_id = ?", boardID, userID).
		First(&boardMembership).Error
	if err == nil {
		return boardMembership.Role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("querying board membership: %w", err)
	}

	// Get space ID from board.
	var board models.Board
	if err := e.db.WithContext(ctx).Select("space_id").First(&board, "id = ?", boardID).Error; err != nil {
		return "", fmt.Errorf("looking up board: %w", err)
	}

	return e.resolveForSpace(ctx, userID, board.SpaceID)
}

// resolveForSpace checks space → org membership.
func (e *RBACEngine) resolveForSpace(ctx context.Context, userID, spaceID string) (models.Role, error) {
	// Check space membership.
	var spaceMembership models.SpaceMembership
	err := e.db.WithContext(ctx).
		Where("space_id = ? AND user_id = ?", spaceID, userID).
		First(&spaceMembership).Error
	if err == nil {
		return spaceMembership.Role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("querying space membership: %w", err)
	}

	// Get org ID from space.
	var space models.Space
	if err := e.db.WithContext(ctx).Select("org_id").First(&space, "id = ?", spaceID).Error; err != nil {
		return "", fmt.Errorf("looking up space: %w", err)
	}

	return e.resolveForOrg(ctx, userID, space.OrgID)
}

// resolveForOrg checks org membership.
func (e *RBACEngine) resolveForOrg(ctx context.Context, userID, orgID string) (models.Role, error) {
	var orgMembership models.OrgMembership
	err := e.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		First(&orgMembership).Error
	if err == nil {
		return orgMembership.Role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("querying org membership: %w", err)
	}
	return "", nil // No membership found.
}

// HasPermission checks if the given role has a specific permission.
func (e *RBACEngine) HasPermission(role models.Role, permission string) bool {
	return e.policy.HasPermission(string(role), permission)
}

// IsHigherOrEqual returns true if roleA has equal or higher rank than roleB.
func (e *RBACEngine) IsHigherOrEqual(roleA, roleB models.Role) bool {
	return e.policy.IsHigherOrEqual(string(roleA), string(roleB))
}

// LookupOrgForEntity finds the org ID for any entity in the hierarchy.
func (e *RBACEngine) LookupOrgForEntity(ctx context.Context, entityType, entityID string) (string, error) {
	switch entityType {
	case "org":
		return entityID, nil
	case "space":
		var space models.Space
		if err := e.db.WithContext(ctx).Select("org_id").First(&space, "id = ?", entityID).Error; err != nil {
			return "", fmt.Errorf("looking up space: %w", err)
		}
		return space.OrgID, nil
	case "board":
		var board models.Board
		if err := e.db.WithContext(ctx).Select("space_id").First(&board, "id = ?", entityID).Error; err != nil {
			return "", fmt.Errorf("looking up board: %w", err)
		}
		var space models.Space
		if err := e.db.WithContext(ctx).Select("org_id").First(&space, "id = ?", board.SpaceID).Error; err != nil {
			return "", fmt.Errorf("looking up space for board: %w", err)
		}
		return space.OrgID, nil
	case "thread":
		var thread models.Thread
		if err := e.db.WithContext(ctx).Select("board_id").First(&thread, "id = ?", entityID).Error; err != nil {
			return "", fmt.Errorf("looking up thread: %w", err)
		}
		return e.LookupOrgForEntity(ctx, "board", thread.BoardID)
	case "message":
		var message models.Message
		if err := e.db.WithContext(ctx).Select("thread_id").First(&message, "id = ?", entityID).Error; err != nil {
			return "", fmt.Errorf("looking up message: %w", err)
		}
		return e.LookupOrgForEntity(ctx, "thread", message.ThreadID)
	default:
		return "", fmt.Errorf("unknown entity type: %s", entityType)
	}
}
