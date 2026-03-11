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

func createOrg(t *testing.T, db *gorm.DB) *models.Org {
	t.Helper()
	o := &models.Org{Name: "test-org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(o).Error)
	return o
}

func createSpace(t *testing.T, db *gorm.DB, orgID string) *models.Space {
	t.Helper()
	sp := &models.Space{OrgID: orgID, Name: "test-space", Slug: "test-space", Metadata: "{}", Type: models.SpaceTypeGeneral}
	require.NoError(t, db.Create(sp).Error)
	return sp
}

func createBoard(t *testing.T, db *gorm.DB, spaceID string) *models.Board {
	t.Helper()
	b := &models.Board{SpaceID: spaceID, Name: "test-board", Slug: "test-board", Metadata: "{}"}
	require.NoError(t, db.Create(b).Error)
	return b
}

// --- Org Membership ---

func TestService_AddOrgMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))

	err := svc.AddOrgMember(context.Background(), org.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)
}

func TestService_AddOrgMember_DefaultRole(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))

	err := svc.AddOrgMember(context.Background(), org.ID, MemberInput{UserID: "user1"})
	require.NoError(t, err)

	members, err := svc.ListOrgMembers(context.Background(), org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, models.RoleViewer, members[0].Role)
}

func TestService_AddOrgMember_EmptyUserID(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))

	err := svc.AddOrgMember(context.Background(), org.ID, MemberInput{UserID: ""})
	assert.ErrorIs(t, err, ErrUserRequired)
}

func TestService_AddOrgMember_InvalidRole(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))

	err := svc.AddOrgMember(context.Background(), org.ID, MemberInput{UserID: "user1", Role: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestService_AddOrgMember_Duplicate(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)

	err = svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "user1", Role: "admin"})
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestService_ListOrgMembers(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "user" + string(rune('1'+i)), Role: "viewer"})
		require.NoError(t, err)
	}

	members, err := svc.ListOrgMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 3)
}

func TestService_UpdateOrgMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)

	members, err := svc.ListOrgMembers(ctx, org.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)

	err = svc.UpdateOrgMember(ctx, members[0].ID, MemberInput{Role: "admin"})
	require.NoError(t, err)
}

func TestService_UpdateOrgMember_InvalidRole(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)

	members, _ := svc.ListOrgMembers(ctx, org.ID)
	err = svc.UpdateOrgMember(ctx, members[0].ID, MemberInput{Role: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestService_UpdateOrgMember_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	err := svc.UpdateOrgMember(context.Background(), "nonexistent", MemberInput{Role: "admin"})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_UpdateOrgMember_LastOwner(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "owner1", Role: "owner"})
	require.NoError(t, err)

	members, _ := svc.ListOrgMembers(ctx, org.ID)
	err = svc.UpdateOrgMember(ctx, members[0].ID, MemberInput{Role: "viewer"})
	assert.ErrorIs(t, err, ErrLastOwner)
}

func TestService_RemoveOrgMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)

	members, _ := svc.ListOrgMembers(ctx, org.ID)
	err = svc.RemoveOrgMember(ctx, members[0].ID)
	require.NoError(t, err)

	members, _ = svc.ListOrgMembers(ctx, org.ID)
	assert.Len(t, members, 0)
}

func TestService_RemoveOrgMember_LastOwner(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	err := svc.AddOrgMember(ctx, org.ID, MemberInput{UserID: "owner1", Role: "owner"})
	require.NoError(t, err)

	members, _ := svc.ListOrgMembers(ctx, org.ID)
	err = svc.RemoveOrgMember(ctx, members[0].ID)
	assert.ErrorIs(t, err, ErrLastOwner)
}

func TestService_RemoveOrgMember_NotFound(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	err := svc.RemoveOrgMember(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Space Membership ---

func TestService_AddSpaceMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))

	err := svc.AddSpaceMember(context.Background(), sp.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)
}

func TestService_AddSpaceMember_EmptyUserID(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))

	err := svc.AddSpaceMember(context.Background(), sp.ID, MemberInput{UserID: ""})
	assert.ErrorIs(t, err, ErrUserRequired)
}

func TestService_AddSpaceMember_InvalidRole(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))

	err := svc.AddSpaceMember(context.Background(), sp.ID, MemberInput{UserID: "user1", Role: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestService_ListSpaceMembers(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "user1", Role: "viewer"})
	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "user2", Role: "admin"})

	members, err := svc.ListSpaceMembers(ctx, sp.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestService_UpdateSpaceMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "user1", Role: "viewer"})
	members, _ := svc.ListSpaceMembers(ctx, sp.ID)

	err := svc.UpdateSpaceMember(ctx, members[0].ID, MemberInput{Role: "admin"})
	require.NoError(t, err)
}

func TestService_UpdateSpaceMember_InvalidRole(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "user1", Role: "viewer"})
	members, _ := svc.ListSpaceMembers(ctx, sp.ID)

	err := svc.UpdateSpaceMember(ctx, members[0].ID, MemberInput{Role: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestService_RemoveSpaceMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddSpaceMember(ctx, sp.ID, MemberInput{UserID: "user1", Role: "viewer"})
	members, _ := svc.ListSpaceMembers(ctx, sp.ID)

	err := svc.RemoveSpaceMember(ctx, members[0].ID)
	require.NoError(t, err)
}

// --- Board Membership ---

func TestService_AddBoardMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))

	err := svc.AddBoardMember(context.Background(), b.ID, MemberInput{UserID: "user1", Role: "viewer"})
	require.NoError(t, err)
}

func TestService_AddBoardMember_EmptyUserID(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))

	err := svc.AddBoardMember(context.Background(), b.ID, MemberInput{UserID: ""})
	assert.ErrorIs(t, err, ErrUserRequired)
}

func TestService_AddBoardMember_InvalidRole(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))

	err := svc.AddBoardMember(context.Background(), b.ID, MemberInput{UserID: "user1", Role: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestService_ListBoardMembers(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddBoardMember(ctx, b.ID, MemberInput{UserID: "user1", Role: "viewer"})
	_ = svc.AddBoardMember(ctx, b.ID, MemberInput{UserID: "user2", Role: "admin"})

	members, err := svc.ListBoardMembers(ctx, b.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestService_UpdateBoardMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddBoardMember(ctx, b.ID, MemberInput{UserID: "user1", Role: "viewer"})
	members, _ := svc.ListBoardMembers(ctx, b.ID)

	err := svc.UpdateBoardMember(ctx, members[0].ID, MemberInput{Role: "admin"})
	require.NoError(t, err)
}

func TestService_RemoveBoardMember(t *testing.T) {
	db := testDB(t)
	org := createOrg(t, db)
	sp := createSpace(t, db, org.ID)
	b := createBoard(t, db, sp.ID)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_ = svc.AddBoardMember(ctx, b.ID, MemberInput{UserID: "user1", Role: "viewer"})
	members, _ := svc.ListBoardMembers(ctx, b.ID)

	err := svc.RemoveBoardMember(ctx, members[0].ID)
	require.NoError(t, err)
}

// --- Fuzz Tests ---

func FuzzAddOrgMember(f *testing.F) {
	f.Add("user1", "viewer")
	f.Add("", "")
	f.Add("user2", "invalid")
	f.Add("user3", "owner")
	f.Add("user4", "admin")
	f.Fuzz(func(t *testing.T, userID, role string) {
		db := testDB(t)
		org := createOrg(t, db)
		svc := NewService(NewRepository(db))
		_ = svc.AddOrgMember(context.Background(), org.ID, MemberInput{UserID: userID, Role: role})
	})
}
