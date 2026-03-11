package org

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
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func setupDB(t *testing.T) *gorm.DB {
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

func TestOrgService_Create(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		o, err := svc.Create(ctx, CreateInput{Name: "Acme Corp", Description: "Test org"})
		require.NoError(t, err)
		assert.NotEmpty(t, o.ID)
		assert.Equal(t, "Acme Corp", o.Name)
		assert.Equal(t, "acme-corp", o.Slug)
		assert.Equal(t, "{}", o.Metadata)
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateInput{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("invalid metadata", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateInput{Name: "Bad", Metadata: "not json"})
		assert.Error(t, err)
	})

	t.Run("duplicate slug gets suffix", func(t *testing.T) {
		o1, err := svc.Create(ctx, CreateInput{Name: "Dup Org"})
		require.NoError(t, err)
		o2, err := svc.Create(ctx, CreateInput{Name: "Dup Org"})
		require.NoError(t, err)
		assert.NotEqual(t, o1.Slug, o2.Slug)
		assert.Equal(t, "dup-org-2", o2.Slug)
	})

	t.Run("with metadata", func(t *testing.T) {
		o, err := svc.Create(ctx, CreateInput{Name: "Meta Org", Metadata: `{"tier":"pro"}`})
		require.NoError(t, err)
		assert.Contains(t, o.Metadata, "tier")
	})
}

func TestOrgService_Get(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	o, err := svc.Create(ctx, CreateInput{Name: "Lookup Org"})
	require.NoError(t, err)

	t.Run("by ID", func(t *testing.T) {
		found, err := svc.Get(ctx, o.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, o.ID, found.ID)
	})

	t.Run("by slug", func(t *testing.T) {
		found, err := svc.Get(ctx, o.Slug)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, o.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := svc.Get(ctx, "nonexistent-slug")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestOrgService_List(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, CreateInput{Name: "List Org " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	t.Run("default pagination", func(t *testing.T) {
		orgs, pageInfo, err := svc.List(ctx, pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Len(t, orgs, 5)
		assert.False(t, pageInfo.HasMore)
	})

	t.Run("limited", func(t *testing.T) {
		orgs, pageInfo, err := svc.List(ctx, pagination.Params{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, orgs, 2)
		assert.True(t, pageInfo.HasMore)
		assert.NotEmpty(t, pageInfo.NextCursor)
	})

	t.Run("cursor continuation", func(t *testing.T) {
		orgs1, pi1, err := svc.List(ctx, pagination.Params{Limit: 3})
		require.NoError(t, err)
		assert.Len(t, orgs1, 3)

		orgs2, pi2, err := svc.List(ctx, pagination.Params{Limit: 3, Cursor: pi1.NextCursor})
		require.NoError(t, err)
		assert.Len(t, orgs2, 2)
		assert.False(t, pi2.HasMore)
		assert.NotEqual(t, orgs1[0].ID, orgs2[0].ID)
	})
}

func TestOrgService_Update(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	o, err := svc.Create(ctx, CreateInput{Name: "Update Org", Metadata: `{"key":"value"}`})
	require.NoError(t, err)

	t.Run("update name", func(t *testing.T) {
		newName := "Updated Org"
		updated, err := svc.Update(ctx, o.ID, UpdateInput{Name: &newName})
		require.NoError(t, err)
		assert.Equal(t, "Updated Org", updated.Name)
		assert.Equal(t, "updated-org", updated.Slug)
	})

	t.Run("metadata deep merge", func(t *testing.T) {
		meta := `{"key2":"value2"}`
		updated, err := svc.Update(ctx, o.ID, UpdateInput{Metadata: &meta})
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Contains(t, updated.Metadata, "key")
		assert.Contains(t, updated.Metadata, "key2")
	})

	t.Run("not found", func(t *testing.T) {
		name := "test"
		updated, err := svc.Update(ctx, "nonexistent", UpdateInput{Name: &name})
		require.NoError(t, err)
		assert.Nil(t, updated)
	})
}

func TestOrgService_Delete(t *testing.T) {
	db := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	o, err := svc.Create(ctx, CreateInput{Name: "Delete Org"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := svc.Delete(ctx, o.ID)
		require.NoError(t, err)

		found, err := svc.Get(ctx, o.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, "nonexistent")
		assert.Error(t, err)
	})
}
