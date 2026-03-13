package admin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Phase B Fuzz Tests ---

func FuzzUpdateSettingsValues(f *testing.F) {
	f.Add(`{"max_size":100}`)
	f.Add(`{}`)
	f.Add(`{"a":"b","c":{"d":1}}`)
	f.Add(`[]`)
	f.Add(`"string"`)
	f.Add(`null`)
	f.Add(strings.Repeat(`{"a":`, 100) + `1` + strings.Repeat(`}`, 100))
	f.Add(`{"max_size":-1,"allowed_types":[]}`)

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, value string) {
		patch := map[string]json.RawMessage{
			"file_upload_limits": json.RawMessage(value),
		}
		// Should not panic.
		_ = svc.UpdateSettings(ctx, patch, "fuzzer")
	})
}

func FuzzRBACOverride(f *testing.F) {
	f.Add(`{"roles":{"permissions":{"viewer":["read","write"]}}}`)
	f.Add(`{}`)
	f.Add(`{"defaults":{"org_member_role":"admin"}}`)
	f.Add(`{"roles":{"permissions":{"fake_role":["perm"]}}}`)
	f.Add(`invalid json`)
	f.Add(`{"roles":null}`)

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, input string) {
		var override RBACOverride
		if err := json.Unmarshal([]byte(input), &override); err != nil {
			return // Skip invalid JSON.
		}
		// Should not panic.
		_ = svc.UpdateRBACOverride(ctx, override, "fuzzer")
	})
}

func FuzzFeatureFlagKey(f *testing.F) {
	f.Add("maintenance_mode")
	f.Add("community_voting")
	f.Add("")
	f.Add(strings.Repeat("x", 1000))
	f.Add("<script>alert('xss')</script>")
	f.Add("flag with spaces")
	f.Add("flag\nwith\nnewlines")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.SeedFeatureFlags(ctx)

	f.Fuzz(func(t *testing.T, key string) {
		// Should not panic.
		_, _ = svc.GetFeatureFlag(ctx, key)
		_ = svc.ToggleFeatureFlag(ctx, key, true, nil)
		_, _ = svc.IsFeatureEnabled(ctx, key)
	})
}

func FuzzSettingsKey(f *testing.F) {
	f.Add("file_upload_limits")
	f.Add("unknown_key")
	f.Add("")
	f.Add(strings.Repeat("a", 500))
	f.Add("key with special <>&\"' chars")

	db := setupFuzzDB(f)
	svc := NewService(db)
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, key string) {
		patch := map[string]json.RawMessage{
			key: json.RawMessage(`{}`),
		}
		// Should not panic.
		_ = svc.UpdateSettings(ctx, patch, "fuzzer")
		_, _ = svc.GetSetting(ctx, key)
	})
}

func setupFuzzDB(f interface{ Fatal(...any) }) *gorm.DB {
	db := setupFuzzDBInner()
	if db == nil {
		f.Fatal("failed to setup fuzz DB")
	}
	return db
}

func setupFuzzDBInner() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil
	}
	_ = db.AutoMigrate(
		&models.SystemSetting{},
		&models.FeatureFlag{},
		&models.Org{},
		&models.UserShadow{},
		&models.OrgMembership{},
	)
	return db
}
