package board

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

func createSpace(t *testing.T, db *gorm.DB) *models.Space {
	t.Helper()
	org := &models.Org{Name: "test-org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	sp := &models.Space{OrgID: org.ID, Name: "test-space", Slug: "test-space", Metadata: "{}", Type: models.SpaceTypeGeneral}
	require.NoError(t, db.Create(sp).Error)
	return sp
}

func TestService_Create_Valid(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	b, err := svc.Create(context.Background(), sp.ID, CreateInput{Name: "My Board"})
	require.NoError(t, err)
	assert.Equal(t, "My Board", b.Name)
	assert.Equal(t, "my-board", b.Slug)
	assert.Equal(t, sp.ID, b.SpaceID)
	assert.False(t, b.IsLocked)
}

func TestService_Create_EmptyName(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	_, err := svc.Create(context.Background(), sp.ID, CreateInput{Name: ""})
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_InvalidMetadata(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	_, err := svc.Create(context.Background(), sp.ID, CreateInput{Name: "Board", Metadata: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidMeta)
}

func TestService_Create_SlugDedup(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	b1, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Dupe"})
	require.NoError(t, err)
	assert.Equal(t, "dupe", b1.Slug)

	b2, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Dupe"})
	require.NoError(t, err)
	assert.Equal(t, "dupe-1", b2.Slug)
}

func TestService_GetByRef(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Get Board"})
	require.NoError(t, err)

	got, err := svc.GetByRef(ctx, sp.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	got, err = svc.GetByRef(ctx, sp.ID, "get-board")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestService_GetByRef_NotFound(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	_, err := svc.GetByRef(context.Background(), sp.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_List(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Board " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	boards, hasMore, err := svc.List(ctx, sp.ID, "", 2)
	require.NoError(t, err)
	assert.Len(t, boards, 2)
	assert.True(t, hasMore)
}

func TestService_Update(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Original", Metadata: `{"a":"1"}`})
	require.NoError(t, err)

	newName := "Updated"
	meta := `{"b":"2"}`
	updated, err := svc.Update(ctx, sp.ID, created.ID, UpdateInput{Name: &newName, Metadata: &meta})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.JSONEq(t, `{"a":"1","b":"2"}`, updated.Metadata)
}

func TestService_Update_NotFound(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	name := "X"
	_, err := svc.Update(context.Background(), sp.ID, "nonexistent", UpdateInput{Name: &name})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Update_EmptyName(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Test"})
	require.NoError(t, err)

	empty := ""
	_, err = svc.Update(ctx, sp.ID, created.ID, UpdateInput{Name: &empty})
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_SetLock(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Lock Board"})
	require.NoError(t, err)
	assert.False(t, created.IsLocked)

	locked, err := svc.SetLock(ctx, sp.ID, created.ID, true)
	require.NoError(t, err)
	assert.True(t, locked.IsLocked)

	unlocked, err := svc.SetLock(ctx, sp.ID, created.ID, false)
	require.NoError(t, err)
	assert.False(t, unlocked.IsLocked)
}

func TestService_SetLock_NotFound(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	_, err := svc.SetLock(context.Background(), sp.ID, "nonexistent", true)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, sp.ID, CreateInput{Name: "To Delete"})
	require.NoError(t, err)

	err = svc.Delete(ctx, sp.ID, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByRef(ctx, sp.ID, created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	svc := NewService(NewRepository(db))

	err := svc.Delete(context.Background(), sp.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_GetByID(t *testing.T) {
	db := testDB(t)
	sp := createSpace(t, db)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	created, err := svc.Create(ctx, sp.ID, CreateInput{Name: "Get Board"})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

// --- Fuzz Tests ---

func FuzzServiceCreate(f *testing.F) {
	f.Add("Valid Board", "{}")
	f.Add("", "{}")
	f.Add("Test", "invalid-json")
	f.Add("Board!", `{"k":"v"}`)
	f.Add("A B C", "")
	f.Fuzz(func(t *testing.T, name, meta string) {
		db := testDB(t)
		sp := createSpace(t, db)
		svc := NewService(NewRepository(db))
		_, _ = svc.Create(context.Background(), sp.ID, CreateInput{Name: name, Metadata: meta})
	})
}
