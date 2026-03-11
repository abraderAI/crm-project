package revision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func setupTestDB(t *testing.T) *gorm.DB {
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
	return db
}

func TestRepository_ListAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// Create revisions.
	for i := 1; i <= 3; i++ {
		rev := &models.Revision{
			EntityType:      "thread",
			EntityID:        "t-123",
			Version:         i,
			PreviousContent: `{"title":"old"}`,
			EditorID:        "user1",
		}
		require.NoError(t, db.Create(rev).Error)
	}

	// List.
	revs, pageInfo, err := repo.List(context.Background(), "thread", "t-123", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, revs, 3)
	assert.False(t, pageInfo.HasMore)
	// Should be ordered by ID DESC.
	assert.Greater(t, revs[0].Version, revs[2].Version)

	// Get single.
	rev, err := repo.Get(context.Background(), revs[0].ID)
	require.NoError(t, err)
	assert.Equal(t, revs[0].ID, rev.ID)

	// Get not found.
	rev, err = repo.Get(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, rev)
}

func TestRepository_List_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	revs, pageInfo, err := repo.List(context.Background(), "thread", "none", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Empty(t, revs)
	assert.False(t, pageInfo.HasMore)
}

func TestRepository_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	for i := 0; i < 5; i++ {
		rev := &models.Revision{
			EntityType: "message", EntityID: "m-1", Version: i + 1,
			PreviousContent: `{}`, EditorID: "u1",
		}
		require.NoError(t, db.Create(rev).Error)
	}

	revs, pageInfo, err := repo.List(context.Background(), "message", "m-1", pagination.Params{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, revs, 2)
	assert.True(t, pageInfo.HasMore)
}

func TestHandler_List_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	r := chi.NewRouter()
	r.Get("/v1/revisions/{entityType}/{entityID}", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/revisions/invalid/id123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List_Valid(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	rev := &models.Revision{
		EntityType: "thread", EntityID: "t-1", Version: 1,
		PreviousContent: `{}`, EditorID: "u1",
	}
	require.NoError(t, db.Create(rev).Error)

	r := chi.NewRouter()
	r.Get("/v1/revisions/{entityType}/{entityID}", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/revisions/thread/t-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	r := chi.NewRouter()
	r.Get("/v1/revisions/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/v1/revisions/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Get_Valid(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	rev := &models.Revision{
		EntityType: "thread", EntityID: "t-1", Version: 1,
		PreviousContent: `{}`, EditorID: "u1",
	}
	require.NoError(t, db.Create(rev).Error)

	r := chi.NewRouter()
	r.Get("/v1/revisions/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/v1/revisions/"+rev.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
