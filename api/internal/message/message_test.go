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
	th := &models.Thread{BoardID: bd.ID, Title: "Test Thread", Slug: "test-thread", AuthorID: "user1", Metadata: "{}"}
	require.NoError(t, db.Create(th).Error)
	return db, th.ID
}

func TestMessageService_Create(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Hello", Type: models.MessageTypeComment})
		require.NoError(t, err)
		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, "Hello", msg.Body)
		assert.Equal(t, "user1", msg.AuthorID)
		assert.Equal(t, models.MessageTypeComment, msg.Type)
	})

	t.Run("empty body", func(t *testing.T) {
		_, err := svc.Create(ctx, threadID, "user1", false, CreateInput{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "body is required")
	})

	t.Run("thread locked", func(t *testing.T) {
		_, err := svc.Create(ctx, threadID, "user1", true, CreateInput{Body: "Locked"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "thread is locked")
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Bad", Type: "invalid"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid message type")
	})

	t.Run("default type comment", func(t *testing.T) {
		msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "No type"})
		require.NoError(t, err)
		assert.Equal(t, models.MessageTypeComment, msg.Type)
	})

	t.Run("valid types", func(t *testing.T) {
		for _, mt := range models.ValidMessageTypes() {
			msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Type " + string(mt), Type: mt})
			require.NoError(t, err, "type %s should be valid", mt)
			assert.Equal(t, mt, msg.Type)
		}
	})
}

func TestMessageService_Get(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Lookup"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		found, err := svc.Get(ctx, threadID, msg.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, msg.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := svc.Get(ctx, threadID, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestMessageService_List(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Message " + string(rune('A'+i))})
		require.NoError(t, err)
	}

	t.Run("all", func(t *testing.T) {
		msgs, pi, err := svc.List(ctx, threadID, pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Len(t, msgs, 5)
		assert.False(t, pi.HasMore)
	})

	t.Run("paginated", func(t *testing.T) {
		msgs, pi, err := svc.List(ctx, threadID, pagination.Params{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
		assert.True(t, pi.HasMore)
	})
}

func TestMessageService_Update(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Original"})
	require.NoError(t, err)

	t.Run("author can update", func(t *testing.T) {
		body := "Updated"
		updated, err := svc.Update(ctx, threadID, msg.ID, "user1", UpdateInput{Body: &body})
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Body)
	})

	t.Run("non-author cannot update", func(t *testing.T) {
		body := "Hacked"
		_, err := svc.Update(ctx, threadID, msg.ID, "user2", UpdateInput{Body: &body})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only the author can update this message")
	})

	t.Run("creates revision", func(t *testing.T) {
		body := "Revision"
		_, err := svc.Update(ctx, threadID, msg.ID, "user1", UpdateInput{Body: &body})
		require.NoError(t, err)

		var revisions []models.Revision
		err = db.Where("entity_type = ? AND entity_id = ?", "message", msg.ID).Find(&revisions).Error
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(revisions), 1)
	})

	t.Run("not found", func(t *testing.T) {
		body := "test"
		u, err := svc.Update(ctx, threadID, "nonexistent", "user1", UpdateInput{Body: &body})
		require.NoError(t, err)
		assert.Nil(t, u)
	})
}

func TestMessageService_Delete(t *testing.T) {
	db, threadID := setupDB(t)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	msg, err := svc.Create(ctx, threadID, "user1", false, CreateInput{Body: "Delete me"})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := svc.Delete(ctx, threadID, msg.ID)
		require.NoError(t, err)
		found, err := svc.Get(ctx, threadID, msg.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, threadID, "nonexistent")
		assert.Error(t, err)
	})
}
