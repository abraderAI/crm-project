package thread

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
	"github.com/abraderAI/crm-project/api/pkg/metadata"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() { sqlDB.Close() })
	return db
}

func createBoard(t *testing.T, db *gorm.DB) *models.Board {
	t.Helper()
	org := &models.Org{Name: "test-org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	sp := &models.Space{OrgID: org.ID, Name: "test-space", Slug: "test-space", Metadata: "{}", Type: models.SpaceTypeGeneral}
	require.NoError(t, db.Create(sp).Error)
	b := &models.Board{SpaceID: sp.ID, Name: "test-board", Slug: "test-board", Metadata: "{}"}
	require.NoError(t, db.Create(b).Error)
	return b
}

// stubBoardChecker implements BoardChecker for tests.
type stubBoardChecker struct{ locked bool }

func (s *stubBoardChecker) IsLocked(_ context.Context, _ string) (bool, error) {
	return s.locked, nil
}

func TestService_Create_Valid(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})

	th, err := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "My Thread"})
	require.NoError(t, err)
	assert.Equal(t, "My Thread", th.Title)
	assert.Equal(t, "my-thread", th.Slug)
	assert.Equal(t, board.ID, th.BoardID)
	assert.Equal(t, "user1", th.AuthorID)
	assert.False(t, th.IsPinned)
	assert.False(t, th.IsLocked)
}

func TestService_Create_EmptyTitle(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})

	_, err := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: ""})
	assert.ErrorIs(t, err, ErrTitleRequired)
}

func TestService_Create_BoardLocked(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: true})

	_, err := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Thread"})
	assert.ErrorIs(t, err, ErrBoardLocked)
}

func TestService_Create_InvalidMetadata(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})

	_, err := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "Thread", Metadata: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidMeta)
}

func TestService_Create_SlugDedup(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	t1, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Dupe"})
	require.NoError(t, err)
	assert.Equal(t, "dupe", t1.Slug)

	t2, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Dupe"})
	require.NoError(t, err)
	assert.Equal(t, "dupe-1", t2.Slug)
}

func TestService_GetByRef(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Get Thread"})
	require.NoError(t, err)

	got, err := svc.GetByRef(ctx, board.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	got, err = svc.GetByRef(ctx, board.ID, "get-thread")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestService_GetByRef_NotFound(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)

	_, err := svc.GetByRef(context.Background(), board.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_List(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Thread " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	threads, hasMore, err := svc.List(ctx, board.ID, "", 2, nil)
	require.NoError(t, err)
	assert.Len(t, threads, 2)
	assert.True(t, hasMore)
}

func TestService_List_WithMetadataFilter(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	_, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Open", Metadata: `{"status":"open"}`})
	require.NoError(t, err)
	_, err = svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Closed", Metadata: `{"status":"closed"}`})
	require.NoError(t, err)

	filters := []metadata.Filter{{Key: "status", Operator: "eq", Value: "open"}}
	threads, _, err := svc.List(ctx, board.ID, "", 10, filters)
	require.NoError(t, err)
	assert.Len(t, threads, 1)
	assert.Equal(t, "Open", threads[0].Title)
}

func TestService_Update_WithRevision(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Original", Metadata: `{"a":"1"}`})
	require.NoError(t, err)

	newTitle := "Updated"
	meta := `{"b":"2"}`
	updated, err := svc.Update(ctx, board.ID, created.ID, "user1", UpdateInput{Title: &newTitle, Metadata: &meta})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)
	assert.JSONEq(t, `{"a":"1","b":"2"}`, updated.Metadata)

	// Verify revision was created.
	var revisions []models.Revision
	require.NoError(t, db.Where("entity_type = ? AND entity_id = ?", "thread", created.ID).Find(&revisions).Error)
	assert.Len(t, revisions, 1)
	assert.Equal(t, 1, revisions[0].Version)
}

func TestService_Update_EmptyTitle(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Test"})
	require.NoError(t, err)

	empty := ""
	_, err = svc.Update(ctx, board.ID, created.ID, "user1", UpdateInput{Title: &empty})
	assert.ErrorIs(t, err, ErrTitleRequired)
}

func TestService_Update_NotFound(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)

	title := "X"
	_, err := svc.Update(context.Background(), board.ID, "nonexistent", "user1", UpdateInput{Title: &title})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_SetPin(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Pin Thread"})
	require.NoError(t, err)
	assert.False(t, created.IsPinned)

	pinned, err := svc.SetPin(ctx, board.ID, created.ID, true)
	require.NoError(t, err)
	assert.True(t, pinned.IsPinned)

	unpinned, err := svc.SetPin(ctx, board.ID, created.ID, false)
	require.NoError(t, err)
	assert.False(t, unpinned.IsPinned)
}

func TestService_SetPin_NotFound(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)

	_, err := svc.SetPin(context.Background(), board.ID, "nonexistent", true)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_SetLock(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "Lock Thread"})
	require.NoError(t, err)

	locked, err := svc.SetLock(ctx, board.ID, created.ID, true)
	require.NoError(t, err)
	assert.True(t, locked.IsLocked)

	unlocked, err := svc.SetLock(ctx, board.ID, created.ID, false)
	require.NoError(t, err)
	assert.False(t, unlocked.IsLocked)
}

func TestService_Delete(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, board.ID, "user1", CreateInput{Title: "To Delete"})
	require.NoError(t, err)

	err = svc.Delete(ctx, board.ID, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByRef(ctx, board.ID, created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)

	err := svc.Delete(context.Background(), board.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Create_NilBoardChecker(t *testing.T) {
	db := testDB(t)
	board := createBoard(t, db)
	svc := NewService(NewRepository(db), nil)

	th, err := svc.Create(context.Background(), board.ID, "user1", CreateInput{Title: "No Checker"})
	require.NoError(t, err)
	assert.Equal(t, "No Checker", th.Title)
}

// --- Fuzz Tests ---

func FuzzServiceCreate(f *testing.F) {
	f.Add("Valid Thread", "{}", "body text")
	f.Add("", "{}", "")
	f.Add("Test", "invalid", "body")
	f.Add("Thread!", `{"k":"v"}`, "")
	f.Add("A B C", "", "something")
	f.Fuzz(func(t *testing.T, title, meta, body string) {
		db := testDB(t)
		board := createBoard(t, db)
		svc := NewService(NewRepository(db), &stubBoardChecker{locked: false})
		_, _ = svc.Create(context.Background(), board.ID, "fuzz-user", CreateInput{Title: title, Metadata: meta, Body: body})
	})
}
