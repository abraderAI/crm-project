package database

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testMigratedDB creates a fresh SQLite DB with all migrations applied.
func testMigratedDB(t *testing.T) *gorm.DB {
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
	require.NoError(t, Migrate(db))
	return db
}

func TestMigrate_CreatesAllTables(t *testing.T) {
	db := testMigratedDB(t)

	expectedTables := []string{
		"orgs", "spaces", "boards", "threads", "messages",
		"org_memberships", "space_memberships", "board_memberships",
		"api_keys", "audit_logs", "revisions",
		"webhook_subscriptions", "webhook_deliveries",
		"notifications", "notification_preferences", "digest_schedules",
		"votes", "uploads",
	}

	for _, table := range expectedTables {
		var count int64
		row := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Row()
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, int64(1), count, "table %s should exist", table)
	}
}

func TestMigrate_CreatesFTS5Tables(t *testing.T) {
	db := testMigratedDB(t)

	fts5Tables := []string{
		"orgs_fts", "spaces_fts", "boards_fts", "threads_fts", "messages_fts",
	}

	for _, table := range fts5Tables {
		var count int64
		row := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Row()
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, int64(1), count, "FTS5 table %s should exist", table)
	}
}

func TestMigrate_CreatesTriggers(t *testing.T) {
	db := testMigratedDB(t)

	// Each FTS table should have 3 triggers (ai, au, ad).
	ftsTables := []string{"orgs_fts", "spaces_fts", "boards_fts", "threads_fts", "messages_fts"}
	for _, fts := range ftsTables {
		for _, suffix := range []string{"_ai", "_au", "_ad"} {
			triggerName := fts + suffix
			var count int64
			row := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?", triggerName).Row()
			require.NoError(t, row.Scan(&count))
			assert.Equal(t, int64(1), count, "trigger %s should exist", triggerName)
		}
	}
}

func TestMigrate_OrgGeneratedColumns(t *testing.T) {
	db := testMigratedDB(t)

	// Insert an org with metadata and verify generated columns.
	require.NoError(t, db.Exec(
		`INSERT INTO orgs (id, name, slug, metadata, created_at, updated_at) 
		 VALUES ('test-1', 'Test', 'test-gen', '{"billing_tier":"enterprise","payment_status":"paid"}', datetime('now'), datetime('now'))`,
	).Error)

	var row struct {
		BillingTier   string
		PaymentStatus string
	}
	require.NoError(t, db.Raw("SELECT billing_tier, payment_status FROM orgs WHERE id = 'test-1'").Scan(&row).Error)
	assert.Equal(t, "enterprise", row.BillingTier)
	assert.Equal(t, "paid", row.PaymentStatus)
}

func TestMigrate_ThreadGeneratedColumns(t *testing.T) {
	db := testMigratedDB(t)

	// Create hierarchy for FK constraints.
	require.NoError(t, db.Exec(
		`INSERT INTO orgs (id, name, slug, metadata, created_at, updated_at) VALUES ('o1', 'O', 'gen-col-org', '{}', datetime('now'), datetime('now'))`,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO spaces (id, org_id, name, slug, type, metadata, created_at, updated_at) VALUES ('s1', 'o1', 'S', 'gen-col-space', 'general', '{}', datetime('now'), datetime('now'))`,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO boards (id, space_id, name, slug, metadata, created_at, updated_at) VALUES ('b1', 's1', 'B', 'gen-col-board', '{}', datetime('now'), datetime('now'))`,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO threads (id, board_id, title, slug, author_id, metadata, created_at, updated_at) 
		 VALUES ('t1', 'b1', 'T', 'gen-col-thread', 'u1', '{"status":"in_progress","priority":"2","stage":"negotiation","assigned_to":"u3"}', datetime('now'), datetime('now'))`,
	).Error)

	var row struct {
		Status     string
		Priority   string
		Stage      string
		AssignedTo string
	}
	require.NoError(t, db.Raw("SELECT status, priority, stage, assigned_to FROM threads WHERE id = 't1'").Scan(&row).Error)
	assert.Equal(t, "in_progress", row.Status)
	assert.Equal(t, "2", row.Priority)
	assert.Equal(t, "negotiation", row.Stage)
	assert.Equal(t, "u3", row.AssignedTo)
}

func TestMigrate_FTS5InsertSync(t *testing.T) {
	db := testMigratedDB(t)

	// Insert org.
	require.NoError(t, db.Exec(
		`INSERT INTO orgs (id, name, slug, description, metadata, created_at, updated_at)
		 VALUES ('fts-1', 'Acme Corp', 'acme-corp', 'A great company', '{}', datetime('now'), datetime('now'))`,
	).Error)

	// Search FTS.
	var count int64
	row := db.Raw("SELECT COUNT(*) FROM orgs_fts WHERE orgs_fts MATCH 'acme'").Row()
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(1), count)
}

func TestMigrate_FTS5UpdateSync(t *testing.T) {
	db := testMigratedDB(t)

	require.NoError(t, db.Exec(
		`INSERT INTO orgs (id, name, slug, description, metadata, created_at, updated_at)
		 VALUES ('fts-2', 'OldName', 'old-name', 'desc', '{}', datetime('now'), datetime('now'))`,
	).Error)

	// Update org name.
	require.NoError(t, db.Exec("UPDATE orgs SET name = 'NewName' WHERE id = 'fts-2'").Error)

	// Search for new name.
	var count int64
	row := db.Raw("SELECT COUNT(*) FROM orgs_fts WHERE orgs_fts MATCH 'NewName'").Row()
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(1), count)

	// Old name should not match.
	var oldCount int64
	row2 := db.Raw("SELECT COUNT(*) FROM orgs_fts WHERE orgs_fts MATCH 'OldName'").Row()
	require.NoError(t, row2.Scan(&oldCount))
	assert.Equal(t, int64(0), oldCount)
}

func TestMigrate_FTS5DeleteSync(t *testing.T) {
	db := testMigratedDB(t)

	require.NoError(t, db.Exec(
		`INSERT INTO orgs (id, name, slug, description, metadata, created_at, updated_at)
		 VALUES ('fts-3', 'DeleteMe', 'delete-me', 'desc', '{}', datetime('now'), datetime('now'))`,
	).Error)

	// Delete org.
	require.NoError(t, db.Exec("DELETE FROM orgs WHERE id = 'fts-3'").Error)

	// Should not appear in search.
	var count int64
	row := db.Raw("SELECT COUNT(*) FROM orgs_fts WHERE orgs_fts MATCH 'DeleteMe'").Row()
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(0), count)
}

func TestMigrate_FTS5ThreadsSearch(t *testing.T) {
	db := testMigratedDB(t)

	require.NoError(t, db.Exec(
		`INSERT INTO orgs (id, name, slug, metadata, created_at, updated_at) VALUES ('ft-o1', 'O', 'fts-thread-org', '{}', datetime('now'), datetime('now'))`,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO spaces (id, org_id, name, slug, type, metadata, created_at, updated_at) VALUES ('ft-s1', 'ft-o1', 'S', 'fts-thread-space', 'general', '{}', datetime('now'), datetime('now'))`,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO boards (id, space_id, name, slug, metadata, created_at, updated_at) VALUES ('ft-b1', 'ft-s1', 'B', 'fts-thread-board', '{}', datetime('now'), datetime('now'))`,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO threads (id, board_id, title, body, slug, author_id, metadata, created_at, updated_at)
		 VALUES ('ft-t1', 'ft-b1', 'Bug Report', 'Application crashes on startup', 'bug-report', 'u1', '{}', datetime('now'), datetime('now'))`,
	).Error)

	var count int64
	row := db.Raw("SELECT COUNT(*) FROM threads_fts WHERE threads_fts MATCH 'crashes'").Row()
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(1), count)
}

func TestMigrate_FTS5MessagesSearch(t *testing.T) {
	db := testMigratedDB(t)

	require.NoError(t, db.Exec(`INSERT INTO orgs (id, name, slug, metadata, created_at, updated_at) VALUES ('fm-o1', 'O', 'fts-msg-org', '{}', datetime('now'), datetime('now'))`).Error)
	require.NoError(t, db.Exec(`INSERT INTO spaces (id, org_id, name, slug, type, metadata, created_at, updated_at) VALUES ('fm-s1', 'fm-o1', 'S', 'fts-msg-space', 'general', '{}', datetime('now'), datetime('now'))`).Error)
	require.NoError(t, db.Exec(`INSERT INTO boards (id, space_id, name, slug, metadata, created_at, updated_at) VALUES ('fm-b1', 'fm-s1', 'B', 'fts-msg-board', '{}', datetime('now'), datetime('now'))`).Error)
	require.NoError(t, db.Exec(`INSERT INTO threads (id, board_id, title, slug, author_id, metadata, created_at, updated_at) VALUES ('fm-t1', 'fm-b1', 'T', 'fts-msg-thread', 'u1', '{}', datetime('now'), datetime('now'))`).Error)
	require.NoError(t, db.Exec(`INSERT INTO messages (id, thread_id, body, author_id, type, metadata, created_at, updated_at) VALUES ('fm-m1', 'fm-t1', 'This is a detailed analysis of performance', 'u1', 'comment', '{}', datetime('now'), datetime('now'))`).Error)

	var count int64
	row := db.Raw("SELECT COUNT(*) FROM messages_fts WHERE messages_fts MATCH 'performance'").Row()
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(1), count)
}

func TestMigrate_Idempotent(t *testing.T) {
	db := testMigratedDB(t)
	// Running migrate again should not fail.
	require.NoError(t, Migrate(db))
}

func TestMigrate_Indexes(t *testing.T) {
	db := testMigratedDB(t)

	expectedIndexes := []string{
		"idx_spaces_org_slug",
		"idx_boards_space_slug",
		"idx_threads_board_slug",
		"idx_threads_status",
		"idx_threads_priority",
		"idx_threads_stage",
		"idx_threads_assigned_to",
		"idx_orgs_billing_tier",
		"idx_orgs_payment_status",
		"idx_webhook_delivery_next_retry",
		"idx_notifications_user_read",
	}

	for _, idx := range expectedIndexes {
		var count int64
		row := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Row()
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, int64(1), count, "index %s should exist", idx)
	}
}

func TestPrefixColumns(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		columns string
		want    string
	}{
		{"single", "new", "name", "new.name"},
		{"multiple", "old", "name, slug, description", "old.name, old.slug, old.description"},
		{"extra spaces", "new", " name , slug ", "new.name, new.slug"},
		{"single body", "new", "body", "new.body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prefixColumns(tt.prefix, tt.columns)
			assert.Equal(t, tt.want, got)
		})
	}
}
