package admin

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Phase C Fuzz Tests ---

func FuzzImpersonationToken(f *testing.F) {
	f.Add("valid.token")
	f.Add("")
	f.Add("dGVzdA.dGVzdA")
	f.Add("no-dot-token")
	f.Add(strings.Repeat("a", 5000))
	f.Add("ab.cd.ef.gh")
	f.Add("eyJ0ZXN0IjoidmFsdWUifQ.AAAA")
	f.Add("payload with spaces.signature with spaces")
	f.Add("<script>alert(1)</script>.<img src=x>")
	f.Add("unicode-日本語.テスト")

	f.Fuzz(func(t *testing.T, token string) {
		// Must not panic for any input.
		_, _ = ValidateImpersonationToken(token)
	})
}

func FuzzExportFilters(f *testing.F) {
	f.Add("{}")
	f.Add("")
	f.Add("{invalid json}")
	f.Add(`{"key":"value"}`)
	f.Add(strings.Repeat("{", 1000))
	f.Add(`[1,2,3]`)
	f.Add(`{"nested":{"deep":{"value":true}}}`)
	f.Add("null")
	f.Add("<script>alert(1)</script>")
	f.Add("unicode: 日本語テスト")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.AdminExport{}, &models.UserShadow{}, &models.Org{}, &models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, filters string) {
		// Must not panic.
		_, _ = svc.CreateExport(ctx, "users", "csv", filters, "admin1", nil)
	})
}

func FuzzUsageQueryParams(f *testing.F) {
	f.Add("24h")
	f.Add("7d")
	f.Add("30d")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("x", 5000))
	f.Add("1h")
	f.Add("-1d")
	f.Add("<script>")
	f.Add("日本語")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.APIUsageStat{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, period string) {
		// Must not panic.
		_, _ = svc.GetAPIUsage(ctx, period)
	})
}

func FuzzExportType(f *testing.F) {
	f.Add("users")
	f.Add("orgs")
	f.Add("audit")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("x", 5000))
	f.Add("<script>alert(1)</script>")
	f.Add("USERS")
	f.Add("日本語")
	f.Add("users; DROP TABLE--")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.AdminExport{}, &models.UserShadow{}, &models.Org{}, &models.AuditLog{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, exportType string) {
		// Must not panic.
		_, _ = svc.CreateExport(ctx, exportType, "csv", "{}", "admin1", nil)
	})
}

func FuzzFailedAuthPeriod(f *testing.F) {
	f.Add("24h")
	f.Add("7d")
	f.Add("")
	f.Add("invalid")
	f.Add(strings.Repeat("a", 5000))
	f.Add("-1d")
	f.Add("1000d")
	f.Add("<script>")
	f.Add("日本語")
	f.Add("'; DROP TABLE--")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.FailedAuth{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, period string) {
		// Must not panic.
		_, _ = svc.GetFailedAuths(ctx, period)
	})
}

func FuzzImpersonationCreate(f *testing.F) {
	f.Add("admin1", "user1", "reason", 30)
	f.Add("", "", "", 0)
	f.Add("admin", "admin", "self", -1)
	f.Add(strings.Repeat("a", 500), "target", "long admin id", 999)
	f.Add("admin1", "target", "<script>alert(1)</script>", 120)
	f.Add("admin1", "target", "unicode: 日本語", 60)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.PlatformAdmin{}, &models.UserShadow{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, impersonator, target, reason string, duration int) {
		// Must not panic.
		_, _, _ = svc.ImpersonateUser(ctx, impersonator, target, reason, duration)
	})
}
