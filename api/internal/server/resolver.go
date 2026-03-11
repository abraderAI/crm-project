package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Resolver provides entity ID resolution for hierarchical URL params.
// It implements OrgGetter, SpaceGetter, BoardGetter, and ThreadGetter interfaces.
type Resolver struct {
	db *gorm.DB
}

// NewResolver creates a new resolver.
func NewResolver(db *gorm.DB) *Resolver {
	return &Resolver{db: db}
}

// ResolveOrgID resolves an org ref (ID or slug) to an org ID.
func (r *Resolver) ResolveOrgID(ctx context.Context, ref string) (string, error) {
	var org models.Org
	q := r.db.WithContext(ctx).Select("id")
	if isUUID(ref) {
		q = q.Where("id = ?", ref)
	} else {
		q = q.Where("slug = ?", ref)
	}
	if err := q.First(&org).Error; err != nil {
		return "", fmt.Errorf("resolving org: %w", err)
	}
	return org.ID, nil
}

// ResolveSpaceID resolves an org+space ref to a space ID.
func (r *Resolver) ResolveSpaceID(ctx context.Context, orgRef, spaceRef string) (string, error) {
	orgID, err := r.ResolveOrgID(ctx, orgRef)
	if err != nil {
		return "", err
	}
	var space models.Space
	q := r.db.WithContext(ctx).Select("id").Where("org_id = ?", orgID)
	if isUUID(spaceRef) {
		q = q.Where("id = ?", spaceRef)
	} else {
		q = q.Where("slug = ?", spaceRef)
	}
	if err := q.First(&space).Error; err != nil {
		return "", fmt.Errorf("resolving space: %w", err)
	}
	return space.ID, nil
}

// ResolveBoardID resolves an org+space+board ref to a board ID.
func (r *Resolver) ResolveBoardID(ctx context.Context, orgRef, spaceRef, boardRef string) (string, error) {
	spaceID, err := r.ResolveSpaceID(ctx, orgRef, spaceRef)
	if err != nil {
		return "", err
	}
	var board models.Board
	q := r.db.WithContext(ctx).Select("id").Where("space_id = ?", spaceID)
	if isUUID(boardRef) {
		q = q.Where("id = ?", boardRef)
	} else {
		q = q.Where("slug = ?", boardRef)
	}
	if err := q.First(&board).Error; err != nil {
		return "", fmt.Errorf("resolving board: %w", err)
	}
	return board.ID, nil
}

// ResolveThreadID resolves a full hierarchy ref to a thread ID.
func (r *Resolver) ResolveThreadID(ctx context.Context, orgRef, spaceRef, boardRef, threadRef string) (string, error) {
	boardID, err := r.ResolveBoardID(ctx, orgRef, spaceRef, boardRef)
	if err != nil {
		return "", err
	}
	var thread models.Thread
	q := r.db.WithContext(ctx).Select("id").Where("board_id = ?", boardID)
	if isUUID(threadRef) {
		q = q.Where("id = ?", threadRef)
	} else {
		q = q.Where("slug = ?", threadRef)
	}
	if err := q.First(&thread).Error; err != nil {
		return "", fmt.Errorf("resolving thread: %w", err)
	}
	return thread.ID, nil
}

// BoardLockChecker checks if a board is locked.
type BoardLockChecker struct {
	db *gorm.DB
}

// NewBoardLockChecker creates a new board lock checker.
func NewBoardLockChecker(db *gorm.DB) *BoardLockChecker {
	return &BoardLockChecker{db: db}
}

// IsLocked returns true if the board is locked.
func (c *BoardLockChecker) IsLocked(ctx context.Context, boardID string) (bool, error) {
	var board models.Board
	if err := c.db.WithContext(ctx).Select("is_locked").First(&board, "id = ?", boardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return board.IsLocked, nil
}

// ThreadLockChecker checks if a thread is locked.
type ThreadLockChecker struct {
	db *gorm.DB
}

// NewThreadLockChecker creates a new thread lock checker.
func NewThreadLockChecker(db *gorm.DB) *ThreadLockChecker {
	return &ThreadLockChecker{db: db}
}

// IsLocked returns true if the thread is locked.
func (c *ThreadLockChecker) IsLocked(ctx context.Context, threadID string) (bool, error) {
	var thread models.Thread
	if err := c.db.WithContext(ctx).Select("is_locked").First(&thread, "id = ?", threadID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return thread.IsLocked, nil
}

// MemberRepoAdapter adapts the membership repository for the org handler.
type MemberRepoAdapter struct {
	db *gorm.DB
}

// NewMemberRepoAdapter creates a new member repo adapter.
func NewMemberRepoAdapter(db *gorm.DB) *MemberRepoAdapter {
	return &MemberRepoAdapter{db: db}
}

// CreateOrgMembership creates an org membership.
func (a *MemberRepoAdapter) CreateOrgMembership(ctx interface{ Value(any) any }, orgID, userID string, role models.Role) error {
	m := &models.OrgMembership{OrgID: orgID, UserID: userID, Role: role}
	return a.db.WithContext(ctx.(context.Context)).Create(m).Error
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
