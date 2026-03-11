package org

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
)

func fuzzDB(f *testing.F) *gorm.DB {
	f.Helper()
	dir := f.TempDir()
	dbPath := filepath.Join(dir, "fuzz.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		f.Fatal(err)
	}
	sqlDB, _ := db.DB()
	_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON")
	if err := database.Migrate(db); err != nil {
		f.Fatal(err)
	}
	return db
}

func FuzzOrgCreate(f *testing.F) {
	f.Add("Acme Corp", "A test org", `{"tier":"free"}`)
	f.Add("", "", "{}")
	f.Add("Org <script>alert(1)</script>", "Desc", "")
	f.Add("名前テスト", "Unicode org", `{"lang":"ja"}`)
	f.Add("A", "", "not json")
	f.Add("Very Long Name "+string(make([]byte, 500)), "", "{}")

	db := fuzzDB(f)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, name, desc, meta string) {
		_, _ = svc.Create(ctx, CreateInput{
			Name:        name,
			Description: desc,
			Metadata:    meta,
		})
	})
}

func FuzzOrgUpdate(f *testing.F) {
	f.Add("New Name", "New Desc", `{"key":"value"}`)
	f.Add("", "", "not json")
	f.Add("Renamed", "", `{"nested":{"deep":true}}`)

	db := fuzzDB(f)
	svc := NewService(NewRepository(db))
	ctx := context.Background()

	org, err := svc.Create(ctx, CreateInput{Name: "Fuzz Target"})
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, name, desc, meta string) {
		input := UpdateInput{}
		if name != "" {
			input.Name = &name
		}
		if desc != "" {
			input.Description = &desc
		}
		if meta != "" {
			input.Metadata = &meta
		}
		_, _ = svc.Update(ctx, org.ID, input)
	})
}
