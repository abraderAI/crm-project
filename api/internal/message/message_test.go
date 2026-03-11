package message

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

func createThread(t *testing.T, db *gorm.DB) *models.Thread {
	t.Helper()
	org := &models.Org{Name: "test-org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	sp := &models.Space{OrgID: org.ID, Name: "test-space", Slug: "test-space", Metadata: "{}", Type: models.SpaceTypeGeneral}
	require.NoError(t, db.Create(sp).Error)
	b := &models.Board{SpaceID: sp.ID, Name: "test-board", Slug: "test-board", Metadata: "{}"}
	require.NoError(t, db.Create(b).Error)
	th := &models.Thread{BoardID: b.ID, Title: "test-thread", Slug: "test-thread", Metadata: "{}", AuthorID: "user1"}
	require.NoError(t, db.Create(th).Error)
	return th
}

// stubThreadChecker implements ThreadChecker for tests.
type stubThreadChecker struct{ locked bool }

func (s *stubThreadChecker) IsLocked(_ context.Context, _ string) (bool, error) {
	return s.locked, nil
}

func TestService_Create_Valid(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})

	m, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "Hello"})
	require.NoError(t, err)
	assert.Equal(t, "Hello", m.Body)
	assert.Equal(t, th.ID, m.ThreadID)
	assert.Equal(t, "user1", m.AuthorID)
	assert.Equal(t, models.MessageTypeComment, m.Type)
}

func TestService_Create_WithType(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})

	m, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "Note", Type: "note"})
	require.NoError(t, err)
	assert.Equal(t, models.MessageTypeNote, m.Type)
}

func TestService_Create_InvalidType(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})

	_, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "X", Type: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidType)
}

func TestService_Create_EmptyBody(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})

	_, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: ""})
	assert.ErrorIs(t, err, ErrBodyRequired)
}

func TestService_Create_ThreadLocked(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: true})

	_, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "Hello"})
	assert.ErrorIs(t, err, ErrThreadLocked)
}

func TestService_Create_InvalidMetadata(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})

	_, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "Hello", Metadata: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidMeta)
}

func TestService_GetByID(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, th.ID, "user1", CreateInput{Body: "Find Me"})
	require.NoError(t, err)

	got, err := svc.GetByID(ctx, th.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestService_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)

	_, err := svc.GetByID(context.Background(), th.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_List(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, th.ID, "user1", CreateInput{Body: "Msg " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	messages, hasMore, err := svc.List(ctx, th.ID, "", 2)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.True(t, hasMore)
}

func TestService_Update_AuthorOnly(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, th.ID, "author1", CreateInput{Body: "Original"})
	require.NoError(t, err)

	// Author can update.
	newBody := "Updated"
	updated, err := svc.Update(ctx, th.ID, created.ID, "author1", UpdateInput{Body: &newBody})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Body)

	// Non-author cannot update.
	_, err = svc.Update(ctx, th.ID, created.ID, "other-user", UpdateInput{Body: &newBody})
	assert.ErrorIs(t, err, ErrNotAuthor)
}

func TestService_Update_EmptyBody(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, th.ID, "user1", CreateInput{Body: "Original"})
	require.NoError(t, err)

	empty := ""
	_, err = svc.Update(ctx, th.ID, created.ID, "user1", UpdateInput{Body: &empty})
	assert.ErrorIs(t, err, ErrBodyRequired)
}

func TestService_Update_WithRevision(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, th.ID, "user1", CreateInput{Body: "Original"})
	require.NoError(t, err)

	newBody := "V2"
	_, err = svc.Update(ctx, th.ID, created.ID, "user1", UpdateInput{Body: &newBody})
	require.NoError(t, err)

	// Verify revision was created.
	var revisions []models.Revision
	require.NoError(t, db.Where("entity_type = ? AND entity_id = ?", "message", created.ID).Find(&revisions).Error)
	assert.Len(t, revisions, 1)
	assert.Equal(t, 1, revisions[0].Version)
}

func TestService_Update_MetadataDeepMerge(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, th.ID, "user1", CreateInput{Body: "Msg", Metadata: `{"a":"1"}`})
	require.NoError(t, err)

	meta := `{"b":"2"}`
	updated, err := svc.Update(ctx, th.ID, created.ID, "user1", UpdateInput{Metadata: &meta})
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"1","b":"2"}`, updated.Metadata)
}

func TestService_Update_NotFound(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)

	body := "X"
	_, err := svc.Update(context.Background(), th.ID, "nonexistent", "user1", UpdateInput{Body: &body})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
	ctx := context.Background()

	created, err := svc.Create(ctx, th.ID, "user1", CreateInput{Body: "To Delete"})
	require.NoError(t, err)

	err = svc.Delete(ctx, th.ID, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(ctx, th.ID, created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)

	err := svc.Delete(context.Background(), th.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Create_NilThreadChecker(t *testing.T) {
	db := testDB(t)
	th := createThread(t, db)
	svc := NewService(NewRepository(db), nil)

	m, err := svc.Create(context.Background(), th.ID, "user1", CreateInput{Body: "No Checker"})
	require.NoError(t, err)
	assert.Equal(t, "No Checker", m.Body)
}

// --- Fuzz Tests ---

func FuzzServiceCreate(f *testing.F) {
	f.Add("Hello World", "{}", "comment")
	f.Add("", "{}", "")
	f.Add("Test body", "invalid", "note")
	f.Add("Msg!", `{"k":"v"}`, "invalid_type")
	f.Add("A B C", "", "email")
	f.Fuzz(func(t *testing.T, body, meta, msgType string) {
		db := testDB(t)
		th := createThread(t, db)
		svc := NewService(NewRepository(db), &stubThreadChecker{locked: false})
		_, _ = svc.Create(context.Background(), th.ID, "fuzz-user", CreateInput{Body: body, Metadata: meta, Type: msgType})
	})
}
