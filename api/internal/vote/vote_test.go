package vote

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
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func seedThread(t *testing.T, db *gorm.DB) *models.Thread {
	t.Helper()
	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "Space", Slug: "space", Metadata: "{}", Type: "general"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Board", Slug: "board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "Test Thread", Slug: "test-thread", AuthorID: "author1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	return thread
}

func TestRepository_FindByUserAndThread_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	thread := seedThread(t, db)
	vote, err := repo.FindByUserAndThread(context.Background(), "no-user", thread.ID)
	assert.NoError(t, err)
	assert.Nil(t, vote)
}

func TestRepository_CreateAndFind(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	thread := seedThread(t, db)

	v := &models.Vote{ThreadID: thread.ID, UserID: "user1", Weight: 2}
	require.NoError(t, repo.Create(context.Background(), v))
	assert.NotEmpty(t, v.ID)

	found, err := repo.FindByUserAndThread(context.Background(), "user1", thread.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "user1", found.UserID)
	assert.Equal(t, 2, found.Weight)
}

func TestRepository_Delete(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	thread := seedThread(t, db)

	v := &models.Vote{ThreadID: thread.ID, UserID: "user1", Weight: 1}
	require.NoError(t, repo.Create(context.Background(), v))
	require.NoError(t, repo.Delete(context.Background(), v.ID))

	found, err := repo.FindByUserAndThread(context.Background(), "user1", thread.ID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_RecalculateThreadScore(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	thread := seedThread(t, db)
	ctx := context.Background()

	v1 := &models.Vote{ThreadID: thread.ID, UserID: "user1", Weight: 3}
	v2 := &models.Vote{ThreadID: thread.ID, UserID: "user2", Weight: 5}
	require.NoError(t, repo.Create(ctx, v1))
	require.NoError(t, repo.Create(ctx, v2))

	score, err := repo.RecalculateThreadScore(ctx, thread.ID)
	require.NoError(t, err)
	assert.Equal(t, 8, score)

	// Verify thread is updated in DB.
	updated, err := repo.FindThread(ctx, thread.ID)
	require.NoError(t, err)
	assert.Equal(t, 8, updated.VoteScore)
}

func TestRepository_RecalculateThreadScore_Empty(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	thread := seedThread(t, db)

	score, err := repo.RecalculateThreadScore(context.Background(), thread.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, score)
}

func TestRepository_FindThread(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	thread := seedThread(t, db)

	found, err := repo.FindThread(context.Background(), thread.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, thread.Title, found.Title)
}

func TestRepository_FindThread_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)

	found, err := repo.FindThread(context.Background(), "nonexistent-id")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestWeightConfig_CalculateWeight(t *testing.T) {
	wc := DefaultWeightConfig()
	tests := []struct {
		name        string
		role        models.Role
		billingTier string
		want        int
	}{
		{"viewer free", models.RoleViewer, "free", 1},
		{"viewer pro", models.RoleViewer, "pro", 2},
		{"contributor enterprise", models.RoleContributor, "enterprise", 4},
		{"moderator free", models.RoleModerator, "free", 3},
		{"admin pro", models.RoleAdmin, "pro", 5},
		{"owner enterprise", models.RoleOwner, "enterprise", 7},
		{"unknown role", models.Role("unknown"), "free", 1},
		{"unknown tier", models.RoleViewer, "unknown", 1},
		{"commenter free", models.RoleCommenter, "free", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wc.CalculateWeight(tt.role, tt.billingTier)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultWeightConfig(t *testing.T) {
	wc := DefaultWeightConfig()
	assert.NotNil(t, wc)
	assert.Equal(t, 1, wc.DefaultWeight)
	assert.Len(t, wc.RoleWeights, 6)
	assert.Len(t, wc.TierBonuses, 3)
}

func TestService_Toggle_On(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	thread := seedThread(t, db)
	ctx := context.Background()

	result, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "free")
	require.NoError(t, err)
	assert.True(t, result.Voted)
	assert.Equal(t, 1, result.VoteScore)
	assert.Equal(t, 1, result.Weight)
}

func TestService_Toggle_Off(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	thread := seedThread(t, db)
	ctx := context.Background()

	// Vote on.
	_, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "free")
	require.NoError(t, err)

	// Vote off.
	result, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "free")
	require.NoError(t, err)
	assert.False(t, result.Voted)
	assert.Equal(t, 0, result.VoteScore)
}

func TestService_Toggle_MultipleUsers(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	thread := seedThread(t, db)
	ctx := context.Background()

	r1, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleContributor, "pro")
	require.NoError(t, err)
	assert.True(t, r1.Voted)
	assert.Equal(t, 3, r1.VoteScore) // contributor(2) + pro(1) = 3

	r2, err := svc.Toggle(ctx, thread.ID, "user2", models.RoleAdmin, "enterprise")
	require.NoError(t, err)
	assert.True(t, r2.Voted)
	assert.Equal(t, 9, r2.VoteScore) // 3 + admin(4) + enterprise(2) = 9
}

func TestService_Toggle_ThreadNotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	ctx := context.Background()

	_, err := svc.Toggle(ctx, "nonexistent", "user1", models.RoleViewer, "free")
	assert.Error(t, err)
	assert.Equal(t, "thread not found", err.Error())
}

func TestService_Toggle_ReVote(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	thread := seedThread(t, db)
	ctx := context.Background()

	// Toggle on, off, on again.
	_, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "free")
	require.NoError(t, err)
	_, err = svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "free")
	require.NoError(t, err)
	r, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "free")
	require.NoError(t, err)
	assert.True(t, r.Voted)
	assert.Equal(t, 1, r.VoteScore)
}

func TestService_GetWeightConfig(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil)
	wc := svc.GetWeightConfig()
	assert.NotNil(t, wc)
	assert.Equal(t, 1, wc.DefaultWeight)
}

func TestService_CustomWeightConfig(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	custom := &WeightConfig{
		RoleWeights:   map[models.Role]int{models.RoleViewer: 10},
		TierBonuses:   map[string]int{"gold": 100},
		DefaultWeight: 5,
	}
	svc := NewService(repo, custom)
	thread := seedThread(t, db)
	ctx := context.Background()

	r, err := svc.Toggle(ctx, thread.ID, "user1", models.RoleViewer, "gold")
	require.NoError(t, err)
	assert.True(t, r.Voted)
	assert.Equal(t, 110, r.VoteScore) // viewer(10) + gold(100) = 110
}
