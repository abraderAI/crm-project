package tier

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
)

func fuzzDB(t *testing.T) *gorm.DB {
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
	// Close the DB before t.TempDir() cleanup removes the directory.
	// Cleanup functions run LIFO, so registering after TempDir ensures this runs first.
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// FuzzResolveTier fuzzes tier resolution with arbitrary user IDs.
func FuzzResolveTier(f *testing.F) {
	seeds := []string{
		"",
		"user_abc123",
		"user_registered",
		"user_customer",
		"user_admin",
		"user_owner",
		"user_deft",
		"platform_admin",
		"a",
		"nonexistent",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"user\x00null",
		"user\nnewline",
		"user\ttab",
		"123numeric",
		"UPPERCASE",
		"MiXeD",
		"user-hyphens",
		"user_underscores",
		"αβγ",
		"кирилл",
		"中文",
		"user 🎉",
		"null",
		"undefined",
		"true",
		"false",
		"0",
		"-1",
		"   ",
		"user/slash",
		"user\\backslash",
		"user#hash",
		"user?query",
		"user.email@domain.com",
		"clerk_user_abc123",
		"clerk_user_xyz789",
		"user_000000000000",
		"very-long-user-id-exceeding-normal-length-for-clerk-user-ids",
		"user!@#$%",
		"user+plus",
		"user%pct",
		"user*star",
		"user(paren",
		"user)close",
		"user[bracket",
		"user]close",
		"user|pipe",
		"user~tilde",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, userID string) {
		db := fuzzDB(t)
		svc := NewService(NewRepository(db))
		// Should never panic for any user ID.
		_, _ = svc.ResolveTier(userID)
	})
}

// FuzzResolveDeftDepartment fuzzes the department resolution with arbitrary space slugs.
func FuzzResolveDeftDepartment(f *testing.F) {
	seeds := []string{
		"deft-sales",
		"deft-support",
		"deft-finance",
		"deft-engineering",
		"deft-marketing",
		"",
		"a",
		"-sales",
		"-support",
		"-finance",
		"sales",
		"support",
		"finance",
		"deft",
		"deft-",
		"deft-unknown",
		"DEFT-SALES",
		"Deft-Sales",
		"deft_sales",
		"deft sales",
		"dept-sales",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"slug\x00null",
		"slug\nnewline",
		"slug-with-dashes",
		"slug_underscores",
		"123",
		"null",
		"true",
		"αβγ-sales",
		"кирилл-support",
		"中文-finance",
		"slug 🎉",
		"   ",
		"slug/slash",
		"slug\\backslash",
		"very-long-space-slug-exceeding-normal-expected-length-for-testing-purposes",
		"deft-sales-extra",
		"prefix-deft-sales",
		"-",
		"--",
		"---",
		"deft--sales",
		"deft-sales-",
		"-deft-sales",
		"deft-sales-support",
		"multi-suffix-sales",
		"multi-suffix-support",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, spaceSlug string) {
		// Pure function — no DB needed. Should never panic.
		_, _ = resolveDeftDepartment(spaceSlug)
	})
}
