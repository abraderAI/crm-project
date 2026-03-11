package space

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

	// Create parent org.
	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return db, org.ID
}

func TestSpaceService_Create(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		sp, err := svc.Create(ctx, orgID, CreateInput{Name: "General", Type: models.SpaceTypeGeneral})
		require.NoError(t, err)
		assert.NotEmpty(t, sp.ID)
		assert.Equal(t, "General", sp.Name)
		assert.Equal(t, "general", sp.Slug)
		assert.Equal(t, models.SpaceTypeGeneral, sp.Type)
		assert.Equal(t, orgID, sp.OrgID)
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, orgID, CreateInput{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := svc.Create(ctx, orgID, CreateInput{Name: "Bad", Type: "invalid"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid space type")
	})

	t.Run("default type general", func(t *testing.T) {
		sp, err := svc.Create(ctx, orgID, CreateInput{Name: "NoType"})
		require.NoError(t, err)
		assert.Equal(t, models.SpaceTypeGeneral, sp.Type)
	})

	t.Run("invalid metadata", func(t *testing.T) {
		_, err := svc.Create(ctx, orgID, CreateInput{Name: "Bad Meta", Metadata: "not json"})
		assert.Error(t, err)
	})

	t.Run("duplicate slug suffix", func(t *testing.T) {
		s1, err := svc.Create(ctx, orgID, CreateInput{Name: "Dup Space"})
		require.NoError(t, err)
		s2, err := svc.Create(ctx, orgID, CreateInput{Name: "Dup Space"})
		require.NoError(t, err)
		assert.NotEqual(t, s1.Slug, s2.Slug)
		assert.Equal(t, "dup-space-2", s2.Slug)
	})

	t.Run("valid types", func(t *testing.T) {
		for _, st := range models.ValidSpaceTypes() {
			sp, err := svc.Create(ctx, orgID, CreateInput{Name: "Space " + string(st), Type: st})
			require.NoError(t, err, "type %s should be valid", st)
			assert.Equal(t, st, sp.Type)
		}
	})
}

func TestSpaceService_Get(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	sp, err := svc.Create(ctx, orgID, CreateInput{Name: "Lookup Space"})
	require.NoError(t, err)

	t.Run("by ID", func(t *testing.T) {
		found, err := svc.Get(ctx, orgID, sp.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, sp.ID, found.ID)
	})

	t.Run("by slug", func(t *testing.T) {
		found, err := svc.Get(ctx, orgID, sp.Slug)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, sp.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := svc.Get(ctx, orgID, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestSpaceService_List(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, orgID, CreateInput{Name: "List Space " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	t.Run("all", func(t *testing.T) {
		spaces, pi, err := svc.List(ctx, orgID, pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Len(t, spaces, 5)
		assert.False(t, pi.HasMore)
	})

	t.Run("paginated", func(t *testing.T) {
		spaces, pi, err := svc.List(ctx, orgID, pagination.Params{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, spaces, 2)
		assert.True(t, pi.HasMore)
		assert.NotEmpty(t, pi.NextCursor)
	})

	t.Run("cursor", func(t *testing.T) {
		s1, pi1, err := svc.List(ctx, orgID, pagination.Params{Limit: 3})
		require.NoError(t, err)
		s2, _, err := svc.List(ctx, orgID, pagination.Params{Limit: 3, Cursor: pi1.NextCursor})
		require.NoError(t, err)
		assert.Len(t, s1, 3)
		assert.Len(t, s2, 2)
		assert.NotEqual(t, s1[0].ID, s2[0].ID)
	})
}

func TestSpaceService_Update(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	sp, err := svc.Create(ctx, orgID, CreateInput{Name: "Update Space", Metadata: `{"k":"v"}`})
	require.NoError(t, err)

	t.Run("update name", func(t *testing.T) {
		name := "Renamed"
		updated, err := svc.Update(ctx, orgID, sp.ID, UpdateInput{Name: &name})
		require.NoError(t, err)
		assert.Equal(t, "Renamed", updated.Name)
		assert.Equal(t, "renamed", updated.Slug)
	})

	t.Run("update type", func(t *testing.T) {
		newType := models.SpaceTypeCRM
		updated, err := svc.Update(ctx, orgID, sp.ID, UpdateInput{Type: &newType})
		require.NoError(t, err)
		assert.Equal(t, models.SpaceTypeCRM, updated.Type)
	})

	t.Run("invalid type update", func(t *testing.T) {
		bad := models.SpaceType("bad")
		_, err := svc.Update(ctx, orgID, sp.ID, UpdateInput{Type: &bad})
		assert.Error(t, err)
	})

	t.Run("metadata deep merge", func(t *testing.T) {
		meta := `{"k2":"v2"}`
		updated, err := svc.Update(ctx, orgID, sp.ID, UpdateInput{Metadata: &meta})
		require.NoError(t, err)
		assert.Contains(t, updated.Metadata, "k")
		assert.Contains(t, updated.Metadata, "k2")
	})

	t.Run("not found", func(t *testing.T) {
		name := "test"
		u, err := svc.Update(ctx, orgID, "nonexistent", UpdateInput{Name: &name})
		require.NoError(t, err)
		assert.Nil(t, u)
	})
}

func TestSpaceService_Delete(t *testing.T) {
	db, orgID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	sp, err := svc.Create(ctx, orgID, CreateInput{Name: "Delete Space"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := svc.Delete(ctx, orgID, sp.ID)
		require.NoError(t, err)
		found, err := svc.Get(ctx, orgID, sp.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, orgID, "nonexistent")
		assert.Error(t, err)
	})
}
