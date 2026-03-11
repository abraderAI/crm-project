package membership

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

type testEnv struct {
	db      *gorm.DB
	orgID   string
	spaceID string
	boardID string
}

func setupDB(t *testing.T) testEnv {
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

	return testEnv{db: db, orgID: org.ID, spaceID: sp.ID, boardID: bd.ID}
}

// --- Org Membership ---

func TestOrgMembership_Add(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &models.OrgMembership{OrgID: env.orgID, UserID: "user1", Role: models.RoleAdmin}
		err := repo.AddOrgMember(ctx, m)
		require.NoError(t, err)
		assert.NotEmpty(t, m.ID)
	})

	t.Run("duplicate user", func(t *testing.T) {
		m := &models.OrgMembership{OrgID: env.orgID, UserID: "user1", Role: models.RoleViewer}
		err := repo.AddOrgMember(ctx, m)
		assert.Error(t, err) // unique constraint violation
	})
}

func TestOrgMembership_Get(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	m := &models.OrgMembership{OrgID: env.orgID, UserID: "user_get", Role: models.RoleAdmin}
	require.NoError(t, repo.AddOrgMember(ctx, m))

	t.Run("found", func(t *testing.T) {
		found, err := repo.GetOrgMember(ctx, env.orgID, "user_get")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, models.RoleAdmin, found.Role)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.GetOrgMember(ctx, env.orgID, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestOrgMembership_List(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	require.NoError(t, repo.AddOrgMember(ctx, &models.OrgMembership{OrgID: env.orgID, UserID: "u1", Role: models.RoleAdmin}))
	require.NoError(t, repo.AddOrgMember(ctx, &models.OrgMembership{OrgID: env.orgID, UserID: "u2", Role: models.RoleViewer}))

	members, err := repo.ListOrgMembers(ctx, env.orgID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestOrgMembership_Update(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	m := &models.OrgMembership{OrgID: env.orgID, UserID: "user_upd", Role: models.RoleViewer}
	require.NoError(t, repo.AddOrgMember(ctx, m))

	m.Role = models.RoleModerator
	err := repo.UpdateOrgMember(ctx, m)
	require.NoError(t, err)

	found, err := repo.GetOrgMember(ctx, env.orgID, "user_upd")
	require.NoError(t, err)
	assert.Equal(t, models.RoleModerator, found.Role)
}

func TestOrgMembership_Remove(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	m := &models.OrgMembership{OrgID: env.orgID, UserID: "user_rm", Role: models.RoleAdmin}
	require.NoError(t, repo.AddOrgMember(ctx, m))

	t.Run("success", func(t *testing.T) {
		err := repo.RemoveOrgMember(ctx, env.orgID, "user_rm")
		require.NoError(t, err)
		found, err := repo.GetOrgMember(ctx, env.orgID, "user_rm")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.RemoveOrgMember(ctx, env.orgID, "nonexistent")
		assert.Error(t, err)
	})
}

func TestOrgMembership_CountOwners(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	require.NoError(t, repo.AddOrgMember(ctx, &models.OrgMembership{OrgID: env.orgID, UserID: "owner1", Role: models.RoleOwner}))
	require.NoError(t, repo.AddOrgMember(ctx, &models.OrgMembership{OrgID: env.orgID, UserID: "admin1", Role: models.RoleAdmin}))

	count, err := repo.CountOrgOwners(ctx, env.orgID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Add second owner.
	require.NoError(t, repo.AddOrgMember(ctx, &models.OrgMembership{OrgID: env.orgID, UserID: "owner2", Role: models.RoleOwner}))
	count, err = repo.CountOrgOwners(ctx, env.orgID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// --- Space Membership ---

func TestSpaceMembership_CRUD(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	t.Run("add", func(t *testing.T) {
		m := &models.SpaceMembership{SpaceID: env.spaceID, UserID: "suser1", Role: models.RoleContributor}
		err := repo.AddSpaceMember(ctx, m)
		require.NoError(t, err)
		assert.NotEmpty(t, m.ID)
	})

	t.Run("get", func(t *testing.T) {
		found, err := repo.GetSpaceMember(ctx, env.spaceID, "suser1")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, models.RoleContributor, found.Role)
	})

	t.Run("list", func(t *testing.T) {
		require.NoError(t, repo.AddSpaceMember(ctx, &models.SpaceMembership{SpaceID: env.spaceID, UserID: "suser2", Role: models.RoleViewer}))
		members, err := repo.ListSpaceMembers(ctx, env.spaceID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(members), 2)
	})

	t.Run("update", func(t *testing.T) {
		found, _ := repo.GetSpaceMember(ctx, env.spaceID, "suser1")
		found.Role = models.RoleAdmin
		err := repo.UpdateSpaceMember(ctx, found)
		require.NoError(t, err)
		updated, _ := repo.GetSpaceMember(ctx, env.spaceID, "suser1")
		assert.Equal(t, models.RoleAdmin, updated.Role)
	})

	t.Run("remove", func(t *testing.T) {
		err := repo.RemoveSpaceMember(ctx, env.spaceID, "suser1")
		require.NoError(t, err)
		found, _ := repo.GetSpaceMember(ctx, env.spaceID, "suser1")
		assert.Nil(t, found)
	})
}

// --- Board Membership ---

func TestBoardMembership_CRUD(t *testing.T) {
	env := setupDB(t)
	repo := NewRepository(env.db)
	ctx := context.Background()

	t.Run("add", func(t *testing.T) {
		m := &models.BoardMembership{BoardID: env.boardID, UserID: "buser1", Role: models.RoleModerator}
		err := repo.AddBoardMember(ctx, m)
		require.NoError(t, err)
		assert.NotEmpty(t, m.ID)
	})

	t.Run("get", func(t *testing.T) {
		found, err := repo.GetBoardMember(ctx, env.boardID, "buser1")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, models.RoleModerator, found.Role)
	})

	t.Run("list", func(t *testing.T) {
		require.NoError(t, repo.AddBoardMember(ctx, &models.BoardMembership{BoardID: env.boardID, UserID: "buser2", Role: models.RoleViewer}))
		members, err := repo.ListBoardMembers(ctx, env.boardID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(members), 2)
	})

	t.Run("update", func(t *testing.T) {
		found, _ := repo.GetBoardMember(ctx, env.boardID, "buser1")
		found.Role = models.RoleOwner
		err := repo.UpdateBoardMember(ctx, found)
		require.NoError(t, err)
		updated, _ := repo.GetBoardMember(ctx, env.boardID, "buser1")
		assert.Equal(t, models.RoleOwner, updated.Role)
	})

	t.Run("remove", func(t *testing.T) {
		err := repo.RemoveBoardMember(ctx, env.boardID, "buser1")
		require.NoError(t, err)
		found, _ := repo.GetBoardMember(ctx, env.boardID, "buser1")
		assert.Nil(t, found)
	})
}
