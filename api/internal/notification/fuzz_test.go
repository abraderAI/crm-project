package notification

import (
	"testing"
	"time"

	"github.com/abraderAI/crm-project/api/internal/eventbus"
)

// FuzzMapEventToNotification fuzzes the event-to-notification mapper with random event types.
func FuzzMapEventToNotification(f *testing.F) {
	type seed struct {
		eventType  string
		entityType string
		entityID   string
		userID     string
	}
	seeds := []seed{
		{"message.created", "thread", "thread-1", "user1"},
		{"thread.updated", "thread", "thread-2", "user2"},
		{"mention", "thread", "thread-3", "user3"},
		{"invite", "org", "org-1", "user4"},
		{"", "thread", "thread-1", "user1"},
		{"unknown.event", "thread", "thread-1", "user1"},
		{"message.deleted", "message", "msg-1", "user1"},
		{"thread.created", "thread", "thread-5", "user5"},
		{"org.updated", "org", "org-2", "user6"},
		{"member.added", "org", "org-3", "user7"},
		{"member.removed", "org", "org-4", "user8"},
		{"' OR 1=1 --", "thread", "x", "y"},
		{"<script>alert(1)</script>", "thread", "x", "y"},
		{"event\x00null", "thread", "x", "y"},
		{"event\nnewline", "thread", "x", "y"},
		{"event.created.extra.parts", "thread", "x", "y"},
		{"UPPERCASE.EVENT", "thread", "x", "y"},
		{"MiXeD.EveNt", "thread", "x", "y"},
		{"message.created", "", "", ""},
		{"message.created", "thread", "", ""},
		{"message.created", "", "id", "user"},
		{"thread.updated", "thread", "id", "user"},
		{"a", "b", "c", "d"},
		{".", ".", ".", "."},
		{"..", "..", "..", ".."},
		{"event type with spaces", "thread", "x", "y"},
		{"event-with-hyphens", "thread", "x", "y"},
		{"event_underscores", "thread", "x", "y"},
		{"αβγ.event", "thread", "x", "y"},
		{"кирилл", "thread", "x", "y"},
		{"中文事件", "thread", "x", "y"},
		{"event 🎉", "thread", "x", "y"},
		{"null", "null", "null", "null"},
		{"true", "false", "0", "-1"},
		{"0", "0", "0", "0"},
		{"", "", "", ""},
		{"message.created", "message", "msg-123", "author-1"},
		{"thread.updated", "thread", "thread-abc", "editor-2"},
		{"mention", "message", "msg-456", "user-3"},
		{"invite", "space", "space-789", "user-4"},
		{"message.created", "thread\x00null", "id", "user"},
		{"message.created", "thread", "id\x00null", "user"},
		{"message.created", "thread", "id", "user\x00null"},
	}
	for _, s := range seeds {
		f.Add(s.eventType, s.entityType, s.entityID, s.userID)
	}

	f.Fuzz(func(t *testing.T, eventType, entityType, entityID, userID string) {
		event := eventbus.Event{
			Type:       eventType,
			EntityType: entityType,
			EntityID:   entityID,
			UserID:     userID,
			Timestamp:  time.Now(),
		}
		// Pure function — should never panic.
		_, _, _ = mapEventToNotification(event)
	})
}

// FuzzNotificationValidate fuzzes NotificationInput.Validate with random inputs.
func FuzzNotificationValidate(f *testing.F) {
	type seed struct {
		userID string
		ntype  string
		title  string
	}
	seeds := []seed{
		{"user1", "message", "Title"},
		{"", "message", "Title"},
		{"user1", "", "Title"},
		{"user1", "message", ""},
		{"", "", ""},
		{"a", "a", "a"},
		{"user", "type", "title with spaces"},
		{"' OR 1=1", "test", "title"},
		{"user", "' OR 1=1", "title"},
		{"user", "test", "' OR 1=1"},
		{"<script>", "type", "title"},
		{"user", "<script>", "title"},
		{"user", "type", "<script>"},
		{"user\x00null", "type", "title"},
		{"user", "type\x00null", "title"},
		{"user", "type", "title\x00null"},
		{"user\nnewline", "type", "title"},
		{"αβγ", "type", "title"},
		{"кирилл", "type", "title"},
		{"中文", "type", "title"},
		{"user 🎉", "type", "title"},
		{"null", "null", "null"},
		{"true", "false", "0"},
		{"   ", "   ", "   "},
		{"user", TypeNewMessage, "New message"},
		{"user", TypeStageChange, "Stage changed"},
		{"user", TypeAssignment, "Assigned"},
		{"user", TypeMention, "You were mentioned"},
		{"user", TypeInvite, "You're invited"},
		{"user", "custom_type", "Custom title"},
		{"user", "type_with_very_long_name_that_exceeds_expectations", "title"},
		{"very_long_user_id_that_exceeds_normal_expectations_for_clerk_ids", "type", "title"},
		{"user", "type", "a very long title that might exceed any length limits in place for titles"},
		{"user", "new_message", "Title"},
		{"user", "stage_change", "Title"},
		{"user", "assignment", "Title"},
		{"user", "mention", "Title"},
		{"user", "invite", "Title"},
		{"-", "-", "-"},
		{"user_id", "type-name", "Title: Subtitle"},
		{"user_id", "type.sub", "Title with 🎉"},
		{"u", "t", "T"},
	}
	for _, s := range seeds {
		f.Add(s.userID, s.ntype, s.title)
	}

	f.Fuzz(func(t *testing.T, userID, ntype, title string) {
		input := NotificationInput{
			UserID: userID,
			Type:   ntype,
			Title:  title,
		}
		// Validate should never panic.
		_ = input.Validate()
	})
}
