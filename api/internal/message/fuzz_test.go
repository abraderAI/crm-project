package message

import (
	"context"
	"testing"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// FuzzMessageCreate fuzzes message creation with random bodies and message types.
func FuzzMessageCreate(f *testing.F) {
	type seed struct{ body, msgType string }
	seeds := []seed{
		{"Hello", "comment"},
		{"", "comment"},
		{"a", "comment"},
		{"A very long message body with lots of content and special characters!@#$%^&*()", "comment"},
		{"Message with Unicode: 日本語テスト", "comment"},
		{"Message 🎉🚀✅", "comment"},
		{"' OR 1=1 --", "comment"},
		{"<script>alert(1)</script>", "comment"},
		{"message\x00null", "comment"},
		{"message\nnewline", "comment"},
		{"message\ttab", "comment"},
		{"Hello", ""},
		{"Hello", "invalid"},
		{"Hello", "COMMENT"},
		{"Hello", "Comment"},
		{"Hello", "system_message"},
		{"Hello", "note"},
		{"Hello", "action"},
		{"Hello", "question"},
		{"Hello", "answer"},
		{"Hello", "status_update"},
		{"Hello", "email"},
		{"Hello", "call_transcript"},
		{"Hello", "unknown_type"},
		{"Hello", "' OR 1=1 --"},
		{"Hello", "<script>"},
		{"Hello", "123"},
		{"Hello", "true"},
		{"Hello", "null"},
		{"Hello", "type with spaces"},
		{"Hello", "type-with-hyphens"},
		{"Hello", "type_underscore"},
		{"", ""},
		{"", "invalid"},
		{"message with\nnewlines\nand tabs\there", "comment"},
		{"unicode message: αβγδ кириллица 中文 العربية", "comment"},
		{"  leading spaces  ", "comment"},
		{"null", "comment"},
		{"false", "comment"},
		{"0", "comment"},
		{"[]", "comment"},
		{"{}", "comment"},
		{"<b>bold</b>", "comment"},
		{"[link](http://example.com)", "comment"},
		{"# Heading", "comment"},
		{"**bold** _italic_", "comment"},
		{"```code```", "comment"},
		{"\x00\x01\x02\x03", "comment"},
	}
	for _, s := range seeds {
		f.Add(s.body, s.msgType)
	}

	f.Fuzz(func(t *testing.T, body, msgType string) {
		db, threadID := setupDB(t)
		svc := NewService(NewRepository(db))
		ctx := context.Background()
		// Should never panic for any input.
		_, _ = svc.Create(ctx, threadID, "fuzz-author", false, CreateInput{
			Body: body,
			Type: models.MessageType(msgType),
		})
	})
}
