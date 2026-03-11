package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
	ws "github.com/abraderAI/crm-project/api/internal/websocket"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- NotificationInput tests ---

func TestNotificationInput_Validate(t *testing.T) {
	tests := []struct {
		name  string
		input NotificationInput
		err   error
	}{
		{"valid", NotificationInput{UserID: "u1", Type: "test", Title: "hi"}, nil},
		{"missing user", NotificationInput{Type: "test", Title: "hi"}, ErrUserIDRequired},
		{"missing type", NotificationInput{UserID: "u1", Title: "hi"}, ErrTypeRequired},
		{"missing title", NotificationInput{UserID: "u1", Type: "test"}, ErrTitleRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			assert.Equal(t, tt.err, err)
		})
	}
}

// --- Repository tests ---

func TestRepository_CreateAndFindByID(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	notif := &models.Notification{
		UserID: "user1",
		Type:   TypeNewMessage,
		Title:  "New message",
		Body:   "You have a new message",
	}
	require.NoError(t, repo.Create(ctx, notif))
	assert.NotEmpty(t, notif.ID)

	found, err := repo.FindByID(ctx, notif.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "New message", found.Title)
	assert.False(t, found.IsRead)
}

func TestRepository_FindByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)

	found, err := repo.FindByID(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_ListByUser(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, &models.Notification{
			UserID: "user1",
			Type:   TypeNewMessage,
			Title:  "Msg",
		}))
	}
	require.NoError(t, repo.Create(ctx, &models.Notification{
		UserID: "user2",
		Type:   TypeNewMessage,
		Title:  "Other",
	}))

	notifs, pageInfo, err := repo.ListByUser(ctx, "user1", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, notifs, 5)
	assert.False(t, pageInfo.HasMore)
}

func TestRepository_ListByUser_Pagination(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, &models.Notification{
			UserID: "user1",
			Type:   TypeNewMessage,
			Title:  "Msg",
		}))
	}

	notifs, pageInfo, err := repo.ListByUser(ctx, "user1", pagination.Params{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, notifs, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)
}

func TestRepository_MarkRead(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	notif := &models.Notification{UserID: "user1", Type: TypeNewMessage, Title: "Test"}
	require.NoError(t, repo.Create(ctx, notif))

	require.NoError(t, repo.MarkRead(ctx, notif.ID, "user1"))

	found, _ := repo.FindByID(ctx, notif.ID)
	assert.True(t, found.IsRead)
}

func TestRepository_MarkRead_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)

	err := repo.MarkRead(context.Background(), "nonexistent", "user1")
	assert.Equal(t, ErrNotFound, err)
}

func TestRepository_MarkRead_WrongUser(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	notif := &models.Notification{UserID: "user1", Type: TypeNewMessage, Title: "Test"}
	require.NoError(t, repo.Create(ctx, notif))

	err := repo.MarkRead(ctx, notif.ID, "user2")
	assert.Equal(t, ErrNotFound, err)
}

func TestRepository_MarkAllRead(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &models.Notification{
			UserID: "user1", Type: TypeNewMessage, Title: "Test",
		}))
	}

	count, err := repo.MarkAllRead(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	unread, _ := repo.CountUnread(ctx, "user1")
	assert.Equal(t, int64(0), unread)
}

func TestRepository_CountUnread(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &models.Notification{
			UserID: "user1", Type: TypeNewMessage, Title: "Test",
		}))
	}

	count, err := repo.CountUnread(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestRepository_Preferences(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	pref := &models.NotificationPreference{
		UserID:    "user1",
		EventType: TypeNewMessage,
		Channel:   ChannelEmail,
		Enabled:   false,
	}
	require.NoError(t, repo.UpsertPreference(ctx, pref))

	prefs, err := repo.GetPreferences(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, prefs, 1)
	assert.False(t, prefs[0].Enabled)

	// Update.
	pref.Enabled = true
	require.NoError(t, repo.UpsertPreference(ctx, pref))

	prefs, _ = repo.GetPreferences(ctx, "user1")
	assert.True(t, prefs[0].Enabled)
}

func TestRepository_IsChannelEnabled_Default(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)

	enabled, err := repo.IsChannelEnabled(context.Background(), "user1", TypeNewMessage, ChannelInApp)
	require.NoError(t, err)
	assert.True(t, enabled) // Default enabled.
}

func TestRepository_IsChannelEnabled_Disabled(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.UpsertPreference(ctx, &models.NotificationPreference{
		UserID: "user1", EventType: TypeNewMessage, Channel: ChannelEmail, Enabled: false,
	}))

	enabled, err := repo.IsChannelEnabled(ctx, "user1", TypeNewMessage, ChannelEmail)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestRepository_DigestSchedule(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// No schedule.
	sched, err := repo.GetDigestSchedule(ctx, "user1")
	require.NoError(t, err)
	assert.Nil(t, sched)

	// Create.
	require.NoError(t, repo.UpsertDigestSchedule(ctx, &models.DigestSchedule{
		UserID: "user1", Frequency: "daily", Enabled: true,
	}))

	sched, err = repo.GetDigestSchedule(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, sched)
	assert.Equal(t, "daily", sched.Frequency)

	// Update.
	require.NoError(t, repo.UpsertDigestSchedule(ctx, &models.DigestSchedule{
		UserID: "user1", Frequency: "weekly", Enabled: true,
	}))

	sched, _ = repo.GetDigestSchedule(ctx, "user1")
	assert.Equal(t, "weekly", sched.Frequency)
}

func TestRepository_GetUsersWithDigestEnabled(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.UpsertDigestSchedule(ctx, &models.DigestSchedule{
		UserID: "user1", Frequency: "daily", Enabled: true,
	}))
	require.NoError(t, repo.UpsertDigestSchedule(ctx, &models.DigestSchedule{
		UserID: "user2", Frequency: "weekly", Enabled: true,
	}))
	require.NoError(t, repo.UpsertDigestSchedule(ctx, &models.DigestSchedule{
		UserID: "user3", Frequency: "daily", Enabled: false,
	}))

	users, err := repo.GetUsersWithDigestEnabled(ctx, "daily")
	require.NoError(t, err)
	assert.Equal(t, []string{"user1"}, users)
}

func TestRepository_GetUnreadNotifications(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "Unread1",
	}))
	require.NoError(t, repo.Create(ctx, &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "Unread2",
	}))

	notifs, err := repo.GetUnreadNotifications(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, notifs, 2)
}

// --- InApp Provider tests ---

func TestInAppProvider_Send(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	hub := ws.NewHub(testLogger())

	provider := NewInAppProvider(repo, hub, testLogger())
	assert.Equal(t, ChannelInApp, provider.Name())

	notif := &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "Test", Body: "Body",
	}
	require.NoError(t, provider.Send(context.Background(), notif))
	assert.NotEmpty(t, notif.ID)

	// Verify stored in DB.
	found, _ := repo.FindByID(context.Background(), notif.ID)
	require.NotNil(t, found)
	assert.Equal(t, "Test", found.Title)
}

// --- Email Provider tests ---

func TestEmailProvider_Send(t *testing.T) {
	sender := &NoOpEmailSender{}
	resolver := &NoOpEmailResolver{Emails: map[string]string{"user1": "test@example.com"}}
	provider := NewEmailProvider(sender, resolver, testLogger())

	assert.Equal(t, ChannelEmail, provider.Name())

	notif := &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "New message", Body: "Hello!",
	}
	require.NoError(t, provider.Send(context.Background(), notif))
	assert.Len(t, sender.Sent, 1)
	assert.Equal(t, "test@example.com", sender.Sent[0].To)
	assert.Equal(t, "New message", sender.Sent[0].Subject)
}

func TestEmailProvider_Send_NoEmail(t *testing.T) {
	sender := &NoOpEmailSender{}
	resolver := &NoOpEmailResolver{} // No emails configured.
	provider := NewEmailProvider(sender, resolver, testLogger())

	notif := &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "Test",
	}
	require.NoError(t, provider.Send(context.Background(), notif))
	assert.Empty(t, sender.Sent) // Should not send.
}

func TestRenderNotificationEmail(t *testing.T) {
	notif := &models.Notification{
		Title: "New Message",
		Body:  "Hello world",
		Type:  TypeNewMessage,
	}
	html := RenderNotificationEmail(notif)
	assert.Contains(t, html, "New Message")
	assert.Contains(t, html, "Hello world")
	assert.Contains(t, html, TypeNewMessage)
}

func TestRenderDigestEmail(t *testing.T) {
	notifs := []models.Notification{
		{Title: "Msg 1", Body: "Body 1"},
		{Title: "Msg 2", Body: "Body 2"},
	}
	html := RenderDigestEmail(notifs)
	assert.Contains(t, html, "Notification Digest")
	assert.Contains(t, html, "2 unread")
	assert.Contains(t, html, "Msg 1")
	assert.Contains(t, html, "Msg 2")
}

// --- Digest Engine tests ---

func TestDigestEngine_SendUserDigest(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Create unreads.
	require.NoError(t, repo.Create(ctx, &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "Msg 1", Body: "B1",
	}))

	sender := &NoOpEmailSender{}
	resolver := &NoOpEmailResolver{Emails: map[string]string{"user1": "test@example.com"}}

	engine := NewDigestEngine(repo, sender, resolver, testLogger(), time.Hour)
	require.NoError(t, engine.SendUserDigest(ctx, "user1"))

	assert.Len(t, sender.Sent, 1)
	assert.Equal(t, "test@example.com", sender.Sent[0].To)
	assert.Contains(t, sender.Sent[0].Body, "Notification Digest")
}

func TestDigestEngine_SendUserDigest_NoUnreads(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	sender := &NoOpEmailSender{}
	resolver := &NoOpEmailResolver{Emails: map[string]string{"user1": "test@example.com"}}

	engine := NewDigestEngine(repo, sender, resolver, testLogger(), time.Hour)
	require.NoError(t, engine.SendUserDigest(context.Background(), "user1"))

	assert.Empty(t, sender.Sent) // No unreads, no email.
}

func TestDigestEngine_SendDigestsNow(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.UpsertDigestSchedule(ctx, &models.DigestSchedule{
		UserID: "user1", Frequency: "daily", Enabled: true,
	}))
	require.NoError(t, repo.Create(ctx, &models.Notification{
		UserID: "user1", Type: TypeNewMessage, Title: "Test",
	}))

	sender := &NoOpEmailSender{}
	resolver := &NoOpEmailResolver{Emails: map[string]string{"user1": "test@example.com"}}

	engine := NewDigestEngine(repo, sender, resolver, testLogger(), time.Hour)
	engine.SendDigestsNow(ctx, "daily")

	assert.Len(t, sender.Sent, 1)
}

func TestDigestEngine_StartStop(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	sender := &NoOpEmailSender{}
	resolver := &NoOpEmailResolver{}

	engine := NewDigestEngine(repo, sender, resolver, testLogger(), time.Millisecond*50)
	engine.Start()
	time.Sleep(10 * time.Millisecond)
	engine.Stop()
}

func TestCreateDigestNotification(t *testing.T) {
	notif := CreateDigestNotification("user1", 5)
	assert.Equal(t, "user1", notif.UserID)
	assert.Equal(t, TypeDigest, notif.Type)
	assert.Equal(t, "Notification Digest", notif.Title)
	assert.Contains(t, notif.Body, "5")
}

// --- Trigger Engine tests ---

func TestTriggerEngine_MessageCreated(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:       "message.created",
		EntityType: "message",
		EntityID:   "msg1",
		UserID:     "sender",
		Payload: map[string]any{
			"participants": []any{"user2", "user3"},
			"title":        "Thread title",
		},
	})

	time.Sleep(100 * time.Millisecond)

	// user2 and user3 should have notifications (sender excluded).
	notifs2, _, _ := repo.ListByUser(context.Background(), "user2", pagination.Params{Limit: 50})
	assert.Len(t, notifs2, 1)
	assert.Equal(t, TypeNewMessage, notifs2[0].Type)

	notifs3, _, _ := repo.ListByUser(context.Background(), "user3", pagination.Params{Limit: 50})
	assert.Len(t, notifs3, 1)
}

func TestTriggerEngine_MentionEvent(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:   "mention",
		UserID: "sender",
		Payload: map[string]any{
			"mentions": []any{"user2"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _, _ := repo.ListByUser(context.Background(), "user2", pagination.Params{Limit: 50})
	assert.Len(t, notifs, 1)
	assert.Equal(t, TypeMention, notifs[0].Type)
}

func TestTriggerEngine_StageChange(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:   "thread.updated",
		UserID: "sender",
		Payload: map[string]any{
			"stage":        "qualified",
			"participants": []any{"user2"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _, _ := repo.ListByUser(context.Background(), "user2", pagination.Params{Limit: 50})
	assert.Len(t, notifs, 1)
	assert.Equal(t, TypeStageChange, notifs[0].Type)
}

func TestTriggerEngine_Assignment(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:   "thread.updated",
		UserID: "sender",
		Payload: map[string]any{
			"assigned_to": "user2",
		},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _, _ := repo.ListByUser(context.Background(), "user2", pagination.Params{Limit: 50})
	assert.Len(t, notifs, 1)
	assert.Equal(t, TypeAssignment, notifs[0].Type)
}

func TestTriggerEngine_SkipsSender(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:   "message.created",
		UserID: "sender",
		Payload: map[string]any{
			"participants": []any{"sender"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _, _ := repo.ListByUser(context.Background(), "sender", pagination.Params{Limit: 50})
	assert.Empty(t, notifs) // Sender should not get own notification.
}

func TestTriggerEngine_RespectsPreferences(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())
	ctx := context.Background()

	// Disable in_app for user2.
	require.NoError(t, repo.UpsertPreference(ctx, &models.NotificationPreference{
		UserID: "user2", EventType: TypeNewMessage, Channel: ChannelInApp, Enabled: false,
	}))

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:   "message.created",
		UserID: "sender",
		Payload: map[string]any{
			"participants": []any{"user2"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _, _ := repo.ListByUser(ctx, "user2", pagination.Params{Limit: 50})
	assert.Empty(t, notifs) // Preference disabled.
}

func TestTriggerEngine_UnknownEvent(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	defer trigger.Stop()

	bus.Publish(eventbus.Event{
		Type:   "unknown.event",
		UserID: "sender",
		Payload: map[string]any{
			"participants": []any{"user2"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _, _ := repo.ListByUser(context.Background(), "user2", pagination.Params{Limit: 50})
	assert.Empty(t, notifs) // Unknown event type, no notification.
}

func TestTriggerEngine_StartStop(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	bus := eventbus.New()
	hub := ws.NewHub(testLogger())

	inApp := NewInAppProvider(repo, hub, testLogger())
	trigger := NewTriggerEngine(bus, repo, []NotificationProvider{inApp}, testLogger())
	trigger.Start()
	trigger.Stop()
}

// --- Helper function tests ---

func TestMapEventToNotification(t *testing.T) {
	tests := []struct {
		name      string
		event     eventbus.Event
		wantType  string
		wantTitle string
	}{
		{"message.created", eventbus.Event{Type: "message.created"}, TypeNewMessage, "New message"},
		{"stage change", eventbus.Event{Type: "thread.updated", Payload: map[string]any{"stage": "x"}}, TypeStageChange, "Stage changed"},
		{"assignment", eventbus.Event{Type: "thread.updated", Payload: map[string]any{"assigned_to": "u1"}}, TypeAssignment, "Assigned to you"},
		{"mention", eventbus.Event{Type: "mention"}, TypeMention, "You were mentioned"},
		{"invite", eventbus.Event{Type: "invite"}, TypeInvite, "You're invited"},
		{"unknown", eventbus.Event{Type: "unknown"}, "", ""},
		{"thread.updated no match", eventbus.Event{Type: "thread.updated", Payload: map[string]any{"other": "x"}}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifType, title, _ := mapEventToNotification(tt.event)
			assert.Equal(t, tt.wantType, notifType)
			assert.Equal(t, tt.wantTitle, title)
		})
	}
}

func TestExtractPayloadString(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		field   string
		want    string
		ok      bool
	}{
		{"valid", map[string]any{"k": "v"}, "k", "v", true},
		{"missing", map[string]any{}, "k", "", false},
		{"nil", nil, "k", "", false},
		{"non-string", map[string]any{"k": 1}, "k", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractPayloadString(tt.payload, tt.field)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestExtractPayloadStrings(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		field   string
		want    []string
		ok      bool
	}{
		{"[]any", map[string]any{"k": []any{"a", "b"}}, "k", []string{"a", "b"}, true},
		{"[]string", map[string]any{"k": []string{"a", "b"}}, "k", []string{"a", "b"}, true},
		{"csv", map[string]any{"k": "a,b"}, "k", []string{"a", "b"}, true},
		{"missing", map[string]any{}, "k", nil, false},
		{"nil", nil, "k", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractPayloadStrings(tt.payload, tt.field)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestHasPayloadField(t *testing.T) {
	assert.True(t, hasPayloadField(map[string]any{"k": "v"}, "k"))
	assert.False(t, hasPayloadField(map[string]any{"k": "v"}, "other"))
	assert.False(t, hasPayloadField(nil, "k"))
	assert.False(t, hasPayloadField("string", "k"))
}

// --- Handler tests ---

func notifRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/notifications", h.List)
	r.Patch("/notifications/{id}/read", h.MarkRead)
	r.Post("/notifications/mark-all-read", h.MarkAllRead)
	r.Get("/notifications/preferences", h.GetPreferences)
	r.Put("/notifications/preferences", h.UpdatePreferences)
	return r
}

func authRequest(method, url, body, userID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	if userID != "" {
		ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: userID})
		req = req.WithContext(ctx)
	}
	return req
}

func TestHandler_List(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &models.Notification{
			UserID: "handler_user", Type: TypeNewMessage, Title: "Msg",
		}))
	}

	h := NewHandler(repo)
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("GET", "/notifications", "", "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	data := result["data"].([]any)
	assert.Len(t, data, 3)
	assert.Equal(t, float64(3), result["unread_count"])
}

func TestHandler_List_NoAuth(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("GET", "/notifications", "", ""))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_MarkRead(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	notif := &models.Notification{UserID: "handler_user", Type: TypeNewMessage, Title: "Msg"}
	require.NoError(t, repo.Create(ctx, notif))

	h := NewHandler(repo)
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PATCH", "/notifications/"+notif.ID+"/read", "", "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, true, result["is_read"])
}

func TestHandler_MarkRead_NotFound(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PATCH", "/notifications/nonexistent/read", "", "handler_user"))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_MarkRead_NoAuth(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PATCH", "/notifications/some-id/read", "", ""))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_MarkAllRead(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		require.NoError(t, repo.Create(ctx, &models.Notification{
			UserID: "handler_user", Type: TypeNewMessage, Title: "Msg",
		}))
	}

	h := NewHandler(repo)
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("POST", "/notifications/mark-all-read", "", "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, float64(2), result["marked_read"])
}

func TestHandler_MarkAllRead_NoAuth(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("POST", "/notifications/mark-all-read", "", ""))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_GetPreferences(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.UpsertPreference(ctx, &models.NotificationPreference{
		UserID: "handler_user", EventType: TypeNewMessage, Channel: ChannelEmail, Enabled: true,
	}))

	h := NewHandler(repo)
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("GET", "/notifications/preferences", "", "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	prefs := result["preferences"].([]any)
	assert.Len(t, prefs, 1)
}

func TestHandler_GetPreferences_NoAuth(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("GET", "/notifications/preferences", "", ""))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_UpdatePreferences(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := NewHandler(repo)
	router := notifRouter(h)

	body := `{"preferences":[{"event_type":"new_message","channel":"email","enabled":false}],"digest":{"frequency":"weekly","enabled":true}}`
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PUT", "/notifications/preferences", body, "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify preference was saved.
	prefs, err := repo.GetPreferences(context.Background(), "handler_user")
	require.NoError(t, err)
	assert.Len(t, prefs, 1)
	assert.False(t, prefs[0].Enabled)

	// Verify digest schedule.
	sched, err := repo.GetDigestSchedule(context.Background(), "handler_user")
	require.NoError(t, err)
	require.NotNil(t, sched)
	assert.Equal(t, "weekly", sched.Frequency)
	assert.True(t, sched.Enabled)
}

func TestHandler_UpdatePreferences_NoAuth(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PUT", "/notifications/preferences", "{}", ""))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_UpdatePreferences_InvalidBody(t *testing.T) {
	db := testDB(t)
	h := NewHandler(NewRepository(db))
	router := notifRouter(h)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PUT", "/notifications/preferences", "not-json", "handler_user"))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_UpdatePreferences_InvalidFrequency(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := NewHandler(repo)
	router := notifRouter(h)

	body := `{"preferences":[],"digest":{"frequency":"monthly","enabled":true}}`
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PUT", "/notifications/preferences", body, "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Invalid frequency defaults to daily.
	sched, _ := repo.GetDigestSchedule(context.Background(), "handler_user")
	require.NotNil(t, sched)
	assert.Equal(t, "daily", sched.Frequency)
}

func TestHandler_UpdatePreferences_EmptyPref(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	h := NewHandler(repo)
	router := notifRouter(h)

	// Preferences with empty event_type/channel should be skipped.
	body := `{"preferences":[{"event_type":"","channel":"","enabled":false}]}`
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authRequest("PUT", "/notifications/preferences", body, "handler_user"))
	assert.Equal(t, http.StatusOK, rec.Code)

	prefs, _ := repo.GetPreferences(context.Background(), "handler_user")
	assert.Empty(t, prefs)
}

// --- Fuzz tests ---

func FuzzNotificationPayload(f *testing.F) {
	f.Add("user1", "new_message", "Title", "Body", "message", "msg1")
	f.Add("", "", "", "", "", "")
	f.Add("u", "t", "T", "B", "e", "i")
	f.Add("user-with-special-chars!@#$", "type", "title<script>", "body", "entity", "id")

	f.Fuzz(func(t *testing.T, userID, notifType, title, body, entityType, entityID string) {
		input := NotificationInput{
			UserID:     userID,
			Type:       notifType,
			Title:      title,
			Body:       body,
			EntityType: entityType,
			EntityID:   entityID,
		}
		// Should not panic.
		_ = input.Validate()

		// Notification JSON roundtrip.
		notif := &models.Notification{
			UserID:     userID,
			Type:       notifType,
			Title:      title,
			Body:       body,
			EntityType: entityType,
			EntityID:   entityID,
		}
		data, err := json.Marshal(notif)
		if err == nil {
			var decoded models.Notification
			_ = json.Unmarshal(data, &decoded)
		}
	})
}
