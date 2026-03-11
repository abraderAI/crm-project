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

func createOrg(t *testing.T, db *gorm.DB, name string) *models.Org {
	t.Helper()
	o := &models.Org{Name: name, Slug: name, Metadata: "{}"}
	require.NoError(t, db.Create(o).Error)
	return o
}

// --- Service Tests ---

func TestService_Create_Valid(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "test-org")
	svc := NewService(NewRepository(db))

	sp, err := svc.Create(context.Background(), org.ID, CreateInput{Name: "My Space"})
	require.NoError(t, err)
	assert.Equal(t, "My Space", sp.Name)
	assert.Equal(t, "my-space", sp.Slug)
	assert.Equal(t, org.ID, sp.OrgID)
	assert.Equal(t, models.SpaceTypeGeneral, sp.Type)
}

func TestService_Create_WithType(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "type-org")
	svc := NewService(NewRepository(db))

	sp, err := svc.Create(context.Background(), org.ID, CreateInput{Name: "CRM Space", Type: "crm"})
	require.NoError(t, err)
	assert.Equal(t, models.SpaceTypeCRM, sp.Type)
}

func TestService_Create_InvalidType(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "bad-type-org")
	svc := NewService(NewRepository(db))

	_, err := svc.Create(context.Background(), org.ID, CreateInput{Name: "Bad", Type: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidType)
}

func TestService_Create_EmptyName(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "empty-org")
	svc := NewService(NewRepository(db))

	_, err := svc.Create(context.Background(), org.ID, CreateInput{Name: ""})
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_InvalidMetadata(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "meta-org")
	svc := NewService(NewRepository(db))

	_, err := svc.Create(context.Background(), org.ID, CreateInput{Name: "Space", Metadata: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidMeta)
}

func TestService_Create_SlugDedup(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "dedup-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	s1, err := svc.Create(ctx, org.ID, CreateInput{Name: "Dupe"})
	require.NoError(t, err)
	assert.Equal(t, "dupe", s1.Slug)

	s2, err := svc.Create(ctx, org.ID, CreateInput{Name: "Dupe"})
	require.NoError(t, err)
	assert.Equal(t, "dupe-1", s2.Slug)
}

func TestService_GetByRef(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "get-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInput{Name: "Find Me"})
	require.NoError(t, err)

	// By ID.
	got, err := svc.GetByRef(ctx, org.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	// By slug.
	got, err = svc.GetByRef(ctx, org.ID, "find-me")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestService_GetByRef_NotFound(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "nf-org")
	svc := NewService(NewRepository(db))

	_, err := svc.GetByRef(context.Background(), org.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_List(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "list-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, org.ID, CreateInput{Name: "Space " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	spaces, hasMore, err := svc.List(ctx, org.ID, "", 2)
	require.NoError(t, err)
	assert.Len(t, spaces, 2)
	assert.True(t, hasMore)

	spaces, hasMore, err = svc.List(ctx, org.ID, "", 10)
	require.NoError(t, err)
	assert.Len(t, spaces, 3)
	assert.False(t, hasMore)
}

func TestService_Update(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "upd-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInput{Name: "Original", Metadata: `{"a":"1"}`})
	require.NoError(t, err)

	newName := "Updated"
	newType := "crm"
	meta := `{"b":"2"}`
	updated, err := svc.Update(ctx, org.ID, created.ID, UpdateInput{
		Name:     &newName,
		Type:     &newType,
		Metadata: &meta,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, models.SpaceTypeCRM, updated.Type)
	assert.JSONEq(t, `{"a":"1","b":"2"}`, updated.Metadata)
}

func TestService_Update_EmptyName(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "upd-empty-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInput{Name: "Test"})
	require.NoError(t, err)

	empty := ""
	_, err = svc.Update(ctx, org.ID, created.ID, UpdateInput{Name: &empty})
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_InvalidType(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "upd-bad-type")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInput{Name: "Test"})
	require.NoError(t, err)

	bad := "invalid"
	_, err = svc.Update(ctx, org.ID, created.ID, UpdateInput{Type: &bad})
	assert.ErrorIs(t, err, ErrInvalidType)
}

func TestService_Update_NotFound(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "nf-upd-org")
	svc := NewService(NewRepository(db))

	name := "X"
	_, err := svc.Update(context.Background(), org.ID, "nonexistent", UpdateInput{Name: &name})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "del-org")
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, org.ID, CreateInput{Name: "To Delete"})
	require.NoError(t, err)

	err = svc.Delete(ctx, org.ID, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByRef(ctx, org.ID, created.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db, "nf-del-org")
	svc := NewService(NewRepository(db))

	err := svc.Delete(context.Background(), org.ID, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Helpers ---

func TestIsUUID(t *testing.T) {
	assert.True(t, isUUID("01906a2b-5e4c-7c00-8000-000000000000"))
	assert.False(t, isUUID("slug"))
}

// --- Fuzz Tests ---

func FuzzServiceCreate(f *testing.F) {
	f.Add("Valid Space", "{}", "general")
	f.Add("", "{}", "")
	f.Add("Test", "invalid", "crm")
	f.Add("A B", "", "support")
	f.Add("Space!", `{"k":"v"}`, "invalid_type")
	f.Fuzz(func(t *testing.T, name, meta, spaceType string) {
		db := testDB(t)
		org := createOrg(t, db, "fuzz-org")
		svc := NewService(NewRepository(db))
		_, _ = svc.Create(context.Background(), org.ID, CreateInput{Name: name, Metadata: meta, Type: spaceType})
	})
}
