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

// --- Repository Tests ---

func TestRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	svc := NewService(repo)
	created, err := svc.Create(ctx, CreateInput{Name: "Test Org"})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Test Org", created.Name)
	assert.Equal(t, "test-org", created.Slug)

	// Get by ID.
	got, err := repo.GetByIDOrSlug(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	// Get by slug.
	got, err = repo.GetByIDOrSlug(ctx, "test-org")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestRepository_GetByIDOrSlug_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	_, err := repo.GetByIDOrSlug(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestRepository_List(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, CreateInput{Name: "Org " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	orgs, err := repo.List(ctx, "", 10, "")
	require.NoError(t, err)
	assert.Len(t, orgs, 5)
}

func TestRepository_ListWithCursor(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 5; i++ {
		o, err := svc.Create(ctx, CreateInput{Name: "Org " + string(rune('A'+i))})
		require.NoError(t, err)
		ids = append(ids, o.ID)
	}

	// Use second org's ID as cursor.
	orgs, err := repo.List(ctx, ids[1], 10, "")
	require.NoError(t, err)
	assert.Len(t, orgs, 3) // Should get orgs after second.
}

func TestRepository_Delete(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "To Delete"})
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Should not find soft-deleted org.
	_, err = repo.GetByIDOrSlug(ctx, created.ID)
	assert.Error(t, err)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	err := repo.Delete(context.Background(), "nonexistent-id")
	assert.Error(t, err)
}

func TestRepository_SlugExists(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateInput{Name: "Unique Org"})
	require.NoError(t, err)

	exists, err := repo.SlugExists(ctx, "unique-org")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.SlugExists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// --- Service Tests ---

func TestService_Create_Valid(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	org, err := svc.Create(context.Background(), CreateInput{
		Name:     "My Org",
		Metadata: `{"billing_tier":"pro"}`,
	})
	require.NoError(t, err)
	assert.Equal(t, "My Org", org.Name)
	assert.Equal(t, "my-org", org.Slug)
	assert.JSONEq(t, `{"billing_tier":"pro"}`, org.Metadata)
}

func TestService_Create_EmptyName(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	_, err := svc.Create(context.Background(), CreateInput{Name: ""})
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_InvalidMetadata(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	_, err := svc.Create(context.Background(), CreateInput{Name: "Org", Metadata: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidMeta)
}

func TestService_Create_DefaultMetadata(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	org, err := svc.Create(context.Background(), CreateInput{Name: "Default Meta Org"})
	require.NoError(t, err)
	assert.Equal(t, "{}", org.Metadata)
}

func TestService_Create_SlugDedup(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	org1, err := svc.Create(ctx, CreateInput{Name: "Dupe Name"})
	require.NoError(t, err)
	assert.Equal(t, "dupe-name", org1.Slug)

	org2, err := svc.Create(ctx, CreateInput{Name: "Dupe Name"})
	require.NoError(t, err)
	assert.Equal(t, "dupe-name-1", org2.Slug)

	org3, err := svc.Create(ctx, CreateInput{Name: "Dupe Name"})
	require.NoError(t, err)
	assert.Equal(t, "dupe-name-2", org3.Slug)
}

func TestService_GetByRef(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Get Org"})
	require.NoError(t, err)

	// By ID.
	got, err := svc.GetByRef(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	// By slug.
	got, err = svc.GetByRef(ctx, "get-org")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestService_GetByRef_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	_, err := svc.GetByRef(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_List(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, CreateInput{Name: "List Org " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	orgs, hasMore, err := svc.List(ctx, "", 2, "")
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
	assert.True(t, hasMore)

	orgs, hasMore, err = svc.List(ctx, "", 10, "")
	require.NoError(t, err)
	assert.Len(t, orgs, 3)
	assert.False(t, hasMore)
}

func TestService_Update_Name(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Original"})
	require.NoError(t, err)

	newName := "Updated"
	updated, err := svc.Update(ctx, created.ID, UpdateInput{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "updated", updated.Slug)
}

func TestService_Update_EmptyName(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Test"})
	require.NoError(t, err)

	empty := ""
	_, err = svc.Update(ctx, created.ID, UpdateInput{Name: &empty})
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_MetadataDeepMerge(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Meta Org", Metadata: `{"a":"1","b":"2"}`})
	require.NoError(t, err)

	patch := `{"b":"3","c":"4"}`
	updated, err := svc.Update(ctx, created.ID, UpdateInput{Metadata: &patch})
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"1","b":"3","c":"4"}`, updated.Metadata)
}

func TestService_Update_Description(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Desc Org"})
	require.NoError(t, err)

	desc := "new description"
	updated, err := svc.Update(ctx, created.ID, UpdateInput{Description: &desc})
	require.NoError(t, err)
	assert.Equal(t, "new description", updated.Description)
}

func TestService_Update_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	name := "Test"
	_, err := svc.Update(context.Background(), "nonexistent", UpdateInput{Name: &name})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Delete Me"})
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByRef(ctx, created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	err := svc.Delete(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete_BySlug(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateInput{Name: "Slug Delete"})
	require.NoError(t, err)

	err = svc.Delete(ctx, "slug-delete")
	require.NoError(t, err)

	_, err = svc.GetByRef(ctx, "slug-delete")
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- isUUID ---

func TestIsUUID(t *testing.T) {
	assert.True(t, isUUID("01906a2b-5e4c-7c00-8000-000000000000"))
	assert.False(t, isUUID("my-slug"))
	assert.False(t, isUUID(""))
}

// --- Fuzz Tests ---

func FuzzServiceCreate(f *testing.F) {
	f.Add("Valid Org", "{}")
	f.Add("", "{}")
	f.Add("Test", "invalid-json")
	f.Add("Org!", `{"key":"val"}`)
	f.Add("A B C", "")
	f.Fuzz(func(t *testing.T, name, meta string) {
		db := testDB(t)
		svc := NewService(NewRepository(db))
		_, _ = svc.Create(context.Background(), CreateInput{Name: name, Metadata: meta})
	})
}
