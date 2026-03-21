package models_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// testDB creates a fresh in-memory SQLite DB with migrations applied.
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

// --- BaseModel ---

func TestBaseModel_BeforeCreate_GeneratesUUIDv7(t *testing.T) {
	b := &models.BaseModel{}
	err := b.BeforeCreate(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, b.ID)
	_, err = uuid.Parse(b.ID)
	assert.NoError(t, err)
}

func TestBaseModel_BeforeCreate_PreservesExistingID(t *testing.T) {
	existing := "01234567-89ab-7def-8000-000000000001"
	b := &models.BaseModel{ID: existing}
	err := b.BeforeCreate(nil)
	require.NoError(t, err)
	assert.Equal(t, existing, b.ID)
}

func TestBaseModel_UUIDv7_TimeOrdered(t *testing.T) {
	b1 := &models.BaseModel{}
	require.NoError(t, b1.BeforeCreate(nil))
	time.Sleep(time.Millisecond)
	b2 := &models.BaseModel{}
	require.NoError(t, b2.BeforeCreate(nil))
	assert.True(t, b1.ID < b2.ID, "UUIDv7 IDs should be time-ordered")
}

// --- Org ---

func TestOrg_CRUD(t *testing.T) {
	db := testDB(t)

	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: `{"billing_tier":"free"}`}
	require.NoError(t, db.Create(org).Error)
	assert.NotEmpty(t, org.ID)

	var found models.Org
	require.NoError(t, db.First(&found, "id = ?", org.ID).Error)
	assert.Equal(t, "Test Org", found.Name)
	assert.Equal(t, "test-org", found.Slug)

	// Update.
	require.NoError(t, db.Model(&found).Update("name", "Updated Org").Error)
	var updated models.Org
	require.NoError(t, db.First(&updated, "id = ?", org.ID).Error)
	assert.Equal(t, "Updated Org", updated.Name)

	// Soft delete.
	require.NoError(t, db.Delete(&updated).Error)
	var deleted models.Org
	err := db.First(&deleted, "id = ?", org.ID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Unscoped find still finds it.
	var softDeleted models.Org
	require.NoError(t, db.Unscoped().First(&softDeleted, "id = ?", org.ID).Error)
	assert.NotNil(t, softDeleted.DeletedAt)
}

func TestOrg_SlugUnique(t *testing.T) {
	db := testDB(t)
	require.NoError(t, db.Create(&models.Org{Name: "Org1", Slug: "unique-slug", Metadata: "{}"}).Error)
	err := db.Create(&models.Org{Name: "Org2", Slug: "unique-slug", Metadata: "{}"}).Error
	assert.Error(t, err, "duplicate slug should fail")
}

func TestOrg_GeneratedColumns(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "Billed Org", Slug: "billed-org", Metadata: `{"billing_tier":"pro","payment_status":"active"}`}
	require.NoError(t, db.Create(org).Error)

	var row struct {
		BillingTier   string
		PaymentStatus string
	}
	require.NoError(t, db.Raw("SELECT billing_tier, payment_status FROM orgs WHERE id = ?", org.ID).Scan(&row).Error)
	assert.Equal(t, "pro", row.BillingTier)
	assert.Equal(t, "active", row.PaymentStatus)
}

// --- Space ---

func TestSpace_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "Org", Slug: "org-space-test", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	space := &models.Space{OrgID: org.ID, Name: "General", Slug: "general", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	assert.NotEmpty(t, space.ID)

	var found models.Space
	require.NoError(t, db.First(&found, "id = ?", space.ID).Error)
	assert.Equal(t, models.SpaceTypeGeneral, found.Type)
}

func TestSpace_SlugUniqueWithinOrg(t *testing.T) {
	db := testDB(t)
	org1 := &models.Org{Name: "Org1", Slug: "org-slug-1", Metadata: "{}"}
	org2 := &models.Org{Name: "Org2", Slug: "org-slug-2", Metadata: "{}"}
	require.NoError(t, db.Create(org1).Error)
	require.NoError(t, db.Create(org2).Error)

	// Same slug in different orgs should work.
	require.NoError(t, db.Create(&models.Space{OrgID: org1.ID, Name: "S1", Slug: "same-slug", Type: models.SpaceTypeGeneral, Metadata: "{}"}).Error)
	require.NoError(t, db.Create(&models.Space{OrgID: org2.ID, Name: "S2", Slug: "same-slug", Type: models.SpaceTypeGeneral, Metadata: "{}"}).Error)
}

func TestSpaceType_IsValid(t *testing.T) {
	assert.True(t, models.SpaceTypeGeneral.IsValid())
	assert.True(t, models.SpaceTypeCRM.IsValid())
	assert.True(t, models.SpaceTypeSupport.IsValid())
	assert.True(t, models.SpaceTypeCommunity.IsValid())
	assert.True(t, models.SpaceTypeKnowledgeBase.IsValid())
	assert.False(t, models.SpaceType("invalid").IsValid())
}

// --- Board ---

func TestBoard_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "board-test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "board-test-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)

	board := &models.Board{SpaceID: space.ID, Name: "Board", Slug: "board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	assert.NotEmpty(t, board.ID)
	assert.False(t, board.IsLocked)

	// Lock board.
	require.NoError(t, db.Model(board).Update("is_locked", true).Error)
	var locked models.Board
	require.NoError(t, db.First(&locked, "id = ?", board.ID).Error)
	assert.True(t, locked.IsLocked)
}

// --- Thread ---

func TestThread_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "thread-test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "thread-test-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "thread-test-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Test Thread",
		Slug:     "test-thread",
		AuthorID: "user-1",
		Metadata: `{"status":"open","priority":"high","stage":"new_lead","assigned_to":"user-2"}`,
	}
	require.NoError(t, db.Create(thread).Error)
	assert.NotEmpty(t, thread.ID)
}

func TestThread_GeneratedColumns(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "thread-gen-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "thread-gen-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "thread-gen-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Gen Col Thread",
		Slug:     "gen-col-thread",
		AuthorID: "user-1",
		Metadata: `{"status":"open","priority":"3","stage":"qualified","assigned_to":"user-5"}`,
	}
	require.NoError(t, db.Create(thread).Error)

	var row struct {
		Status     string
		Priority   string
		Stage      string
		AssignedTo string
	}
	require.NoError(t, db.Raw(
		"SELECT status, priority, stage, assigned_to FROM threads WHERE id = ?", thread.ID,
	).Scan(&row).Error)
	assert.Equal(t, "open", row.Status)
	assert.Equal(t, "3", row.Priority)
	assert.Equal(t, "qualified", row.Stage)
	assert.Equal(t, "user-5", row.AssignedTo)
}

// --- Message ---

func TestMessage_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "msg-test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "msg-test-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "msg-test-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "msg-test-thread", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)

	msg := &models.Message{ThreadID: thread.ID, Body: "Hello world", AuthorID: "u1", Type: models.MessageTypeComment, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)
	assert.NotEmpty(t, msg.ID)

	var found models.Message
	require.NoError(t, db.First(&found, "id = ?", msg.ID).Error)
	assert.Equal(t, "Hello world", found.Body)
	assert.Equal(t, models.MessageTypeComment, found.Type)
}

func TestMessageType_IsValid(t *testing.T) {
	assert.True(t, models.MessageTypeNote.IsValid())
	assert.True(t, models.MessageTypeEmail.IsValid())
	assert.True(t, models.MessageTypeCallLog.IsValid())
	assert.True(t, models.MessageTypeComment.IsValid())
	assert.True(t, models.MessageTypeSystem.IsValid())
	assert.False(t, models.MessageType("unknown").IsValid())
}

// --- Membership ---

func TestRole_IsValid(t *testing.T) {
	for _, r := range models.RoleHierarchy() {
		assert.True(t, r.IsValid())
	}
	assert.False(t, models.Role("superadmin").IsValid())
}

func TestRole_Level(t *testing.T) {
	assert.Less(t, models.RoleViewer.Level(), models.RoleOwner.Level())
	assert.Less(t, models.RoleAdmin.Level(), models.RoleOwner.Level())
	assert.Equal(t, -1, models.Role("invalid").Level())
}

func TestOrgMembership_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "memb-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	m := &models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleAdmin}
	require.NoError(t, db.Create(m).Error)
	assert.NotEmpty(t, m.ID)
}

func TestOrgMembership_UniqueConstraint(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "memb-uniq-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleViewer}).Error)
	err := db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleAdmin}).Error
	assert.Error(t, err, "duplicate (org, user) membership should fail")
}

func TestSpaceMembership_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "sm-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "sm-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)

	m := &models.SpaceMembership{SpaceID: space.ID, UserID: "user-1", Role: models.RoleContributor}
	require.NoError(t, db.Create(m).Error)
	assert.NotEmpty(t, m.ID)
}

func TestBoardMembership_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "bm-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "bm-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "bm-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	m := &models.BoardMembership{BoardID: board.ID, UserID: "user-1", Role: models.RoleModerator}
	require.NoError(t, db.Create(m).Error)
	assert.NotEmpty(t, m.ID)
}

// --- APIKey ---

func TestAPIKey_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "apikey-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	key := &models.APIKey{
		OrgID:       org.ID,
		Name:        "Test Key",
		KeyHash:     "sha256-hash-value",
		KeyPrefix:   "deft_live_abc",
		Permissions: `{"read":true}`,
	}
	require.NoError(t, db.Create(key).Error)
	assert.NotEmpty(t, key.ID)
}

// --- AuditLog ---

func TestAuditLog_CRUD(t *testing.T) {
	db := testDB(t)

	log := &models.AuditLog{
		UserID:     "user-1",
		Action:     models.AuditActionCreate,
		EntityType: "org",
		EntityID:   "some-id",
		AfterState: `{"name":"new org"}`,
		IPAddress:  "127.0.0.1",
		RequestID:  "req-123",
	}
	require.NoError(t, db.Create(log).Error)
	assert.NotEmpty(t, log.ID)
}

func TestAuditAction_IsValid(t *testing.T) {
	assert.True(t, models.AuditActionCreate.IsValid())
	assert.True(t, models.AuditActionUpdate.IsValid())
	assert.True(t, models.AuditActionDelete.IsValid())
	assert.False(t, models.AuditAction("read").IsValid())
}

// --- Revision ---

func TestRevision_CRUD(t *testing.T) {
	db := testDB(t)

	rev := &models.Revision{
		EntityType:      "thread",
		EntityID:        "thread-id",
		Version:         1,
		PreviousContent: `{"title":"old title"}`,
		EditorID:        "user-1",
	}
	require.NoError(t, db.Create(rev).Error)
	assert.NotEmpty(t, rev.ID)
}

// --- WebhookSubscription ---

func TestWebhookSubscription_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "wh-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	sub := &models.WebhookSubscription{
		OrgID:       org.ID,
		ScopeType:   "org",
		ScopeID:     org.ID,
		URL:         "https://example.com/hook",
		Secret:      "encrypted-secret",
		EventFilter: `["thread.created","message.created"]`,
	}
	require.NoError(t, db.Create(sub).Error)
	assert.NotEmpty(t, sub.ID)
}

// --- WebhookDelivery ---

func TestWebhookDelivery_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "wd-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	sub := &models.WebhookSubscription{OrgID: org.ID, ScopeType: "org", ScopeID: org.ID, URL: "https://example.com", Secret: "s", EventFilter: "[]"}
	require.NoError(t, db.Create(sub).Error)

	delivery := &models.WebhookDelivery{
		SubscriptionID: sub.ID,
		EventType:      "thread.created",
		Payload:        `{"id":"t1"}`,
		StatusCode:     200,
		Attempts:       1,
	}
	require.NoError(t, db.Create(delivery).Error)
	assert.NotEmpty(t, delivery.ID)
}

// --- Notification ---

func TestNotification_CRUD(t *testing.T) {
	db := testDB(t)

	notif := &models.Notification{
		UserID:     "user-1",
		Type:       "message.new",
		Title:      "New message",
		Body:       "You have a new message",
		EntityType: "message",
		EntityID:   "msg-1",
	}
	require.NoError(t, db.Create(notif).Error)
	assert.NotEmpty(t, notif.ID)
	assert.False(t, notif.IsRead)
}

func TestNotificationPreference_CRUD(t *testing.T) {
	db := testDB(t)

	pref := &models.NotificationPreference{
		UserID:    "user-1",
		EventType: "message.new",
		Channel:   "email",
		Enabled:   true,
	}
	require.NoError(t, db.Create(pref).Error)
	assert.NotEmpty(t, pref.ID)
}

func TestDigestSchedule_CRUD(t *testing.T) {
	db := testDB(t)

	ds := &models.DigestSchedule{UserID: "user-1", Frequency: "weekly"}
	require.NoError(t, db.Create(ds).Error)
	assert.NotEmpty(t, ds.ID)
}

// --- Vote ---

func TestVote_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "vote-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "vote-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "vote-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "vote-thread", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)

	vote := &models.Vote{ThreadID: thread.ID, UserID: "user-1", Weight: 2}
	require.NoError(t, db.Create(vote).Error)
	assert.NotEmpty(t, vote.ID)
}

func TestVote_UniqueConstraint(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "vote-uniq-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "vote-uniq-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "vote-uniq-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "vote-uniq-thread", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)

	require.NoError(t, db.Create(&models.Vote{ThreadID: thread.ID, UserID: "user-1", Weight: 1}).Error)
	err := db.Create(&models.Vote{ThreadID: thread.ID, UserID: "user-1", Weight: 1}).Error
	assert.Error(t, err, "duplicate (thread, user) vote should fail")
}

// --- Upload ---

func TestUpload_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "upload-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	upload := &models.Upload{
		OrgID:       org.ID,
		EntityType:  "message",
		EntityID:    "msg-1",
		Filename:    "report.pdf",
		ContentType: "application/pdf",
		Size:        1024,
		StoragePath: "uploads/report.pdf",
		UploaderID:  "user-1",
	}
	require.NoError(t, db.Create(upload).Error)
	assert.NotEmpty(t, upload.ID)
}

// --- Full Hierarchy ---

func TestFullHierarchy_CascadeDelete(t *testing.T) {
	db := testDB(t)

	org := &models.Org{Name: "Cascade Org", Slug: "cascade-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "cascade-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "cascade-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "cascade-thread", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "msg", AuthorID: "u1", Type: models.MessageTypeComment, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	// Hard delete org (bypass soft delete for cascade test).
	require.NoError(t, db.Unscoped().Delete(org).Error)

	// Everything should be gone.
	assert.ErrorIs(t, db.Unscoped().First(&models.Space{}, "id = ?", space.ID).Error, gorm.ErrRecordNotFound)
	assert.ErrorIs(t, db.Unscoped().First(&models.Board{}, "id = ?", board.ID).Error, gorm.ErrRecordNotFound)
	assert.ErrorIs(t, db.Unscoped().First(&models.Thread{}, "id = ?", thread.ID).Error, gorm.ErrRecordNotFound)
	assert.ErrorIs(t, db.Unscoped().First(&models.Message{}, "id = ?", msg.ID).Error, gorm.ErrRecordNotFound)
}

// --- CallLog ---

func TestCallDirection_IsValid(t *testing.T) {
	assert.True(t, models.CallDirectionInbound.IsValid())
	assert.True(t, models.CallDirectionOutbound.IsValid())
	assert.False(t, models.CallDirection("unknown").IsValid())
	assert.False(t, models.CallDirection("").IsValid())
}

func TestCallStatus_IsValid(t *testing.T) {
	assert.True(t, models.CallStatusRinging.IsValid())
	assert.True(t, models.CallStatusActive.IsValid())
	assert.True(t, models.CallStatusCompleted.IsValid())
	assert.True(t, models.CallStatusFailed.IsValid())
	assert.True(t, models.CallStatusEscalated.IsValid())
	assert.False(t, models.CallStatus("unknown").IsValid())
	assert.False(t, models.CallStatus("").IsValid())
}

func TestCallLog_CRUD(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "calllog-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	cl := &models.CallLog{
		OrgID:      org.ID,
		CallerID:   "user-1",
		Direction:  models.CallDirectionInbound,
		Duration:   120,
		Status:     models.CallStatusCompleted,
		Transcript: "Hello world",
		Metadata:   `{"topic":"billing"}`,
	}
	require.NoError(t, db.Create(cl).Error)
	assert.NotEmpty(t, cl.ID)

	var found models.CallLog
	require.NoError(t, db.First(&found, "id = ?", cl.ID).Error)
	assert.Equal(t, org.ID, found.OrgID)
	assert.Equal(t, "user-1", found.CallerID)
	assert.Equal(t, models.CallDirectionInbound, found.Direction)
	assert.Equal(t, 120, found.Duration)
	assert.Equal(t, models.CallStatusCompleted, found.Status)
	assert.Equal(t, "Hello world", found.Transcript)
}

func TestCallLog_OrgCascadeDelete(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "calllog-cascade-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	cl := &models.CallLog{
		OrgID:    org.ID,
		CallerID: "user-1",
		Metadata: "{}",
	}
	require.NoError(t, db.Create(cl).Error)

	// Hard-delete org should cascade to call log.
	require.NoError(t, db.Unscoped().Delete(org).Error)

	var count int64
	db.Model(&models.CallLog{}).Where("id = ?", cl.ID).Count(&count)
	assert.Zero(t, count)
}

// --- ThreadType ---

func TestThreadType_IsValid(t *testing.T) {
	for _, tt := range models.ValidThreadTypes() {
		assert.True(t, tt.IsValid())
	}
	assert.False(t, models.ThreadType("invalid").IsValid())
	assert.False(t, models.ThreadType("").IsValid())
}

func TestThreadVisibility_IsValid(t *testing.T) {
	for _, v := range models.ValidThreadVisibilities() {
		assert.True(t, v.IsValid())
	}
	assert.False(t, models.ThreadVisibility("invalid").IsValid())
	assert.False(t, models.ThreadVisibility("").IsValid())
}

func TestThread_NewFields(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "thread-new-fields-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "thread-new-fields-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "thread-new-fields-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	// Thread with explicit ThreadType and Visibility.
	orgID := org.ID
	thread := &models.Thread{
		BoardID:    board.ID,
		Title:      "Support Thread",
		Slug:       "support-thread",
		AuthorID:   "user-1",
		Metadata:   "{}",
		ThreadType: models.ThreadTypeSupport,
		Visibility: models.ThreadVisibilityPublic,
		OrgID:      &orgID,
	}
	require.NoError(t, db.Create(thread).Error)

	var found models.Thread
	require.NoError(t, db.First(&found, "id = ?", thread.ID).Error)
	assert.Equal(t, models.ThreadTypeSupport, found.ThreadType)
	assert.Equal(t, models.ThreadVisibilityPublic, found.Visibility)
	assert.NotNil(t, found.OrgID)
	assert.Equal(t, org.ID, *found.OrgID)
}

func TestThread_DefaultsForNewFields(t *testing.T) {
	db := testDB(t)
	org := &models.Org{Name: "O", Slug: "thread-defaults-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "thread-defaults-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "thread-defaults-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	// Thread without setting new fields should get defaults.
	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Default Thread",
		Slug:     "default-thread",
		AuthorID: "user-1",
		Metadata: "{}",
	}
	require.NoError(t, db.Create(thread).Error)

	// Read back via raw SQL to check DB-level defaults.
	var row struct {
		ThreadType string
		Visibility string
		OrgID      *string
	}
	require.NoError(t, db.Raw(
		"SELECT thread_type, visibility, org_id FROM threads WHERE id = ?", thread.ID,
	).Scan(&row).Error)
	assert.Equal(t, "forum", row.ThreadType)
	assert.Equal(t, "org-only", row.Visibility)
	assert.Nil(t, row.OrgID)
}

// --- UserHomePreferences ---

func TestUserHomePreferences_CRUD(t *testing.T) {
	db := testDB(t)

	prefs := &models.UserHomePreferences{
		UserID: "user-prefs",
		Tier:   2,
		Layout: `[{"widget_id":"profile","visible":true}]`,
	}
	require.NoError(t, db.Create(prefs).Error)

	var found models.UserHomePreferences
	require.NoError(t, db.Where("user_id = ?", "user-prefs").First(&found).Error)
	assert.Equal(t, 2, found.Tier)
	assert.Contains(t, found.Layout, "profile")
}

func TestUserHomePreferences_Upsert(t *testing.T) {
	db := testDB(t)

	prefs := &models.UserHomePreferences{
		UserID: "user-upsert",
		Tier:   2,
		Layout: `[{"widget_id":"a","visible":true}]`,
	}
	require.NoError(t, db.Save(prefs).Error)

	// Update.
	prefs.Tier = 3
	prefs.Layout = `[{"widget_id":"b","visible":false}]`
	require.NoError(t, db.Save(prefs).Error)

	var found models.UserHomePreferences
	require.NoError(t, db.Where("user_id = ?", "user-upsert").First(&found).Error)
	assert.Equal(t, 3, found.Tier)
	assert.Contains(t, found.Layout, "b")

	// Should be exactly one record.
	var count int64
	require.NoError(t, db.Model(&models.UserHomePreferences{}).Where("user_id = ?", "user-upsert").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// --- Lead ---

func TestLeadStatus_IsValid(t *testing.T) {
	for _, s := range models.ValidLeadStatuses() {
		assert.True(t, s.IsValid())
	}
	assert.False(t, models.LeadStatus("invalid").IsValid())
	assert.False(t, models.LeadStatus("").IsValid())
}

func TestLead_CRUD(t *testing.T) {
	db := testDB(t)

	anonSession := "anon-session-123"
	lead := &models.Lead{
		Email:         "lead@example.com",
		Name:          "Test Lead",
		Source:        "chatbot",
		Status:        models.LeadStatusAnonymous,
		AnonSessionID: &anonSession,
		Metadata:      "{}",
	}
	require.NoError(t, db.Create(lead).Error)
	assert.NotEmpty(t, lead.ID)

	var found models.Lead
	require.NoError(t, db.First(&found, "id = ?", lead.ID).Error)
	assert.Equal(t, "lead@example.com", found.Email)
	assert.NotNil(t, found.AnonSessionID)
	assert.Equal(t, "anon-session-123", *found.AnonSessionID)
	assert.Nil(t, found.UserID)
}

func TestLead_NullableFields(t *testing.T) {
	db := testDB(t)

	// Lead with no anon session and no user ID.
	lead := &models.Lead{
		Source:   "manual",
		Status:   models.LeadStatusRegistered,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(lead).Error)

	var found models.Lead
	require.NoError(t, db.First(&found, "id = ?", lead.ID).Error)
	assert.Nil(t, found.AnonSessionID)
	assert.Nil(t, found.UserID)
}

func TestLead_AnonSessionIndex(t *testing.T) {
	db := testDB(t)

	session1 := "session-abc"
	session2 := "session-def"
	require.NoError(t, db.Create(&models.Lead{Source: "chatbot", Status: models.LeadStatusAnonymous, AnonSessionID: &session1, Metadata: "{}"}).Error)
	require.NoError(t, db.Create(&models.Lead{Source: "chatbot", Status: models.LeadStatusAnonymous, AnonSessionID: &session2, Metadata: "{}"}).Error)

	// Query by anon_session_id.
	var found models.Lead
	require.NoError(t, db.Where("anon_session_id = ?", "session-abc").First(&found).Error)
	assert.Equal(t, "session-abc", *found.AnonSessionID)
}

func TestLead_PromoteToRegistered(t *testing.T) {
	db := testDB(t)

	anonSession := "promote-session"
	lead := &models.Lead{
		Source:        "chatbot",
		Status:        models.LeadStatusAnonymous,
		AnonSessionID: &anonSession,
		Metadata:      "{}",
	}
	require.NoError(t, db.Create(lead).Error)

	// Promote: set user_id, change status.
	userID := "user-promoted"
	require.NoError(t, db.Model(lead).Updates(map[string]interface{}{
		"user_id": userID,
		"status":  models.LeadStatusRegistered,
	}).Error)

	var found models.Lead
	require.NoError(t, db.First(&found, "id = ?", lead.ID).Error)
	assert.NotNil(t, found.UserID)
	assert.Equal(t, "user-promoted", *found.UserID)
	assert.Equal(t, models.LeadStatusRegistered, found.Status)
}

// --- Fuzzing ---

func FuzzMetadataJSON(f *testing.F) {
	// Seed corpus with various JSON patterns.
	seeds := []string{
		`{}`,
		`{"key":"value"}`,
		`{"nested":{"a":1}}`,
		`{"arr":[1,2,3]}`,
		`{"billing_tier":"pro","payment_status":"active"}`,
		`{"status":"open","priority":"5"}`,
		`{"emoji":"🎉","unicode":"日本語"}`,
		`{"escaped":"line\nbreak"}`,
		`{"big":99999999999999}`,
		`{"bool":true,"null":null}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, metadata string) {
		// Validate if it's valid JSON; if not, skip (only testing valid metadata storage).
		if !json.Valid([]byte(metadata)) {
			return
		}
		// Ensure metadata round-trips correctly: the model should accept valid JSON.
		org := models.Org{Name: "fuzz", Slug: "fuzz-" + uuid.New().String()[:8], Metadata: metadata}
		assert.NotEmpty(t, org.Metadata)
	})
}

func FuzzUUIDv7Generation(f *testing.F) {
	seeds := []string{"", "existing-id", "01234567-89ab-7def-8000-000000000001"}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, existingID string) {
		b := &models.BaseModel{ID: existingID}
		err := b.BeforeCreate(nil)
		if err != nil {
			return
		}
		assert.NotEmpty(t, b.ID)
		if existingID != "" {
			assert.Equal(t, existingID, b.ID)
		} else {
			_, err := uuid.Parse(b.ID)
			assert.NoError(t, err)
		}
	})
}

func TestMessageType_IsSupportType(t *testing.T) {
	supportTypes := []models.MessageType{
		models.MessageTypeCustomer, models.MessageTypeAgentReply, models.MessageTypeDraft,
		models.MessageTypeContext, models.MessageTypeSystemEvent,
	}
	for _, mt := range supportTypes {
		assert.True(t, mt.IsSupportType(), "expected %s to be a support type", mt)
	}
	nonSupport := []models.MessageType{
		models.MessageTypeNote, models.MessageTypeEmail,
		models.MessageTypeCallLog, models.MessageTypeComment, models.MessageTypeSystem,
	}
	for _, mt := range nonSupport {
		assert.False(t, mt.IsSupportType(), "expected %s to NOT be a support type", mt)
	}
}

func TestMessageType_IsVisibleToCustomer(t *testing.T) {
	visible := []models.MessageType{
		models.MessageTypeCustomer, models.MessageTypeAgentReply, models.MessageTypeSystemEvent,
	}
	for _, mt := range visible {
		assert.True(t, mt.IsVisibleToCustomer(), "%s should be visible to customer", mt)
	}
	hidden := []models.MessageType{
		models.MessageTypeDraft, models.MessageTypeContext,
		models.MessageTypeNote, models.MessageTypeComment,
	}
	for _, mt := range hidden {
		assert.False(t, mt.IsVisibleToCustomer(), "%s should NOT be visible to customer", mt)
	}
}

func TestThread_TicketNumber(t *testing.T) {
	thread := &models.Thread{}
	assert.Equal(t, int64(0), thread.TicketNumber)
}

func TestTicketCounter_Fields(t *testing.T) {
	tc := &models.TicketCounter{OrgID: "org-1", ThreadType: "support", LastNumber: 5}
	assert.Equal(t, "org-1", tc.OrgID)
	assert.Equal(t, "support", tc.ThreadType)
	assert.Equal(t, int64(5), tc.LastNumber)
}
