package board

import (
	"context"
	"testing"
)

// FuzzBoardCreate fuzzes board creation with random names and metadata strings.
func FuzzBoardCreate(f *testing.F) {
	type seed struct{ name, meta string }
	seeds := []seed{
		{"Feature Board", "{}"},
		{"Bug Tracker", `{"priority":"high"}`},
		{"", "{}"},
		{"a", "{}"},
		{"A Very Long Board Name That Exceeds Normal Length Expectations For Testing", "{}"},
		{"Board with Unicode: 日本語テスト", "{}"},
		{"Board 🎉", "{}"},
		{"board-with-hyphens", "{}"},
		{"board_with_underscores", "{}"},
		{"Board Name With Spaces", `{"tag":"test"}`},
		{"123numeric", "{}"},
		{"UPPERCASE BOARD", "{}"},
		{"MiXeD CaSe BoArD", "{}"},
		{"' OR 1=1 --", "{}"},
		{"<script>alert(1)</script>", "{}"},
		{"board\x00null", "{}"},
		{"board\nnewline", "{}"},
		{"board\ttab", "{}"},
		{"board name", "not json"},
		{"board", `{invalid json}`},
		{"board", `{"key": null}`},
		{"board", `{"key": 123}`},
		{"board", `{"key": true}`},
		{"board", `{"key": ["array"]}`},
		{"board", `{"key": {"nested": "value"}}`},
		{"board", `{}`},
		{"board", ``},
		{"board", `{"a":"b","c":"d","e":"f"}`},
		{"test board", `{"status":"active","priority":1}`},
		{"unicode meta", `{"emoji":"🎉","unicode":"日本語"}`},
		{"special chars: !@#$%^&*()", "{}"},
		{"board-αβγ", "{}"},
		{"board-кириллица", "{}"},
		{"   leading spaces", "{}"},
		{"trailing spaces   ", "{}"},
		{"  both  ", "{}"},
		{"\t\tonly tabs\t\t", "{}"},
		{"null", "{}"},
		{"true", "{}"},
		{"false", "{}"},
		{"0", "{}"},
		{"-1", "{}"},
		{"NaN", "{}"},
		{"Infinity", "{}"},
		{"board/slash", "{}"},
		{"board\\backslash", "{}"},
		{"board.dots", "{}"},
		{"Dup Board", "{}"},
		{"another-board", `{"key":"value"}`},
		{"yet-another", `{"nested":{"a":1}}`},
	}
	for _, s := range seeds {
		f.Add(s.name, s.meta)
	}

	f.Fuzz(func(t *testing.T, name, metadata string) {
		db, spaceID := setupDB(t)
		svc := NewService(NewRepository(db))
		ctx := context.Background()
		// Should never panic for any input.
		_, _ = svc.Create(ctx, spaceID, CreateInput{Name: name, Metadata: metadata})
	})
}
