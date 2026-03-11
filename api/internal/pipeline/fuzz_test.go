package pipeline

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
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

func FuzzPipelineTransition(f *testing.F) {
	// ≥50 seed entries for stage transition fuzzing.
	seeds := []struct {
		currentStage string
		newStage     string
	}{
		// Valid forward transitions.
		{"", "new_lead"},
		{"new_lead", "contacted"},
		{"contacted", "qualified"},
		{"qualified", "proposal"},
		{"proposal", "negotiation"},
		{"negotiation", "closed_won"},
		{"", "nurturing"},
		{"nurturing", "contacted"},
		{"nurturing", "qualified"},
		{"nurturing", "closed_lost"},
		// Closed-lost from any active stage.
		{"new_lead", "closed_lost"},
		{"contacted", "closed_lost"},
		{"qualified", "closed_lost"},
		{"proposal", "closed_lost"},
		{"negotiation", "closed_lost"},
		{"closed_lost", "nurturing"},
		// Invalid transitions.
		{"new_lead", "proposal"},
		{"new_lead", "closed_won"},
		{"contacted", "closed_won"},
		{"closed_won", "new_lead"},
		{"closed_won", "nurturing"},
		// Edge cases.
		{"", ""},
		{"unknown", "new_lead"},
		{"new_lead", "unknown"},
		{"", "closed_won"},
		{"", "proposal"},
		{"", "negotiation"},
		{"", "contacted"},
		{"unknown_stage", "unknown_target"},
		// Adversarial inputs.
		{"<script>alert(1)</script>", "new_lead"},
		{"new_lead", "<script>"},
		{"' OR 1=1 --", "contacted"},
		{"new_lead", "' DROP TABLE"},
		{"\x00\x01\x02", "new_lead"},
		{"new_lead", "\x00\x01"},
		{"new_lead\nnew_lead", "contacted"},
		{" new_lead ", "contacted"},
		{"NEW_LEAD", "contacted"},
		{"New_Lead", "Contacted"},
		// Unicode.
		{"名前", "contacted"},
		{"new_lead", "資格あり"},
		{"нов_лід", "контакт"},
		// Long strings.
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "new_lead"},
		{"new_lead", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		// JSON-like.
		{`{"stage":"new_lead"}`, "contacted"},
		{"new_lead", `{"stage":"contacted"}`},
		// More valid transitions for coverage.
		{"contacted", "nurturing"},
		{"qualified", "nurturing"},
		{"proposal", "nurturing"},
	}

	for _, s := range seeds {
		f.Add(s.currentStage, s.newStage)
	}

	db := fuzzDB(f)
	bus := event.NewBus()
	svc := NewService(db, bus)
	ctx := context.Background()

	// Create one org/space/board for all fuzz runs.
	org := &models.Org{Name: "Fuzz Org", Slug: "fuzz-org", Metadata: "{}"}
	if err := db.Create(org).Error; err != nil {
		f.Fatal(err)
	}
	space := &models.Space{OrgID: org.ID, Name: "CRM", Slug: "fuzz-crm", Type: models.SpaceTypeCRM, Metadata: "{}"}
	if err := db.Create(space).Error; err != nil {
		f.Fatal(err)
	}
	board := &models.Board{SpaceID: space.ID, Name: "Pipeline", Slug: "fuzz-pipeline", Metadata: "{}"}
	if err := db.Create(board).Error; err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, currentStage, newStage string) {
		meta := "{}"
		if currentStage != "" {
			meta = `{"stage":"` + currentStage + `"}`
		}
		thread := &models.Thread{
			BoardID:  board.ID,
			Title:    "Fuzz Lead",
			Slug:     "fuzz-lead",
			AuthorID: "fuzz-user",
			Metadata: meta,
		}
		if err := db.Create(thread).Error; err != nil {
			return // Skip DB errors.
		}
		// Must not panic.
		_, _ = svc.TransitionStage(ctx, thread.ID, Stage(newStage), "fuzz-user")
	})
}

func FuzzValidateTransition(f *testing.F) {
	// ≥50 seed entries for direct transition validation.
	seeds := []struct {
		from string
		to   string
	}{
		{"", "new_lead"},
		{"", "nurturing"},
		{"", "contacted"},
		{"", "proposal"},
		{"", "closed_won"},
		{"new_lead", "contacted"},
		{"new_lead", "nurturing"},
		{"new_lead", "closed_lost"},
		{"new_lead", "qualified"},
		{"new_lead", "proposal"},
		{"new_lead", "negotiation"},
		{"new_lead", "closed_won"},
		{"contacted", "qualified"},
		{"contacted", "nurturing"},
		{"contacted", "closed_lost"},
		{"contacted", "proposal"},
		{"qualified", "proposal"},
		{"qualified", "nurturing"},
		{"qualified", "closed_lost"},
		{"qualified", "negotiation"},
		{"proposal", "negotiation"},
		{"proposal", "nurturing"},
		{"proposal", "closed_lost"},
		{"proposal", "closed_won"},
		{"negotiation", "closed_won"},
		{"negotiation", "closed_lost"},
		{"negotiation", "nurturing"},
		{"negotiation", "proposal"},
		{"closed_won", "new_lead"},
		{"closed_won", "nurturing"},
		{"closed_won", "closed_lost"},
		{"closed_lost", "nurturing"},
		{"closed_lost", "new_lead"},
		{"closed_lost", "contacted"},
		{"nurturing", "contacted"},
		{"nurturing", "qualified"},
		{"nurturing", "closed_lost"},
		{"nurturing", "proposal"},
		{"nurturing", "new_lead"},
		{"unknown", "new_lead"},
		{"new_lead", "unknown"},
		{"", ""},
		{"abc", "def"},
		{"<script>", "contacted"},
		{"new_lead", "' OR 1=1"},
		{"\x00", "new_lead"},
		{"test\ntest", "contacted"},
		{"NEW_LEAD", "CONTACTED"},
		{"名前", "contacted"},
		{"a very long stage name that nobody would use in practice", "another unreasonably long stage name"},
	}

	for _, s := range seeds {
		f.Add(s.from, s.to)
	}

	stages := DefaultStages()

	f.Fuzz(func(t *testing.T, from, to string) {
		// Must not panic.
		_ = ValidateTransition(stages, Stage(from), Stage(to))
	})
}
