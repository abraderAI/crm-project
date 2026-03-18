package conversion

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/seed"
)

func fuzzSetupDB(t *testing.T) *gorm.DB {
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
	require.NoError(t, seed.Run(db))
	// Close the DB before t.TempDir() cleanup removes the directory.
	// Cleanup functions run LIFO, so registering after TempDir ensures this runs first.
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// FuzzGenerateSlug fuzzes the internal slug generation function with random org names.
func FuzzGenerateSlug(f *testing.F) {
	seeds := []string{
		"My Organization",
		"Acme Corp",
		"",
		"a",
		"123",
		"Test-Org",
		"test_org",
		"TEST ORG",
		"org with spaces",
		"org-with-hyphens",
		"org_with_underscores",
		"123 Numeric Start",
		"!@#$%^&*()",
		"org\x00null",
		"org\nnewline",
		"   ",
		"  leading  ",
		"trailing  ",
		"αβγδ",
		"кириллица",
		"中文",
		"العربية",
		"org 🎉",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"a--b",
		"a___b",
		"---",
		"___",
		"a-b-c",
		"A B C D E F G",
		"mix OF lower AND UPPER",
		"company.name",
		"company/dept",
		"company\\dept",
		"org: subtitle",
		"org (2024)",
		"org [beta]",
		"org {internal}",
		"org #1",
		"org @mention",
		"org $money",
		"org %percent",
		"org+plus",
		"org=equals",
		"org|pipe",
		"org~tilde",
		"org`backtick",
		"org;semicolon",
		"org:colon",
		"null",
		"undefined",
		"true",
		"false",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		slug := generateSlug(name)
		// Slug should never be empty (falls back to "org").
		if slug == "" {
			t.Errorf("generateSlug(%q) returned empty string", name)
		}
		// Slug should not have a trailing hyphen.
		if len(slug) > 0 && slug[len(slug)-1] == '-' {
			t.Errorf("generateSlug(%q) returned slug with trailing hyphen: %q", name, slug)
		}
	})
}

// FuzzSelfServiceUpgrade fuzzes the self-service upgrade flow with random org names.
func FuzzSelfServiceUpgrade(f *testing.F) {
	seeds := []string{
		"My Org",
		"Acme Corp",
		"",
		"a",
		"A",
		"123",
		"Org with Spaces",
		"org-with-hyphens",
		"ORG UPPERCASE",
		"org\x00null",
		"org\nnewline",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"αβγδ org",
		"org 🎉",
		"中文组织",
		"кириллица",
		"   ",
		"   spaces   ",
		"a--b",
		"org.com",
		"org/dept",
		"org\\dept",
		"org: name",
		"org (v2)",
		"null",
		"undefined",
		"true",
		"!@#$%^&*()",
		"very long organization name that exceeds what might be expected for normal usage",
		"short",
		"org1",
		"org2",
		"test org",
		"my company",
		"startup name",
		"enterprise corp",
		"small business llc",
		"venture capital",
		"tech startup inc",
		"software company",
		"digital agency",
		"consulting firm",
		"nonprofit org",
		"research lab",
		"university dept",
		"government agency",
		"foundation",
		"association",
		"cooperative",
		"trust",
		"fund",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, orgName string) {
		db := fuzzSetupDB(t)
		svc := NewService(db)
		ctx := context.Background()
		// Seed a user for the upgrade attempt.
		shadow := &models.UserShadow{
			ClerkUserID: "fuzz-upgrade-user",
			Email:       "fuzz@example.com",
			DisplayName: "Fuzz User",
		}
		_ = db.Create(shadow).Error
		// Should never panic regardless of org name input.
		_, _ = svc.SelfServiceUpgrade(ctx, "fuzz-upgrade-user", orgName)
	})
}
