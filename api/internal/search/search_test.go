package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple word", "hello", `"hello"`},
		{"two words", "hello world", `"hello" "world"`},
		{"special chars", `hello" AND "world`, `"hello" "world"`},
		{"operators removed", "hello AND OR NOT world", `"hello" "world"`},
		{"empty", "", ""},
		{"only special", `"'*()`, ""},
		{"with dash", "dark-mode", `"dark" "mode"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFTSQuery(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterTargets(t *testing.T) {
	// No filter returns all.
	all := filterTargets(nil)
	assert.Len(t, all, 5)

	// Filter specific types.
	filtered := filterTargets([]string{"org", "thread"})
	assert.Len(t, filtered, 2)

	// Unknown type returns empty.
	empty := filterTargets([]string{"unknown"})
	assert.Empty(t, empty)
}

func TestSortByRank(t *testing.T) {
	results := []SearchResult{
		{EntityID: "c", Rank: -5.0},
		{EntityID: "a", Rank: -10.0},
		{EntityID: "b", Rank: -7.0},
	}
	sortByRank(results)
	assert.Equal(t, "a", results[0].EntityID)
	assert.Equal(t, "b", results[1].EntityID)
	assert.Equal(t, "c", results[2].EntityID)
}

func TestSearch_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	results, pageInfo, err := repo.Search(context.Background(), "", nil, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Nil(t, results)
	assert.NotNil(t, pageInfo)
}

func TestSearch_WithResults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// Insert test data with full hierarchy to satisfy foreign keys.
	org := &models.Org{Name: "Searchable Organization", Slug: "searchable-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	space := &models.Space{OrgID: org.ID, Name: "Search Space", Slug: "search-space"}
	require.NoError(t, db.Create(space).Error)

	board := &models.Board{SpaceID: space.ID, Name: "Search Board", Slug: "search-board"}
	require.NoError(t, db.Create(board).Error)

	thread := &models.Thread{
		BoardID: board.ID, Title: "Searchable Thread", Body: "This is searchable content",
		Slug: "searchable-thread", Metadata: "{}", AuthorID: "user1",
	}
	require.NoError(t, db.Create(thread).Error)

	results, pageInfo, err := repo.Search(context.Background(), "searchable", nil, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.NotNil(t, pageInfo)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestSearch_TypeFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	org := &models.Org{Name: "Filtered Org", Slug: "filtered-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	space := &models.Space{OrgID: org.ID, Name: "Filter Space", Slug: "filter-space"}
	require.NoError(t, db.Create(space).Error)

	board := &models.Board{SpaceID: space.ID, Name: "Filter Board", Slug: "filter-board"}
	require.NoError(t, db.Create(board).Error)

	thread := &models.Thread{
		BoardID: board.ID, Title: "Filtered Thread", Slug: "filtered-thread",
		Metadata: "{}", AuthorID: "u1",
	}
	require.NoError(t, db.Create(thread).Error)

	// Search only orgs.
	results, _, err := repo.Search(context.Background(), "filtered", []string{"org"}, pagination.Params{Limit: 50})
	require.NoError(t, err)
	for _, r := range results {
		assert.Equal(t, "org", r.EntityType)
	}
}

func TestHandler_Search_MissingQuery(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	w := httptest.NewRecorder()
	handler.Search(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Search_Valid(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	org := &models.Org{Name: "Handler Test Org", Slug: "handler-test", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=handler", nil)
	w := httptest.NewRecorder()
	handler.Search(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearch_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// Create multiple orgs.
	for i := 0; i < 5; i++ {
		org := &models.Org{Name: fmt.Sprintf("Paginate Org %d", i), Slug: fmt.Sprintf("paginate-org-%d", i), Metadata: "{}"}
		require.NoError(t, db.Create(org).Error)
	}

	results, pageInfo, err := repo.Search(context.Background(), "Paginate", nil, pagination.Params{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)

	// Second page with cursor.
	results2, _, err := repo.Search(context.Background(), "Paginate", nil, pagination.Params{Limit: 2, Cursor: pageInfo.NextCursor})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results2), 1)
}

func TestHandler_Search_WithTypeFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)

	org := &models.Org{Name: "Filter Handler Org", Slug: "filter-handler", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=filter&type=org", nil)
	w := httptest.NewRecorder()
	handler.Search(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearch_OnlySpecialChars(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	results, pageInfo, err := repo.Search(context.Background(), `"'*()`, nil, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Nil(t, results)
	assert.NotNil(t, pageInfo)
}
