package gdpr

import (
	"context"
	"encoding/json"
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

// seedUserData creates a full hierarchy of data for a user.
func seedUserData(t *testing.T, db *gorm.DB, userID string) *models.Org {
	t.Helper()

	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	space := &models.Space{OrgID: org.ID, Name: "Space", Slug: "space", Metadata: "{}", Type: "general"}
	require.NoError(t, db.Create(space).Error)

	board := &models.Board{SpaceID: space.ID, Name: "Board", Slug: "board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Test Thread",
		Slug:     "test-thread",
		AuthorID: userID,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(thread).Error)

	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     "Test message",
		AuthorID: userID,
		Metadata: "{}",
		Type:     models.MessageTypeComment,
	}
	require.NoError(t, db.Create(msg).Error)

	orgMember := &models.OrgMembership{OrgID: org.ID, UserID: userID, Role: models.RoleOwner}
	require.NoError(t, db.Create(orgMember).Error)

	spaceMember := &models.SpaceMembership{SpaceID: space.ID, UserID: userID, Role: models.RoleAdmin}
	require.NoError(t, db.Create(spaceMember).Error)

	boardMember := &models.BoardMembership{BoardID: board.ID, UserID: userID, Role: models.RoleContributor}
	require.NoError(t, db.Create(boardMember).Error)

	vote := &models.Vote{ThreadID: thread.ID, UserID: userID, Weight: 1}
	require.NoError(t, db.Create(vote).Error)

	notif := &models.Notification{
		UserID: userID,
		Type:   "test",
		Title:  "Test Notification",
		Body:   "Test body",
	}
	require.NoError(t, db.Create(notif).Error)

	pref := &models.NotificationPreference{
		UserID:    userID,
		EventType: "message.created",
		Channel:   "email",
		Enabled:   true,
	}
	require.NoError(t, db.Create(pref).Error)

	digest := &models.DigestSchedule{
		UserID:    userID,
		Frequency: "daily",
		Enabled:   true,
	}
	require.NoError(t, db.Create(digest).Error)

	upload := &models.Upload{
		OrgID:       org.ID,
		EntityType:  "thread",
		EntityID:    thread.ID,
		Filename:    "test.txt",
		ContentType: "text/plain",
		Size:        100,
		StoragePath: "/uploads/test.txt",
		UploaderID:  userID,
	}
	require.NoError(t, db.Create(upload).Error)

	callLog := &models.CallLog{
		OrgID:    org.ID,
		CallerID: userID,
		Status:   models.CallStatusCompleted,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(callLog).Error)

	audit := &models.AuditLog{
		UserID:     userID,
		Action:     models.AuditActionCreate,
		EntityType: "org",
		EntityID:   org.ID,
		IPAddress:  "127.0.0.1",
		RequestID:  "req-123",
	}
	require.NoError(t, db.Create(audit).Error)

	return org
}

// --- ExportUserData Tests ---

func TestExportUserData_Success(t *testing.T) {
	db := testDB(t)
	userID := "user_export_test"
	seedUserData(t, db, userID)
	svc := NewService(db)

	export, err := svc.ExportUserData(context.Background(), userID)
	require.NoError(t, err)

	assert.Equal(t, userID, export.UserID)
	assert.Len(t, export.Memberships.Orgs, 1)
	assert.Len(t, export.Memberships.Spaces, 1)
	assert.Len(t, export.Memberships.Boards, 1)
	assert.Len(t, export.Threads, 1)
	assert.Len(t, export.Messages, 1)
	assert.Len(t, export.Votes, 1)
	assert.Len(t, export.Notifications, 1)
	assert.Len(t, export.CallLogs, 1)
	assert.Len(t, export.Uploads, 1)
	assert.Len(t, export.AuditLogs, 1)
	assert.Len(t, export.Preferences, 1)
	assert.Len(t, export.Digests, 1)
}

func TestExportUserData_EmptyUser(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)

	export, err := svc.ExportUserData(context.Background(), "nonexistent_user")
	require.NoError(t, err)

	assert.Equal(t, "nonexistent_user", export.UserID)
	assert.Empty(t, export.Memberships.Orgs)
	assert.Empty(t, export.Threads)
	assert.Empty(t, export.Messages)
}

func TestExportUserDataJSON_EmptyUser(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)

	data, err := svc.ExportUserDataJSON(context.Background(), "nobody")
	require.NoError(t, err)

	var export UserExport
	require.NoError(t, json.Unmarshal(data, &export))
	assert.Equal(t, "nobody", export.UserID)
	assert.Empty(t, export.Threads)
	assert.Empty(t, export.Messages)
	assert.Empty(t, export.Votes)
	assert.Empty(t, export.Notifications)
	assert.Empty(t, export.CallLogs)
	assert.Empty(t, export.Uploads)
	assert.Empty(t, export.AuditLogs)
	assert.Empty(t, export.Preferences)
	assert.Empty(t, export.Digests)
	assert.Empty(t, export.Memberships.Orgs)
	assert.Empty(t, export.Memberships.Spaces)
	assert.Empty(t, export.Memberships.Boards)
}

func TestExportUserDataJSON_Success(t *testing.T) {
	db := testDB(t)
	userID := "user_json_export"
	seedUserData(t, db, userID)
	svc := NewService(db)

	data, err := svc.ExportUserDataJSON(context.Background(), userID)
	require.NoError(t, err)

	var export UserExport
	require.NoError(t, json.Unmarshal(data, &export))
	assert.Equal(t, userID, export.UserID)
	assert.Len(t, export.Threads, 1)
}

// --- PurgeUser Tests ---

func TestPurgeUser_RemovesAllData(t *testing.T) {
	db := testDB(t)
	userID := "user_purge_test"
	seedUserData(t, db, userID)
	svc := NewService(db)

	err := svc.PurgeUser(context.Background(), userID)
	require.NoError(t, err)

	// Verify memberships deleted.
	var orgMemberCount int64
	db.Model(&models.OrgMembership{}).Where("user_id = ?", userID).Count(&orgMemberCount)
	assert.Zero(t, orgMemberCount)

	var spaceMemberCount int64
	db.Model(&models.SpaceMembership{}).Where("user_id = ?", userID).Count(&spaceMemberCount)
	assert.Zero(t, spaceMemberCount)

	var boardMemberCount int64
	db.Model(&models.BoardMembership{}).Where("user_id = ?", userID).Count(&boardMemberCount)
	assert.Zero(t, boardMemberCount)

	// Verify content deleted.
	var threadCount int64
	db.Unscoped().Model(&models.Thread{}).Where("author_id = ?", userID).Count(&threadCount)
	assert.Zero(t, threadCount)

	var msgCount int64
	db.Unscoped().Model(&models.Message{}).Where("author_id = ?", userID).Count(&msgCount)
	assert.Zero(t, msgCount)

	var voteCount int64
	db.Model(&models.Vote{}).Where("user_id = ?", userID).Count(&voteCount)
	assert.Zero(t, voteCount)

	var notifCount int64
	db.Model(&models.Notification{}).Where("user_id = ?", userID).Count(&notifCount)
	assert.Zero(t, notifCount)

	var prefCount int64
	db.Model(&models.NotificationPreference{}).Where("user_id = ?", userID).Count(&prefCount)
	assert.Zero(t, prefCount)

	var digestCount int64
	db.Model(&models.DigestSchedule{}).Where("user_id = ?", userID).Count(&digestCount)
	assert.Zero(t, digestCount)

	var uploadCount int64
	db.Unscoped().Model(&models.Upload{}).Where("uploader_id = ?", userID).Count(&uploadCount)
	assert.Zero(t, uploadCount)

	var callLogCount int64
	db.Model(&models.CallLog{}).Where("caller_id = ?", userID).Count(&callLogCount)
	assert.Zero(t, callLogCount)
}

func TestPurgeUser_AnonymizesAuditLogs(t *testing.T) {
	db := testDB(t)
	userID := "user_audit_anon"
	seedUserData(t, db, userID)
	svc := NewService(db)

	err := svc.PurgeUser(context.Background(), userID)
	require.NoError(t, err)

	// Audit logs should be anonymized, not deleted.
	var audits []models.AuditLog
	db.Where("entity_type = ? AND action = ?", "org", models.AuditActionCreate).Find(&audits)
	require.NotEmpty(t, audits)

	for _, a := range audits {
		assert.Equal(t, "anonymized", a.UserID)
		assert.Empty(t, a.IPAddress)
	}
}

func TestPurgeUser_NonexistentUser(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)

	// Should not error — just no-ops.
	err := svc.PurgeUser(context.Background(), "nonexistent_user")
	require.NoError(t, err)
}

// --- PurgeOrg Tests ---

func TestPurgeOrg_CascadeDeletes(t *testing.T) {
	db := testDB(t)
	userID := "user_org_purge"
	org := seedUserData(t, db, userID)
	svc := NewService(db)

	err := svc.PurgeOrg(context.Background(), org.ID)
	require.NoError(t, err)

	// Verify org is gone.
	var orgCount int64
	db.Unscoped().Model(&models.Org{}).Where("id = ?", org.ID).Count(&orgCount)
	assert.Zero(t, orgCount)

	// Verify all children are gone.
	var spaceCount int64
	db.Unscoped().Model(&models.Space{}).Where("org_id = ?", org.ID).Count(&spaceCount)
	assert.Zero(t, spaceCount)

	var boardCount int64
	db.Unscoped().Model(&models.Board{}).Count(&boardCount)
	assert.Zero(t, boardCount)

	var threadCount int64
	db.Unscoped().Model(&models.Thread{}).Count(&threadCount)
	assert.Zero(t, threadCount)

	var msgCount int64
	db.Unscoped().Model(&models.Message{}).Count(&msgCount)
	assert.Zero(t, msgCount)
}

func TestPurgeOrg_NonexistentOrg(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)

	err := svc.PurgeOrg(context.Background(), "nonexistent_org")
	require.NoError(t, err)
}

// --- Handler Tests ---

func TestHandler_ExportUserData_Success(t *testing.T) {
	db := testDB(t)
	userID := "user_handler_export"
	seedUserData(t, db, userID)
	handler := NewHandler(NewService(db))

	r := chi.NewRouter()
	r.Get("/v1/admin/users/{user}/export", handler.ExportUserData)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users/"+userID+"/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), userID)

	var export UserExport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &export))
	assert.Equal(t, userID, export.UserID)
}

func TestHandler_PurgeUser_Success(t *testing.T) {
	db := testDB(t)
	userID := "user_handler_purge"
	seedUserData(t, db, userID)
	handler := NewHandler(NewService(db))

	r := chi.NewRouter()
	r.Delete("/v1/admin/users/{user}/purge", handler.PurgeUser)

	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/users/"+userID+"/purge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "purged", resp["status"])
}

func TestHandler_PurgeOrg_Success(t *testing.T) {
	db := testDB(t)
	userID := "user_handler_org_purge"
	org := seedUserData(t, db, userID)
	handler := NewHandler(NewService(db))

	r := chi.NewRouter()
	r.Delete("/v1/admin/orgs/{org}/purge", handler.PurgeOrg)

	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/"+org.ID+"/purge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "purged", resp["status"])
}

// --- Handler Edge Case Tests ---

func TestHandler_ExportUserData_EmptyUserParam(t *testing.T) {
	db := testDB(t)
	handler := NewHandler(NewService(db))

	// Call handler directly without chi router so URLParam returns empty.
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users//export", nil)
	w := httptest.NewRecorder()
	handler.ExportUserData(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeUser_EmptyUserParam(t *testing.T) {
	db := testDB(t)
	handler := NewHandler(NewService(db))

	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/users//purge", nil)
	w := httptest.NewRecorder()
	handler.PurgeUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg_EmptyOrgParam(t *testing.T) {
	db := testDB(t)
	handler := NewHandler(NewService(db))

	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs//purge", nil)
	w := httptest.NewRecorder()
	handler.PurgeOrg(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPurgeOrg_WithFullHierarchy(t *testing.T) {
	db := testDB(t)
	userID := "user_full_purge"
	org := seedUserData(t, db, userID)

	// Add a revision to ensure that path is covered.
	var thread models.Thread
	require.NoError(t, db.First(&thread, "author_id = ?", userID).Error)
	rev := &models.Revision{
		EntityType:      "thread",
		EntityID:        thread.ID,
		EditorID:        userID,
		PreviousContent: "original body",
	}
	require.NoError(t, db.Create(rev).Error)

	// Also add an API key and webhook for the org.
	apiKey := &models.APIKey{
		OrgID:       org.ID,
		Name:        "Test Key",
		KeyHash:     "hash",
		KeyPrefix:   "deft_",
		Permissions: "{}",
	}
	require.NoError(t, db.Create(apiKey).Error)

	wh := &models.WebhookSubscription{
		OrgID:       org.ID,
		ScopeType:   "org",
		ScopeID:     org.ID,
		URL:         "https://example.com/hook",
		EventFilter: "[\"thread.created\"]",
		Secret:      "secret",
	}
	require.NoError(t, db.Create(wh).Error)

	svc := NewService(db)
	err := svc.PurgeOrg(context.Background(), org.ID)
	require.NoError(t, err)

	// Verify everything is gone.
	var orgCount int64
	db.Unscoped().Model(&models.Org{}).Where("id = ?", org.ID).Count(&orgCount)
	assert.Zero(t, orgCount)

	var revCount int64
	db.Model(&models.Revision{}).Where("entity_id = ?", thread.ID).Count(&revCount)
	assert.Zero(t, revCount)

	var keyCount int64
	db.Model(&models.APIKey{}).Where("org_id = ?", org.ID).Count(&keyCount)
	assert.Zero(t, keyCount)

	var whCount int64
	db.Model(&models.WebhookSubscription{}).Where("org_id = ?", org.ID).Count(&whCount)
	assert.Zero(t, whCount)
}

func TestNewService_ReturnsNonNil(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	assert.NotNil(t, svc)
}

func TestNewHandler_ReturnsNonNil(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewService(db))
	assert.NotNil(t, h)
}

// --- Service Error Path Tests ---

func closedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	return db
}

func TestExportUserData_DBError(t *testing.T) {
	db := closedDB(t)
	svc := NewService(db)

	_, err := svc.ExportUserData(context.Background(), "user-err")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exporting org memberships")
}

func TestExportUserData_PartialDBErrors(t *testing.T) {
	// Test with specific tables dropped to hit later error branches.
	tables := []struct {
		table   string
		errText string
	}{
		{"space_memberships", "exporting space memberships"},
		{"board_memberships", "exporting board memberships"},
		{"threads", "exporting threads"},
		{"messages", "exporting messages"},
		{"votes", "exporting votes"},
		{"notifications", "exporting notifications"},
		{"call_logs", "exporting call logs"},
		{"uploads", "exporting uploads"},
		{"audit_logs", "exporting audit logs"},
		{"notification_preferences", "exporting preferences"},
		{"digest_schedules", "exporting digests"},
	}

	for _, tc := range tables {
		t.Run(tc.table, func(t *testing.T) {
			db := testDB(t)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			_, err = sqlDB.Exec("DROP TABLE IF EXISTS " + tc.table)
			require.NoError(t, err)

			svc := NewService(db)
			_, err = svc.ExportUserData(context.Background(), "user-err")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestPurgeUser_PartialDBErrors(t *testing.T) {
	tables := []struct {
		table   string
		errText string
	}{
		{"org_memberships", "purging org memberships"},
		{"space_memberships", "purging space memberships"},
		{"board_memberships", "purging board memberships"},
		{"votes", "purging votes"},
		{"notifications", "purging notifications"},
		{"notification_preferences", "purging notification preferences"},
		{"digest_schedules", "purging digest schedules"},
		{"uploads", "purging uploads"},
		{"call_logs", "purging call logs"},
	}

	for _, tc := range tables {
		t.Run(tc.table, func(t *testing.T) {
			db := testDB(t)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			_, err = sqlDB.Exec("DROP TABLE IF EXISTS " + tc.table)
			require.NoError(t, err)

			svc := NewService(db)
			err = svc.PurgeUser(context.Background(), "user-err")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestExportUserDataJSON_DBError(t *testing.T) {
	db := closedDB(t)
	svc := NewService(db)

	_, err := svc.ExportUserDataJSON(context.Background(), "user-err")
	require.Error(t, err)
}

func TestPurgeUser_DBError(t *testing.T) {
	db := closedDB(t)
	svc := NewService(db)

	err := svc.PurgeUser(context.Background(), "user-err")
	require.Error(t, err)
}

func TestPurgeOrg_DBError(t *testing.T) {
	db := closedDB(t)
	svc := NewService(db)

	err := svc.PurgeOrg(context.Background(), "org-err")
	require.Error(t, err)
}

func TestHandler_ExportUserData_ServiceError(t *testing.T) {
	db := closedDB(t)
	handler := NewHandler(NewService(db))

	r := chi.NewRouter()
	r.Get("/v1/admin/users/{user}/export", handler.ExportUserData)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users/user-err/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PurgeUser_ServiceError(t *testing.T) {
	db := closedDB(t)
	handler := NewHandler(NewService(db))

	r := chi.NewRouter()
	r.Delete("/v1/admin/users/{user}/purge", handler.PurgeUser)

	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/users/user-err/purge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_PurgeOrg_ServiceError(t *testing.T) {
	db := closedDB(t)
	handler := NewHandler(NewService(db))

	r := chi.NewRouter()
	r.Delete("/v1/admin/orgs/{org}/purge", handler.PurgeOrg)

	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/org-err/purge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Fuzz Tests ---

func FuzzPurgeUserInput(f *testing.F) {
	f.Add("user_123")
	f.Add("")
	f.Add("user_with_<special>&chars")
	f.Add("a")
	f.Add("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")

	db := func() *gorm.DB {
		dir := f.TempDir()
		dbPath := filepath.Join(dir, "fuzz.db")
		db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			f.Fatal(err)
		}
		if err := database.Migrate(db); err != nil {
			f.Fatal(err)
		}
		return db
	}()

	svc := NewService(db)

	f.Fuzz(func(t *testing.T, userID string) {
		// Should not panic regardless of input.
		_ = svc.PurgeUser(context.Background(), userID)
	})
}
