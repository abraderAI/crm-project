package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// allModels returns the ordered list of models for AutoMigrate.
func allModels() []any {
	return []any{
		&models.Org{},
		&models.Space{},
		&models.Board{},
		&models.Thread{},
		&models.Message{},
		&models.OrgMembership{},
		&models.SpaceMembership{},
		&models.BoardMembership{},
		&models.APIKey{},
		&models.AuditLog{},
		&models.Revision{},
		&models.WebhookSubscription{},
		&models.WebhookDelivery{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.DigestSchedule{},
		&models.Vote{},
		&models.Upload{},
		&models.Flag{},
		&models.CallLog{},
		&models.PlatformAdmin{},
		&models.UserShadow{},
		&models.SystemSetting{},
		&models.FeatureFlag{},
		&models.AdminExport{},
		&models.APIUsageStat{},
		&models.LoginEvent{},
		&models.FailedAuth{},
		&models.LLMUsageLog{},
	}
}

// Migrate runs AutoMigrate for all models, creates generated columns,
// FTS5 virtual tables with sync triggers, and additional indexes.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(allModels()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	if err := createGeneratedColumns(db); err != nil {
		return fmt.Errorf("generated columns: %w", err)
	}

	if err := createFTS5Tables(db); err != nil {
		return fmt.Errorf("fts5 tables: %w", err)
	}

	if err := createIndexes(db); err != nil {
		return fmt.Errorf("indexes: %w", err)
	}

	return nil
}

// createGeneratedColumns adds SQLite generated columns extracted from
// Metadata JSON fields for fast querying and indexing.
func createGeneratedColumns(db *gorm.DB) error {
	// Org generated columns.
	orgColumns := []struct {
		name string
		expr string
	}{
		{"billing_tier", `json_extract(metadata, '$.billing_tier')`},
		{"payment_status", `json_extract(metadata, '$.payment_status')`},
	}
	for _, col := range orgColumns {
		if err := addGeneratedColumnIfNotExists(db, "orgs", col.name, col.expr); err != nil {
			return err
		}
	}

	// Thread generated columns.
	threadColumns := []struct {
		name string
		expr string
	}{
		{"status", `json_extract(metadata, '$.status')`},
		{"priority", `json_extract(metadata, '$.priority')`},
		{"stage", `json_extract(metadata, '$.stage')`},
		{"assigned_to", `json_extract(metadata, '$.assigned_to')`},
	}
	for _, col := range threadColumns {
		if err := addGeneratedColumnIfNotExists(db, "threads", col.name, col.expr); err != nil {
			return err
		}
	}

	return nil
}

// addGeneratedColumnIfNotExists adds a generated column to a table if it does not already exist.
// Uses PRAGMA table_xinfo which includes hidden and generated columns.
func addGeneratedColumnIfNotExists(db *gorm.DB, table, column, expr string) error {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_xinfo('%s') WHERE name = '%s'", table, column)
	row := db.Raw(query).Row()
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("checking column %s.%s: %w", table, column, err)
	}
	if count > 0 {
		return nil
	}
	addSQL := fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN %s TEXT GENERATED ALWAYS AS (%s) STORED",
		table, column, expr,
	)
	if err := db.Exec(addSQL).Error; err != nil {
		return fmt.Errorf("adding generated column %s.%s: %w", table, column, err)
	}
	return nil
}

// createFTS5Tables creates FTS5 virtual tables and sync triggers for full-text search.
func createFTS5Tables(db *gorm.DB) error {
	fts5Defs := []struct {
		table   string
		source  string
		columns string
	}{
		{"orgs_fts", "orgs", "name, slug, description"},
		{"spaces_fts", "spaces", "name, slug, description"},
		{"boards_fts", "boards", "name, slug, description"},
		{"threads_fts", "threads", "title, body, slug"},
		{"messages_fts", "messages", "body"},
	}

	for _, def := range fts5Defs {
		// Create FTS5 virtual table using content= and content_rowid= for external content.
		createSQL := fmt.Sprintf(
			"CREATE VIRTUAL TABLE IF NOT EXISTS %s USING fts5(%s, content='%s', content_rowid='rowid')",
			def.table, def.columns, def.source,
		)
		if err := db.Exec(createSQL).Error; err != nil {
			return fmt.Errorf("creating FTS5 table %s: %w", def.table, err)
		}

		// Create sync triggers.
		if err := createFTS5Triggers(db, def.table, def.source, def.columns); err != nil {
			return fmt.Errorf("creating triggers for %s: %w", def.table, err)
		}
	}

	return nil
}

// createFTS5Triggers creates INSERT/UPDATE/DELETE triggers to keep FTS5 in sync.
func createFTS5Triggers(db *gorm.DB, ftsTable, sourceTable, columns string) error {
	triggers := []string{
		// After INSERT: add to FTS.
		fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %s_ai AFTER INSERT ON %s BEGIN
			INSERT INTO %s(rowid, %s) VALUES (new.rowid, %s);
		END`,
			ftsTable, sourceTable,
			ftsTable, columns, prefixColumns("new", columns),
		),
		// After UPDATE: remove old, add new.
		fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %s_au AFTER UPDATE ON %s BEGIN
			INSERT INTO %s(%s, rowid, %s) VALUES ('delete', old.rowid, %s);
			INSERT INTO %s(rowid, %s) VALUES (new.rowid, %s);
		END`,
			ftsTable, sourceTable,
			ftsTable, ftsTable, columns, prefixColumns("old", columns),
			ftsTable, columns, prefixColumns("new", columns),
		),
		// After DELETE: remove from FTS.
		fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %s_ad AFTER DELETE ON %s BEGIN
			INSERT INTO %s(%s, rowid, %s) VALUES ('delete', old.rowid, %s);
		END`,
			ftsTable, sourceTable,
			ftsTable, ftsTable, columns, prefixColumns("old", columns),
		),
	}

	for _, trigger := range triggers {
		if err := db.Exec(trigger).Error; err != nil {
			return fmt.Errorf("creating trigger: %w", err)
		}
	}

	return nil
}

// prefixColumns takes "new" or "old" and a comma-separated column list,
// returning "prefix.col1, prefix.col2, ...".
func prefixColumns(prefix, columns string) string {
	result := ""
	inWord := false
	wordStart := 0
	for i := 0; i <= len(columns); i++ {
		if i == len(columns) || columns[i] == ',' {
			if inWord {
				col := columns[wordStart:i]
				// Trim spaces.
				for len(col) > 0 && col[0] == ' ' {
					col = col[1:]
				}
				for len(col) > 0 && col[len(col)-1] == ' ' {
					col = col[:len(col)-1]
				}
				if result != "" {
					result += ", "
				}
				result += prefix + "." + col
			}
			inWord = false
		} else if !inWord {
			inWord = true
			wordStart = i
		}
	}
	return result
}

// createIndexes creates additional indexes for common query patterns.
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		// Slug uniqueness within parent (composite indexes).
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_spaces_org_slug ON spaces(org_id, slug) WHERE deleted_at IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_boards_space_slug ON boards(space_id, slug) WHERE deleted_at IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_threads_board_slug ON threads(board_id, slug) WHERE deleted_at IS NULL",

		// Generated column indexes for Thread.
		"CREATE INDEX IF NOT EXISTS idx_threads_status ON threads(status) WHERE status IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_threads_priority ON threads(priority) WHERE priority IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_threads_stage ON threads(stage) WHERE stage IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_threads_assigned_to ON threads(assigned_to) WHERE assigned_to IS NOT NULL",

		// Generated column indexes for Org.
		"CREATE INDEX IF NOT EXISTS idx_orgs_billing_tier ON orgs(billing_tier) WHERE billing_tier IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_orgs_payment_status ON orgs(payment_status) WHERE payment_status IS NOT NULL",

		// Webhook delivery indexes.
		"CREATE INDEX IF NOT EXISTS idx_webhook_delivery_next_retry ON webhook_deliveries(next_retry_at) WHERE next_retry_at IS NOT NULL",

		// Notification indexes.
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read)",
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}
