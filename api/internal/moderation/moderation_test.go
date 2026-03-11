package moderation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

type testHierarchy struct {
	org     *models.Org
	space   *models.Space
	board   *models.Board
	board2  *models.Board
	thread  *models.Thread
	thread2 *models.Thread
}

func seedHierarchy(t *testing.T, db *gorm.DB) *testHierarchy {
	t.Helper()
	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "Space", Slug: "space", Metadata: "{}", Type: "general"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Board 1", Slug: "board-1", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	board2 := &models.Board{SpaceID: space.ID, Name: "Board 2", Slug: "board-2", Metadata: "{}"}
	require.NoError(t, db.Create(board2).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "Thread 1", Slug: "thread-1", AuthorID: "author1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	thread2 := &models.Thread{BoardID: board.ID, Title: "Thread 2", Slug: "thread-2", AuthorID: "author1", Metadata: "{}"}
	require.NoError(t, db.Create(thread2).Error)
	return &testHierarchy{org: org, space: space, board: board, board2: board2, thread: thread, thread2: thread2}
}

// --- Flag Tests ---

func TestService_CreateFlag(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)
	assert.NotEmpty(t, flag.ID)
	assert.Equal(t, models.FlagStatusOpen, flag.Status)
	assert.Equal(t, "spam", flag.Reason)
	assert.Equal(t, "user1", flag.UserID)
}

func TestService_CreateFlag_MissingThreadID(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.CreateFlag(ctx, "user1", FlagInput{Reason: "spam"})
	assert.Error(t, err)
	assert.Equal(t, "thread_id is required", err.Error())
}

func TestService_CreateFlag_MissingReason(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID})
	assert.Error(t, err)
	assert.Equal(t, "reason is required", err.Error())
}

func TestService_CreateFlag_ThreadNotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: "nonexistent", Reason: "spam"})
	assert.Error(t, err)
	assert.Equal(t, "thread not found", err.Error())
}

func TestService_ListOrgFlags(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	// Create two flags.
	_, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)
	_, err = svc.CreateFlag(ctx, "user2", FlagInput{ThreadID: h.thread2.ID, Reason: "offensive"})
	require.NoError(t, err)

	flags, pageInfo, err := svc.ListOrgFlags(ctx, h.org.ID, defaultPaginationParams())
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListOrgFlags_Empty(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flags, pageInfo, err := svc.ListOrgFlags(ctx, h.org.ID, defaultPaginationParams())
	require.NoError(t, err)
	assert.Empty(t, flags)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ResolveFlag(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)

	resolved, err := svc.ResolveFlag(ctx, flag.ID, "mod1")
	require.NoError(t, err)
	assert.Equal(t, models.FlagStatusResolved, resolved.Status)
	assert.Equal(t, "mod1", resolved.ResolvedBy)
}

func TestService_DismissFlag(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)

	dismissed, err := svc.DismissFlag(ctx, flag.ID, "mod1")
	require.NoError(t, err)
	assert.Equal(t, models.FlagStatusDismissed, dismissed.Status)
	assert.Equal(t, "mod1", dismissed.ResolvedBy)
}

func TestService_ResolveFlag_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.ResolveFlag(ctx, "nonexistent", "mod1")
	assert.Error(t, err)
	assert.Equal(t, "flag not found", err.Error())
}

func TestService_ResolveFlag_AlreadyResolved(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)

	_, err = svc.ResolveFlag(ctx, flag.ID, "mod1")
	require.NoError(t, err)

	_, err = svc.ResolveFlag(ctx, flag.ID, "mod2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestService_ResolvedFlagNotInQueue(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)
	_, err = svc.ResolveFlag(ctx, flag.ID, "mod1")
	require.NoError(t, err)

	flags, _, err := svc.ListOrgFlags(ctx, h.org.ID, defaultPaginationParams())
	require.NoError(t, err)
	assert.Empty(t, flags)
}

// --- Move Thread Tests ---

func TestService_MoveThread(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	moved, err := svc.MoveThread(ctx, h.thread.ID, "mod1", MoveInput{TargetBoardID: h.board2.ID})
	require.NoError(t, err)
	assert.Equal(t, h.board2.ID, moved.BoardID)
}

func TestService_MoveThread_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MoveThread(ctx, "nonexistent", "mod1", MoveInput{TargetBoardID: h.board2.ID})
	assert.Error(t, err)
	assert.Equal(t, "thread not found", err.Error())
}

func TestService_MoveThread_SameBoard(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MoveThread(ctx, h.thread.ID, "mod1", MoveInput{TargetBoardID: h.board.ID})
	assert.Error(t, err)
	assert.Equal(t, "thread is already in the target board", err.Error())
}

func TestService_MoveThread_TargetBoardNotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MoveThread(ctx, h.thread.ID, "mod1", MoveInput{TargetBoardID: "nonexistent"})
	assert.Error(t, err)
	assert.Equal(t, "target board not found", err.Error())
}

func TestService_MoveThread_MissingTarget(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MoveThread(ctx, h.thread.ID, "mod1", MoveInput{})
	assert.Error(t, err)
	assert.Equal(t, "target_board_id is required", err.Error())
}

// --- Merge Thread Tests ---

func TestService_MergeThread(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	// Add a message to the source thread.
	msg := &models.Message{ThreadID: h.thread.ID, Body: "Hello", AuthorID: "author1", Type: "comment", Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	target, err := svc.MergeThread(ctx, h.thread.ID, "mod1", MergeInput{TargetThreadID: h.thread2.ID})
	require.NoError(t, err)
	assert.Equal(t, h.thread2.ID, target.ID)

	// Source should be soft-deleted.
	source, err := repo.FindThreadByID(ctx, h.thread.ID)
	assert.NoError(t, err)
	assert.Nil(t, source) // Soft-deleted, not found.

	// Message should now be on target.
	var movedMsg models.Message
	require.NoError(t, db.First(&movedMsg, "id = ?", msg.ID).Error)
	assert.Equal(t, h.thread2.ID, movedMsg.ThreadID)
}

func TestService_MergeThread_SelfMerge(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MergeThread(ctx, h.thread.ID, "mod1", MergeInput{TargetThreadID: h.thread.ID})
	assert.Error(t, err)
	assert.Equal(t, "cannot merge a thread into itself", err.Error())
}

func TestService_MergeThread_TargetNotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MergeThread(ctx, h.thread.ID, "mod1", MergeInput{TargetThreadID: "nonexistent"})
	assert.Error(t, err)
	assert.Equal(t, "target thread not found", err.Error())
}

func TestService_MergeThread_SourceNotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MergeThread(ctx, "nonexistent", "mod1", MergeInput{TargetThreadID: h.thread2.ID})
	assert.Error(t, err)
	assert.Equal(t, "source thread not found", err.Error())
}

func TestService_MergeThread_MissingTarget(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MergeThread(ctx, h.thread.ID, "mod1", MergeInput{})
	assert.Error(t, err)
	assert.Equal(t, "target_thread_id is required", err.Error())
}

// --- Hide/Unhide Tests ---

func TestService_HideThread(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	hidden, err := svc.HideThread(ctx, h.thread.ID, "mod1")
	require.NoError(t, err)
	assert.True(t, hidden.IsHidden)
}

func TestService_UnhideThread(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.HideThread(ctx, h.thread.ID, "mod1")
	require.NoError(t, err)

	unhidden, err := svc.UnhideThread(ctx, h.thread.ID, "mod1")
	require.NoError(t, err)
	assert.False(t, unhidden.IsHidden)
}

func TestService_HideThread_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.HideThread(ctx, "nonexistent", "mod1")
	assert.Error(t, err)
	assert.Equal(t, "thread not found", err.Error())
}

func TestService_UnhideThread_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.UnhideThread(ctx, "nonexistent", "mod1")
	assert.Error(t, err)
	assert.Equal(t, "thread not found", err.Error())
}

// --- Audit Log Tests ---

func TestService_CreateFlag_AuditLogged(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)

	var logs []models.AuditLog
	require.NoError(t, db.Where("entity_type = ?", "flag").Find(&logs).Error)
	assert.GreaterOrEqual(t, len(logs), 1)
	assert.Equal(t, "user1", logs[0].UserID)
	assert.Equal(t, models.AuditActionCreate, logs[0].Action)
}

func TestService_MoveThread_AuditLogged(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.MoveThread(ctx, h.thread.ID, "mod1", MoveInput{TargetBoardID: h.board2.ID})
	require.NoError(t, err)

	var logs []models.AuditLog
	require.NoError(t, db.Where("entity_type = ? AND entity_id = ?", "thread", h.thread.ID).Find(&logs).Error)
	assert.GreaterOrEqual(t, len(logs), 1)
	assert.Equal(t, "mod1", logs[0].UserID)
}

func TestService_HideThread_AuditLogged(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	_, err := svc.HideThread(ctx, h.thread.ID, "mod1")
	require.NoError(t, err)

	var logs []models.AuditLog
	require.NoError(t, db.Where("entity_type = ? AND entity_id = ?", "thread", h.thread.ID).Find(&logs).Error)
	assert.GreaterOrEqual(t, len(logs), 1)
}

// --- Repository Tests ---

func TestRepository_CreateFlag(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag := &models.Flag{ThreadID: h.thread.ID, UserID: "user1", Reason: "test", Status: models.FlagStatusOpen}
	require.NoError(t, repo.CreateFlag(ctx, flag))
	assert.NotEmpty(t, flag.ID)
}

func TestRepository_FindFlagByID(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag := &models.Flag{ThreadID: h.thread.ID, UserID: "user1", Reason: "test", Status: models.FlagStatusOpen}
	require.NoError(t, repo.CreateFlag(ctx, flag))

	found, err := repo.FindFlagByID(ctx, flag.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, flag.ID, found.ID)
}

func TestRepository_FindFlagByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	found, err := repo.FindFlagByID(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_UpdateFlag(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag := &models.Flag{ThreadID: h.thread.ID, UserID: "user1", Reason: "test", Status: models.FlagStatusOpen}
	require.NoError(t, repo.CreateFlag(ctx, flag))

	flag.Status = models.FlagStatusResolved
	flag.ResolvedBy = "mod1"
	require.NoError(t, repo.UpdateFlag(ctx, flag))

	found, err := repo.FindFlagByID(ctx, flag.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FlagStatusResolved, found.Status)
}

func TestRepository_FindThreadByID(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	found, err := repo.FindThreadByID(ctx, h.thread.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, h.thread.Title, found.Title)
}

func TestRepository_FindThreadByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	found, err := repo.FindThreadByID(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_FindBoardByID(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	found, err := repo.FindBoardByID(ctx, h.board.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, h.board.Name, found.Name)
}

func TestRepository_FindBoardByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	found, err := repo.FindBoardByID(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_MoveMessages(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	msg := &models.Message{ThreadID: h.thread.ID, Body: "test", AuthorID: "a", Type: "comment", Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	require.NoError(t, repo.MoveMessages(ctx, h.thread.ID, h.thread2.ID))

	var moved models.Message
	require.NoError(t, db.First(&moved, "id = ?", msg.ID).Error)
	assert.Equal(t, h.thread2.ID, moved.ThreadID)
}

func TestRepository_SoftDeleteThread(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	require.NoError(t, repo.SoftDeleteThread(ctx, h.thread.ID))

	found, err := repo.FindThreadByID(ctx, h.thread.ID)
	assert.NoError(t, err)
	assert.Nil(t, found) // Soft-deleted.
}

func TestRepository_CreateAuditLog(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	log := &models.AuditLog{
		UserID:     "user1",
		Action:     models.AuditActionCreate,
		EntityType: "test",
		EntityID:   "test-id",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, log))
	assert.NotEmpty(t, log.ID)
}

func TestMustJSON(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"map", map[string]string{"key": "val"}, `{"key":"val"}`},
		{"bool", true, "true"},
		{"nil", nil, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mustJSON(tt.v))
		})
	}
}

func TestFlagStatus_IsValid(t *testing.T) {
	assert.True(t, models.FlagStatusOpen.IsValid())
	assert.True(t, models.FlagStatusResolved.IsValid())
	assert.True(t, models.FlagStatusDismissed.IsValid())
	assert.False(t, models.FlagStatus("unknown").IsValid())
}

func TestService_ListOrgFlags_CursorPagination(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	// Create 3 flags.
	for i := 0; i < 3; i++ {
		_, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "reason"})
		require.NoError(t, err)
	}

	// Fetch first page of 2.
	flags, pageInfo, err := svc.ListOrgFlags(ctx, h.org.ID, pagination.Params{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)

	// Fetch second page using cursor.
	flags2, pageInfo2, err := svc.ListOrgFlags(ctx, h.org.ID, pagination.Params{Limit: 2, Cursor: pageInfo.NextCursor})
	require.NoError(t, err)
	assert.Len(t, flags2, 1)
	assert.False(t, pageInfo2.HasMore)
}

func TestService_DismissFlag_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.DismissFlag(ctx, "nonexistent", "mod1")
	assert.Error(t, err)
	assert.Equal(t, "flag not found", err.Error())
}

func TestService_DismissFlag_AlreadyDismissed(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	h := seedHierarchy(t, db)
	ctx := context.Background()

	flag, err := svc.CreateFlag(ctx, "user1", FlagInput{ThreadID: h.thread.ID, Reason: "spam"})
	require.NoError(t, err)

	_, err = svc.DismissFlag(ctx, flag.ID, "mod1")
	require.NoError(t, err)

	_, err = svc.DismissFlag(ctx, flag.ID, "mod2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestRepository_UpdateThread(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := seedHierarchy(t, db)
	ctx := context.Background()

	h.thread.IsHidden = true
	require.NoError(t, repo.UpdateThread(ctx, h.thread))

	updated, err := repo.FindThreadByID(ctx, h.thread.ID)
	require.NoError(t, err)
	assert.True(t, updated.IsHidden)
}

// defaultPaginationParams returns default pagination params for tests.
func defaultPaginationParams() pagination.Params {
	return pagination.Params{Limit: 50}
}
