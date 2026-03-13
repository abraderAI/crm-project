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

// --- Fuzz Tests ---

func FuzzBanUserReason(f *testing.F) {
	f.Add("spam")
	f.Add("")
	f.Add("a very long reason " + strings.Repeat("x", 1000))
	f.Add("<script>alert('xss')</script>")
	f.Add("reason with\nnewlines\tand\ttabs")
	f.Add("emoji 🚫 reason")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.UserShadow{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, reason string) {
		// Should not panic for any input.
		_ = svc.BanUser(ctx, "fuzz_user", reason, "admin")
		_ = svc.UnbanUser(ctx, "fuzz_user")
	})
}

func FuzzSuspendOrgReason(f *testing.F) {
	f.Add("violation")
	f.Add("")
	f.Add(strings.Repeat("a", 2000))
	f.Add("reason with special chars: <>&\"'")
	f.Add("unicode: 日本語テスト")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.Org{})
	svc := NewService(db)
	ctx := context.Background()

	// Create a test org.
	_ = db.Create(&models.Org{Name: "Fuzz Org", Slug: "fuzz-org", Metadata: "{}"}).Error

	f.Fuzz(func(t *testing.T, reason string) {
		_ = svc.SuspendOrg(ctx, "fuzz-org", reason, "admin")
		_ = svc.UnsuspendOrg(ctx, "fuzz-org")
	})
}

func FuzzAddPlatformAdmin(f *testing.F) {
	f.Add("user_123")
	f.Add("")
	f.Add(strings.Repeat("a", 500))
	f.Add("user with spaces")
	f.Add("user<>special")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		f.Fatal(err)
	}
	_ = db.AutoMigrate(&models.PlatformAdmin{})
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, userID string) {
		// Should not panic.
		_, _ = svc.AddPlatformAdmin(ctx, userID, "fuzzer")
		// Clean up to allow repeated adds.
		db.Where("user_id = ?", userID).Delete(&models.PlatformAdmin{})
	})
}
