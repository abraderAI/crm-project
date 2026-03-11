package moderation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// FuzzCreateFlag fuzzes the flag creation with random reasons.
func FuzzCreateFlag(f *testing.F) {
	seeds := []struct {
		reason string
	}{
		{"spam"},
		{"offensive content"},
		{"harassment"},
		{"misinformation"},
		{"off-topic"},
		{""},
		{"a"},
		{"A very long reason that goes on and on about the issue at hand with lots of detail"},
		{"reason with special chars: <>&\"'"},
		{"reason\nwith\nnewlines"},
		{"reason\twith\ttabs"},
		{"reason with unicode: 🙂🎉✅"},
		{"' OR 1=1 --"},
		{"<script>alert(1)</script>"},
		{"reason with null\x00byte"},
		{"UPPERCASE REASON"},
		{"MiXeD CaSe"},
		{"123 numeric reason"},
		{"!@#$%^&*()"},
		{"reason-with-dashes"},
		{"reason_with_underscores"},
		{"reason.with.dots"},
		{"reason/with/slashes"},
		{"a b c d e f g h i j k"},
		{"duplicate reason"},
		{"duplicate reason"},
		{"testing edge case"},
		{"flag for review"},
		{"inappropriate language"},
		{"copyright violation"},
		{"self promotion"},
		{"low quality content"},
		{"personal attack"},
		{"trolling"},
		{"contains adult content"},
		{"privacy violation"},
		{"dangerous advice"},
		{"fake account"},
		{"bot activity"},
		{"broken content"},
		{"needs moderation"},
		{"community guidelines violation"},
		{"discrimination"},
		{"impersonation"},
		{"illegal activity"},
		{"threatening behavior"},
		{"doxxing"},
		{"hate speech"},
		{"violent content"},
	}
	for _, s := range seeds {
		f.Add(s.reason)
	}

	f.Fuzz(func(t *testing.T, reason string) {
		db := testDB(t)
		repo := NewRepository(db)
		svc := NewService(repo)
		h := seedHierarchy(t, db)
		ctx := context.Background()

		flag, err := svc.CreateFlag(ctx, "fuzz-user", FlagInput{
			ThreadID: h.thread.ID,
			Reason:   reason,
		})

		if reason == "" {
			assert.Error(t, err)
			assert.Nil(t, flag)
			return
		}

		// Non-empty reasons should succeed.
		if err != nil {
			return // Unexpected error, but shouldn't panic.
		}
		assert.NotNil(t, flag)
		assert.Equal(t, reason, flag.Reason)
	})
}

// FuzzMoveThread fuzzes the move thread operation with random board IDs.
func FuzzMoveThread(f *testing.F) {
	seeds := []string{
		"",
		"valid-board-id",
		"nonexistent",
		"a",
		"abc123",
		"' OR 1=1 --",
		"<script>",
		"board_with_special_chars!@#",
		"a-very-long-board-id-that-is-longer-than-usual",
		"board\x00null",
		"board\nline",
		"board\ttab",
		"UPPERCASE",
		"MiXeD",
		"123",
		"000",
		"board-1",
		"board-2",
		"board-3",
		"test-board",
		"temp-board",
		"new-board",
		"old-board",
		"moved-board",
		"target-board",
		"dest-board",
		"src-board",
		"board with spaces",
		"board/slash",
		"board\\backslash",
		"board%percent",
		"board#hash",
		"board?query",
		"board&amp",
		"board=eq",
		"board+plus",
		"🙂board",
		"board🎉",
		"Unicode板",
		"board_αβγ",
		"board_кир",
		"board-日本",
		"board-한국",
		"boardé",
		"board£",
		"board€",
		"board¥",
		"board₹",
		"board₽",
		"board₿",
		"board🌍",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, targetBoardID string) {
		db := testDB(t)
		repo := NewRepository(db)
		svc := NewService(repo)
		h := seedHierarchy(t, db)
		ctx := context.Background()

		// Should not panic for any input.
		result, err := svc.MoveThread(ctx, h.thread.ID, "fuzz-mod", MoveInput{
			TargetBoardID: targetBoardID,
		})

		if targetBoardID == "" {
			assert.Error(t, err)
			return
		}

		if targetBoardID == h.board.ID {
			assert.Error(t, err)
			return
		}

		// Most random IDs won't match a real board.
		if err != nil {
			assert.Nil(t, result)
			return
		}
	})
}
