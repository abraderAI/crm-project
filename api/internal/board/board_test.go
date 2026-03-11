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
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func setupDB(t *testing.T) (*gorm.DB, string) {
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

	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	sp := &models.Space{OrgID: org.ID, Name: "Test Space", Slug: "test-space", Metadata: "{}", Type: models.SpaceTypeGeneral}
	require.NoError(t, db.Create(sp).Error)
	return db, sp.ID
}

func TestBoardService_Create(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		b, err := svc.Create(ctx, spaceID, CreateInput{Name: "Feature Board"})
		require.NoError(t, err)
		assert.NotEmpty(t, b.ID)
		assert.Equal(t, "Feature Board", b.Name)
		assert.Equal(t, "feature-board", b.Slug)
		assert.Equal(t, spaceID, b.SpaceID)
		assert.False(t, b.IsLocked)
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, spaceID, CreateInput{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("invalid metadata", func(t *testing.T) {
		_, err := svc.Create(ctx, spaceID, CreateInput{Name: "Bad", Metadata: "not json"})
		assert.Error(t, err)
	})

	t.Run("duplicate slug suffix", func(t *testing.T) {
		b1, err := svc.Create(ctx, spaceID, CreateInput{Name: "Dup Board"})
		require.NoError(t, err)
		b2, err := svc.Create(ctx, spaceID, CreateInput{Name: "Dup Board"})
		require.NoError(t, err)
		assert.NotEqual(t, b1.Slug, b2.Slug)
		assert.Equal(t, "dup-board-2", b2.Slug)
	})

	t.Run("with metadata", func(t *testing.T) {
		b, err := svc.Create(ctx, spaceID, CreateInput{Name: "Meta Board", Metadata: `{"tag":"test"}`})
		require.NoError(t, err)
		assert.Contains(t, b.Metadata, "tag")
	})
}

func TestBoardService_Get(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "Lookup Board"})
	require.NoError(t, err)

	t.Run("by ID", func(t *testing.T) {
		found, err := svc.Get(ctx, spaceID, b.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, b.ID, found.ID)
	})

	t.Run("by slug", func(t *testing.T) {
		found, err := svc.Get(ctx, spaceID, b.Slug)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, b.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := svc.Get(ctx, spaceID, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestBoardService_List(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		_, err := svc.Create(ctx, spaceID, CreateInput{Name: "List Board " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	t.Run("all", func(t *testing.T) {
		boards, pi, err := svc.List(ctx, spaceID, pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Len(t, boards, 4)
		assert.False(t, pi.HasMore)
	})

	t.Run("paginated", func(t *testing.T) {
		boards, pi, err := svc.List(ctx, spaceID, pagination.Params{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, boards, 2)
		assert.True(t, pi.HasMore)
	})
}

func TestBoardService_Update(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "Update Board", Metadata: `{"k":"v"}`})
	require.NoError(t, err)

	t.Run("update name", func(t *testing.T) {
		name := "Renamed Board"
		updated, err := svc.Update(ctx, spaceID, b.ID, UpdateInput{Name: &name})
		require.NoError(t, err)
		assert.Equal(t, "Renamed Board", updated.Name)
		assert.Equal(t, "renamed-board", updated.Slug)
	})

	t.Run("metadata deep merge", func(t *testing.T) {
		meta := `{"k2":"v2"}`
		updated, err := svc.Update(ctx, spaceID, b.ID, UpdateInput{Metadata: &meta})
		require.NoError(t, err)
		assert.Contains(t, updated.Metadata, "k")
		assert.Contains(t, updated.Metadata, "k2")
	})

	t.Run("not found", func(t *testing.T) {
		name := "test"
		u, err := svc.Update(ctx, spaceID, "nonexistent", UpdateInput{Name: &name})
		require.NoError(t, err)
		assert.Nil(t, u)
	})
}

func TestBoardService_Delete(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "Delete Board"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := svc.Delete(ctx, spaceID, b.ID)
		require.NoError(t, err)
		found, err := svc.Get(ctx, spaceID, b.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, spaceID, "nonexistent")
		assert.Error(t, err)
	})
}

func TestBoardService_SetLock(t *testing.T) {
	db, spaceID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	b, err := svc.Create(ctx, spaceID, CreateInput{Name: "Lock Board"})
	require.NoError(t, err)
	assert.False(t, b.IsLocked)

	t.Run("lock", func(t *testing.T) {
		locked, err := svc.SetLock(ctx, spaceID, b.ID, true)
		require.NoError(t, err)
		assert.True(t, locked.IsLocked)
	})

	t.Run("unlock", func(t *testing.T) {
		unlocked, err := svc.SetLock(ctx, spaceID, b.ID, false)
		require.NoError(t, err)
		assert.False(t, unlocked.IsLocked)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.SetLock(ctx, spaceID, "nonexistent", true)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}
