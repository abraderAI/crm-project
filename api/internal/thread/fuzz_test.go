package thread

import (
	"context"
	"testing"
)

// FuzzThreadCreate fuzzes thread creation with random titles and metadata strings.
func FuzzThreadCreate(f *testing.F) {
	type seed struct{ title, meta string }
	seeds := []seed{
		{"Thread Title", "{}"},
		{"", "{}"},
		{"a", "{}"},
		{"A Very Long Thread Title That Exceeds Normal Expectations For Thread Names", "{}"},
		{"Thread with Unicode: 日本語", "{}"},
		{"Thread 🎉🎉🎉", "{}"},
		{"thread-with-hyphens", "{}"},
		{"Thread: A Subtitle", "{}"},
		{"' OR 1=1 --", "{}"},
		{"<script>alert(1)</script>", "{}"},
		{"thread\x00null", "{}"},
		{"thread\nnewline", "{}"},
		{"thread\ttab", "{}"},
		{"123 Numeric Thread", "{}"},
		{"UPPERCASE THREAD", "{}"},
		{"MiXeD CaSe ThReAd", "{}"},
		{"thread/with/slashes", "{}"},
		{"thread\\backslash", "{}"},
		{"   leading spaces   ", "{}"},
		{"null", "{}"},
		{"false", "{}"},
		{"title", "not json"},
		{"title", `{bad json}`},
		{"title", `{"status":"open"}`},
		{"title", `{"status":"closed","priority":5}`},
		{"title", `{"stage":"in_progress"}`},
		{"title", `{"assigned_to":"user_abc"}`},
		{"title", `{"tags":["bug","urgent"]}`},
		{"title", `{"nested":{"deep":{"value":true}}}`},
		{"title", `{"unicode":"日本語","emoji":"🎉"}`},
		{"title", `{"null_field":null}`},
		{"title", `{"number":42}`},
		{"title", `{"float":3.14}`},
		{"title", `{"bool":false}`},
		{"title", `{}`},
		{"title", ``},
		{"Duplicate Title", "{}"},
		{"Thread with many words in its title for slug generation testing purposes", "{}"},
		{"!!!", "{}"},
		{"@@@", "{}"},
		{"###", "{}"},
		{"$$$", "{}"},
		{"%test%", "{}"},
		{"  ", "{}"},
		{"a b c d e f g", "{}"},
		{"αβγδεζ", "{}"},
		{"кириллица", "{}"},
		{"中文", "{}"},
		{"العربية", "{}"},
	}
	for _, s := range seeds {
		f.Add(s.title, s.meta)
	}

	f.Fuzz(func(t *testing.T, title, metadata string) {
		db, boardID := setupDB(t)
		svc := NewService(NewRepository(db))
		ctx := context.Background()
		// Should never panic for any input.
		_, _ = svc.Create(ctx, boardID, "fuzz-author", false, CreateInput{
			Title:    title,
			Metadata: metadata,
		})
	})
}
