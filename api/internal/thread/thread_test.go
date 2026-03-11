package thread

import (
	"context"
	"net/http"
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
	bd := &models.Board{SpaceID: sp.ID, Name: "Test Board", Slug: "test-board", Metadata: "{}"}
	require.NoError(t, db.Create(bd).Error)
	return db, bd.ID
}

func TestThreadService_Create(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Test Thread", Body: "Content"})
		require.NoError(t, err)
		assert.NotEmpty(t, th.ID)
		assert.Equal(t, "Test Thread", th.Title)
		assert.Equal(t, "test-thread", th.Slug)
		assert.Equal(t, "user1", th.AuthorID)
		assert.False(t, th.IsPinned)
		assert.False(t, th.IsLocked)
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := svc.Create(ctx, boardID, "user1", false, CreateInput{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("board locked", func(t *testing.T) {
		_, err := svc.Create(ctx, boardID, "user1", true, CreateInput{Title: "Locked"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "board is locked")
	})

	t.Run("invalid metadata", func(t *testing.T) {
		_, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Bad", Metadata: "not json"})
		assert.Error(t, err)
	})

	t.Run("duplicate slug suffix", func(t *testing.T) {
		t1, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Dup Thread"})
		require.NoError(t, err)
		t2, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Dup Thread"})
		require.NoError(t, err)
		assert.NotEqual(t, t1.Slug, t2.Slug)
		assert.Equal(t, "dup-thread-2", t2.Slug)
	})

	t.Run("with metadata", func(t *testing.T) {
		th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Meta Thread", Metadata: `{"status":"open"}`})
		require.NoError(t, err)
		assert.Contains(t, th.Metadata, "status")
	})
}

func TestThreadService_Get(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Lookup Thread"})
	require.NoError(t, err)

	t.Run("by ID", func(t *testing.T) {
		found, err := svc.Get(ctx, boardID, th.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, th.ID, found.ID)
	})

	t.Run("by slug", func(t *testing.T) {
		found, err := svc.Get(ctx, boardID, th.Slug)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, th.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := svc.Get(ctx, boardID, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestThreadService_List(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "List Thread " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	t.Run("all", func(t *testing.T) {
		threads, pi, err := svc.List(ctx, boardID, ListParams{Params: pagination.Params{Limit: 50}})
		require.NoError(t, err)
		assert.Len(t, threads, 5)
		assert.False(t, pi.HasMore)
	})

	t.Run("paginated", func(t *testing.T) {
		threads, pi, err := svc.List(ctx, boardID, ListParams{Params: pagination.Params{Limit: 2}})
		require.NoError(t, err)
		assert.Len(t, threads, 2)
		assert.True(t, pi.HasMore)
	})
}

func TestThreadService_ListMetadataFilter(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	// Create threads with different metadata.
	_, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Open Bug", Metadata: `{"status":"open","priority":5}`})
	require.NoError(t, err)
	_, err = svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Closed Bug", Metadata: `{"status":"closed","priority":3}`})
	require.NoError(t, err)
	_, err = svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Open Feature", Metadata: `{"status":"open","priority":1}`})
	require.NoError(t, err)

	t.Run("eq filter", func(t *testing.T) {
		threads, _, err := svc.List(ctx, boardID, ListParams{
			Params:  pagination.Params{Limit: 50},
			Filters: []MetadataFilter{{Path: "$.status", Operator: "eq", Value: "open"}},
		})
		require.NoError(t, err)
		assert.Len(t, threads, 2)
	})

	t.Run("gt filter", func(t *testing.T) {
		threads, _, err := svc.List(ctx, boardID, ListParams{
			Params:  pagination.Params{Limit: 50},
			Filters: []MetadataFilter{{Path: "$.priority", Operator: "gt", Value: "2"}},
		})
		require.NoError(t, err)
		assert.Len(t, threads, 2) // priority 5 and 3
	})

	t.Run("lt filter", func(t *testing.T) {
		threads, _, err := svc.List(ctx, boardID, ListParams{
			Params:  pagination.Params{Limit: 50},
			Filters: []MetadataFilter{{Path: "$.priority", Operator: "lt", Value: "4"}},
		})
		require.NoError(t, err)
		assert.Len(t, threads, 2) // priority 3 and 1
	})

	t.Run("combined filters", func(t *testing.T) {
		threads, _, err := svc.List(ctx, boardID, ListParams{
			Params: pagination.Params{Limit: 50},
			Filters: []MetadataFilter{
				{Path: "$.status", Operator: "eq", Value: "open"},
				{Path: "$.priority", Operator: "gte", Value: "5"},
			},
		})
		require.NoError(t, err)
		assert.Len(t, threads, 1) // only the "Open Bug" with priority 5
	})
}

func TestThreadService_Update(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Update Thread", Body: "Original", Metadata: `{"k":"v"}`})
	require.NoError(t, err)

	t.Run("update title", func(t *testing.T) {
		title := "New Title"
		updated, err := svc.Update(ctx, boardID, th.ID, "user1", UpdateInput{Title: &title})
		require.NoError(t, err)
		assert.Equal(t, "New Title", updated.Title)
		assert.Equal(t, "new-title", updated.Slug)
	})

	t.Run("update body", func(t *testing.T) {
		body := "Updated body"
		updated, err := svc.Update(ctx, boardID, th.ID, "user1", UpdateInput{Body: &body})
		require.NoError(t, err)
		assert.Equal(t, "Updated body", updated.Body)
	})

	t.Run("metadata deep merge", func(t *testing.T) {
		meta := `{"k2":"v2"}`
		updated, err := svc.Update(ctx, boardID, th.ID, "user1", UpdateInput{Metadata: &meta})
		require.NoError(t, err)
		assert.Contains(t, updated.Metadata, "k")
		assert.Contains(t, updated.Metadata, "k2")
	})

	t.Run("creates revision", func(t *testing.T) {
		body := "Revision test"
		_, err := svc.Update(ctx, boardID, th.ID, "user1", UpdateInput{Body: &body})
		require.NoError(t, err)

		// Verify revision exists in DB.
		var revisions []models.Revision
		err = db.Where("entity_type = ? AND entity_id = ?", "thread", th.ID).Find(&revisions).Error
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(revisions), 1)
	})

	t.Run("not found", func(t *testing.T) {
		title := "test"
		u, err := svc.Update(ctx, boardID, "nonexistent", "user1", UpdateInput{Title: &title})
		require.NoError(t, err)
		assert.Nil(t, u)
	})
}

func TestThreadService_Delete(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Delete Thread"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := svc.Delete(ctx, boardID, th.ID)
		require.NoError(t, err)
		found, err := svc.Get(ctx, boardID, th.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, boardID, "nonexistent")
		assert.Error(t, err)
	})
}

func TestThreadService_SetPin(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Pin Thread"})
	require.NoError(t, err)
	assert.False(t, th.IsPinned)

	t.Run("pin", func(t *testing.T) {
		pinned, err := svc.SetPin(ctx, boardID, th.ID, true)
		require.NoError(t, err)
		assert.True(t, pinned.IsPinned)
	})

	t.Run("unpin", func(t *testing.T) {
		unpinned, err := svc.SetPin(ctx, boardID, th.ID, false)
		require.NoError(t, err)
		assert.False(t, unpinned.IsPinned)
	})

	t.Run("not found", func(t *testing.T) {
		r, err := svc.SetPin(ctx, boardID, "nonexistent", true)
		require.NoError(t, err)
		assert.Nil(t, r)
	})
}

func TestThreadService_SetLock(t *testing.T) {
	db, boardID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	th, err := svc.Create(ctx, boardID, "user1", false, CreateInput{Title: "Lock Thread"})
	require.NoError(t, err)

	t.Run("lock", func(t *testing.T) {
		locked, err := svc.SetLock(ctx, boardID, th.ID, true)
		require.NoError(t, err)
		assert.True(t, locked.IsLocked)
	})

	t.Run("unlock", func(t *testing.T) {
		unlocked, err := svc.SetLock(ctx, boardID, th.ID, false)
		require.NoError(t, err)
		assert.False(t, unlocked.IsLocked)
	})

	t.Run("not found", func(t *testing.T) {
		r, err := svc.SetLock(ctx, boardID, "nonexistent", true)
		require.NoError(t, err)
		assert.Nil(t, r)
	})
}

func TestParseMetadataFilters(t *testing.T) {
	t.Run("simple equality", func(t *testing.T) {
		filters := parseMetadataFilters(mustRequest(t, "?metadata[status]=open"))
		require.Len(t, filters, 1)
		assert.Equal(t, "$.status", filters[0].Path)
		assert.Equal(t, "eq", filters[0].Operator)
		assert.Equal(t, "open", filters[0].Value)
	})

	t.Run("comparison operator", func(t *testing.T) {
		filters := parseMetadataFilters(mustRequest(t, "?metadata[priority][gt]=3"))
		require.Len(t, filters, 1)
		assert.Equal(t, "$.priority", filters[0].Path)
		assert.Equal(t, "gt", filters[0].Operator)
		assert.Equal(t, "3", filters[0].Value)
	})

	t.Run("no metadata params", func(t *testing.T) {
		filters := parseMetadataFilters(mustRequest(t, "?limit=10"))
		assert.Empty(t, filters)
	})

	t.Run("multiple filters", func(t *testing.T) {
		filters := parseMetadataFilters(mustRequest(t, "?metadata[status]=open&metadata[priority][gte]=3"))
		assert.Len(t, filters, 2)
	})
}

func mustRequest(t *testing.T, query string) *http.Request {
	t.Helper()
	req, err := http.NewRequest("GET", "http://test"+query, nil)
	require.NoError(t, err)
	return req
}
