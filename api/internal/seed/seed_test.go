package seed_test

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/seed"
)

func testDB(t *testing.T) *gorm.DB {
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

func TestRun_CreatesSystemOrgAndGlobalSpaces(t *testing.T) {
	db := testDB(t)
	require.NoError(t, seed.Run(db))

	// Verify system org.
	var sysOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.SystemOrgSlug).First(&sysOrg).Error)
	assert.Equal(t, "System", sysOrg.Name)

	// Verify 4 global spaces.
	var spaces []models.Space
	require.NoError(t, db.Where("org_id = ?", sysOrg.ID).Find(&spaces).Error)
	assert.Len(t, spaces, 4)

	slugs := make(map[string]bool)
	for _, sp := range spaces {
		slugs[sp.Slug] = true
	}
	assert.True(t, slugs["global-docs"])
	assert.True(t, slugs["global-forum"])
	assert.True(t, slugs["global-support"])
	assert.True(t, slugs["global-leads"])
}

func TestRun_CreatesDeftOrgAndDepartmentSpaces(t *testing.T) {
	db := testDB(t)
	require.NoError(t, seed.Run(db))

	// Verify deft org.
	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	assert.Equal(t, "DEFT", deftOrg.Name)

	// Verify 3 department spaces.
	var spaces []models.Space
	require.NoError(t, db.Where("org_id = ?", deftOrg.ID).Find(&spaces).Error)
	assert.Len(t, spaces, 3)

	slugs := make(map[string]bool)
	for _, sp := range spaces {
		slugs[sp.Slug] = true
	}
	assert.True(t, slugs["deft-sales"])
	assert.True(t, slugs["deft-support"])
	assert.True(t, slugs["deft-finance"])
}

func TestRun_IsIdempotent(t *testing.T) {
	db := testDB(t)

	// Run seed twice.
	require.NoError(t, seed.Run(db))
	require.NoError(t, seed.Run(db))

	// Should still have exactly 2 orgs (system + deft).
	var orgs []models.Org
	require.NoError(t, db.Where("slug IN ?", []string{seed.SystemOrgSlug, seed.DeftOrgSlug}).Find(&orgs).Error)
	assert.Len(t, orgs, 2)

	// Should still have exactly 7 spaces (4 global + 3 deft).
	var totalSpaces int64
	require.NoError(t, db.Model(&models.Space{}).Where("org_id IN ?", []string{orgs[0].ID, orgs[1].ID}).Count(&totalSpaces).Error)
	assert.Equal(t, int64(7), totalSpaces)
}

func TestRun_SpaceTypes(t *testing.T) {
	db := testDB(t)
	require.NoError(t, seed.Run(db))

	tests := []struct {
		slug     string
		wantType models.SpaceType
	}{
		{"global-docs", models.SpaceTypeKnowledgeBase},
		{"global-forum", models.SpaceTypeCommunity},
		{"global-support", models.SpaceTypeSupport},
		{"global-leads", models.SpaceTypeCRM},
		{"deft-sales", models.SpaceTypeCRM},
		{"deft-support", models.SpaceTypeSupport},
		{"deft-finance", models.SpaceTypeGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			var space models.Space
			require.NoError(t, db.Where("slug = ?", tt.slug).First(&space).Error)
			assert.Equal(t, tt.wantType, space.Type)
		})
	}
}

func TestRun_CreatesDefaultBoardsForGlobalSpaces(t *testing.T) {
	db := testDB(t)
	require.NoError(t, seed.Run(db))

	// Each global space should have exactly one default board.
	globalSlugs := []string{"global-docs", "global-forum", "global-support", "global-leads"}
	for _, spaceSlug := range globalSlugs {
		t.Run(spaceSlug, func(t *testing.T) {
			var board models.Board
			err := db.Joins("JOIN spaces ON spaces.id = boards.space_id").
				Where("spaces.slug = ? AND boards.slug = ?", spaceSlug, "default").
				First(&board).Error
			require.NoError(t, err)
			assert.Equal(t, "Default", board.Name)
		})
	}
}

func TestRun_ThirdRunStillIdempotent(t *testing.T) {
	db := testDB(t)

	require.NoError(t, seed.Run(db))
	require.NoError(t, seed.Run(db))
	require.NoError(t, seed.Run(db))

	var orgCount int64
	require.NoError(t, db.Model(&models.Org{}).Where("slug IN ?", []string{seed.SystemOrgSlug, seed.DeftOrgSlug}).Count(&orgCount).Error)
	assert.Equal(t, int64(2), orgCount)
}
