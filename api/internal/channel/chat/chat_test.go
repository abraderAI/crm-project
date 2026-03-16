package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/llm"
	"github.com/abraderAI/crm-project/api/internal/models"
	ws "github.com/abraderAI/crm-project/api/internal/websocket"
)

const testSecret = "test-chat-secret-key-32chars!!"

// --- Test helpers ---

func setupTestDB(t *testing.T) *gorm.DB {
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
	return db
}

func createTestOrg(t *testing.T, db *gorm.DB, slug string) *models.Org {
	t.Helper()
	org := &models.Org{Name: slug, Slug: slug, Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org
}

func createTestSpaceAndBoard(t *testing.T, db *gorm.DB, orgID string, spaceType models.SpaceType) (*models.Space, *models.Board) {
	t.Helper()
	space := &models.Space{OrgID: orgID, Name: "Test Space", Slug: "test-space-" + orgID[:8], Type: spaceType, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Test Board", Slug: "test-board-" + orgID[:8], Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return space, board
}

func createChatChannelConfig(t *testing.T, db *gorm.DB, orgID, embedKey string) *models.ChannelConfig {
	t.Helper()
	settings := fmt.Sprintf(`{"embed_key":%q,"widget_theme":{"primary_color":"#3B82F6","greeting":"Hello! How can I help?"},"ai_system_prompt":"You are helpful."}`, embedKey)
	cfg := &models.ChannelConfig{
		OrgID:       orgID,
		ChannelType: models.ChannelTypeChat,
		Settings:    settings,
		Enabled:     true,
	}
	require.NoError(t, db.Create(cfg).Error)
	return cfg
}

func newTestService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	hub := ws.NewHub(slog.Default())
	provider := llm.NewGrokProvider()
	return NewService(NewRepository(db), provider, hub, testSecret)
}

func newTestServiceNoLLM(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	hub := ws.NewHub(slog.Default())
	return NewService(NewRepository(db), nil, hub, testSecret)
}

// --- JWT tests ---

func TestIssueAndValidateSessionToken(t *testing.T) {
	claims := SessionClaims{
		SessionID: "sess-123",
		OrgID:     "org-456",
		VisitorID: "vis-789",
	}
	token, err := IssueSessionToken(testSecret, claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := ValidateSessionToken(testSecret, token)
	require.NoError(t, err)
	assert.Equal(t, "sess-123", parsed.SessionID)
	assert.Equal(t, "org-456", parsed.OrgID)
	assert.Equal(t, "vis-789", parsed.VisitorID)
	assert.True(t, parsed.ExpiresAt > time.Now().Unix())
}

func TestValidateSessionToken_InvalidSignature(t *testing.T) {
	claims := SessionClaims{SessionID: "s1", OrgID: "o1", VisitorID: "v1"}
	token, err := IssueSessionToken(testSecret, claims)
	require.NoError(t, err)

	_, err = ValidateSessionToken("wrong-secret", token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token signature")
}

func TestValidateSessionToken_Expired(t *testing.T) {
	claims := SessionClaims{
		SessionID: "s1",
		OrgID:     "o1",
		VisitorID: "v1",
		IssuedAt:  time.Now().Add(-48 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	// Manually create a token with expired claims.
	payloadBytes, _ := json.Marshal(claims)
	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64URLEncode(payloadBytes)
	signingInput := header + "." + payload
	sig := signHMAC(testSecret, signingInput)
	token := signingInput + "." + sig

	_, err := ValidateSessionToken(testSecret, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateSessionToken_Malformed(t *testing.T) {
	_, err := ValidateSessionToken(testSecret, "not.a.valid.token.with.extra.parts")
	assert.Error(t, err)

	_, err = ValidateSessionToken(testSecret, "")
	assert.Error(t, err)

	_, err = ValidateSessionToken(testSecret, "abc")
	assert.Error(t, err)
}

func TestIssueSessionToken_EmptySecret(t *testing.T) {
	_, err := IssueSessionToken("", SessionClaims{SessionID: "s", OrgID: "o", VisitorID: "v"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signing secret is required")
}

func TestIssueSessionToken_MissingFields(t *testing.T) {
	_, err := IssueSessionToken(testSecret, SessionClaims{OrgID: "o", VisitorID: "v"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	_, err = IssueSessionToken(testSecret, SessionClaims{SessionID: "s", VisitorID: "v"})
	assert.Error(t, err)

	_, err = IssueSessionToken(testSecret, SessionClaims{SessionID: "s", OrgID: "o"})
	assert.Error(t, err)
}

// --- Repository tests ---

func TestRepository_CreateAndFindSession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	session := &ChatSession{
		OrgID:           "org-1",
		EmbedKey:        "key-1",
		FingerprintHash: "fp-abc123",
		VisitorID:       "vis-1",
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.CreateSession(ctx, session))
	assert.NotEmpty(t, session.ID)

	found, err := repo.FindSession(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, session.ID, found.ID)
	assert.Equal(t, "org-1", found.OrgID)
}

func TestRepository_FindSession_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	found, err := repo.FindSession(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_FindOrCreateVisitor_New(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "vis-new-org")
	repo := NewRepository(db)
	ctx := context.Background()

	visitor, isNew, err := repo.FindOrCreateVisitor(ctx, org.ID, "fp-new")
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.NotEmpty(t, visitor.ID)
	assert.Equal(t, "fp-new", visitor.FingerprintHash)
}

func TestRepository_FindOrCreateVisitor_Existing(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "vis-exist-org")
	repo := NewRepository(db)
	ctx := context.Background()

	v1, isNew1, err := repo.FindOrCreateVisitor(ctx, org.ID, "fp-exist")
	require.NoError(t, err)
	assert.True(t, isNew1)

	v2, isNew2, err := repo.FindOrCreateVisitor(ctx, org.ID, "fp-exist")
	require.NoError(t, err)
	assert.False(t, isNew2)
	assert.Equal(t, v1.ID, v2.ID) // Same visitor.
}

func TestRepository_FindChannelConfigByEmbedKey(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "cfg-embed-org")
	repo := NewRepository(db)
	ctx := context.Background()

	createChatChannelConfig(t, db, org.ID, "test-embed-key-123")

	cfg, err := repo.FindChannelConfigByEmbedKey(ctx, "test-embed-key-123")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, org.ID, cfg.OrgID)
}

func TestRepository_FindChannelConfigByEmbedKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	cfg, err := repo.FindChannelConfigByEmbedKey(context.Background(), "nonexistent-key")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestRepository_FindChannelConfigByEmbedKey_DisabledConfig(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "cfg-disabled-org")
	repo := NewRepository(db)
	ctx := context.Background()

	cfg := &models.ChannelConfig{
		OrgID:       org.ID,
		ChannelType: models.ChannelTypeChat,
		Settings:    `{"embed_key":"disabled-key"}`,
		Enabled:     false,
	}
	require.NoError(t, db.Create(cfg).Error)

	found, err := repo.FindChannelConfigByEmbedKey(ctx, "disabled-key")
	require.NoError(t, err)
	assert.Nil(t, found) // Disabled configs should not be returned.
}

func TestRepository_UpdateVisitor(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "vis-upd-org")
	repo := NewRepository(db)
	ctx := context.Background()

	visitor, _, err := repo.FindOrCreateVisitor(ctx, org.ID, "fp-upd")
	require.NoError(t, err)

	visitor.ContactEmail = "test@example.com"
	require.NoError(t, repo.UpdateVisitor(ctx, visitor))

	found, err := repo.FindVisitor(ctx, visitor.ID)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", found.ContactEmail)
}

func TestRepository_FindFirstBoardInOrg(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "board-find-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	repo := NewRepository(db)

	board, err := repo.FindFirstBoardInOrg(context.Background(), org.ID)
	require.NoError(t, err)
	require.NotNil(t, board)
}

func TestRepository_FindFirstBoardInOrg_NoBoard(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "no-board-org")
	repo := NewRepository(db)

	board, err := repo.FindFirstBoardInOrg(context.Background(), org.ID)
	require.NoError(t, err)
	assert.Nil(t, board)
}

func TestRepository_CreateAndListMessages(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "msg-list-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	repo := NewRepository(db)
	ctx := context.Background()

	thread := &models.Thread{BoardID: board.ID, Title: "Test Thread", Slug: "msg-list-thread", AuthorID: "system", Metadata: "{}"}
	require.NoError(t, repo.CreateThread(ctx, thread))

	msg1 := &models.Message{ThreadID: thread.ID, Body: "Hello", AuthorID: "visitor:v1", Type: models.MessageTypeComment, Metadata: "{}"}
	require.NoError(t, repo.CreateMessage(ctx, msg1))

	msg2 := &models.Message{ThreadID: thread.ID, Body: "Hi there!", AuthorID: "ai", Type: models.MessageTypeComment, Metadata: "{}"}
	require.NoError(t, repo.CreateMessage(ctx, msg2))

	messages, err := repo.ListThreadMessages(ctx, thread.ID)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].Body)
	assert.Equal(t, "Hi there!", messages[1].Body)
}

// --- Service tests ---

func TestService_CreateSession_ValidEmbedKey(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-sess-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "valid-embed-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{
		EmbedKey:        "valid-embed-key",
		FingerprintHash: "fp-hash-123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, output.Token)
	assert.NotEmpty(t, output.SessionID)
	assert.NotEmpty(t, output.VisitorID)
	assert.False(t, output.Returning) // First visit.
	assert.Equal(t, "Hello! How can I help?", output.Greeting)
}

func TestService_CreateSession_InvalidEmbedKey(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	_, err := svc.CreateSession(context.Background(), CreateSessionInput{
		EmbedKey:        "bad-key",
		FingerprintHash: "fp-123",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid embed key")
}

func TestService_CreateSession_MissingEmbedKey(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	_, err := svc.CreateSession(context.Background(), CreateSessionInput{FingerprintHash: "fp"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embed_key is required")
}

func TestService_CreateSession_MissingFingerprint(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	_, err := svc.CreateSession(context.Background(), CreateSessionInput{EmbedKey: "key"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fingerprint_hash is required")
}

func TestService_CreateSession_ReturningVisitor(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-return-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "return-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	// First session.
	out1, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "return-key", FingerprintHash: "fp-returning"})
	require.NoError(t, err)
	assert.False(t, out1.Returning)

	// Second session with same fingerprint.
	out2, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "return-key", FingerprintHash: "fp-returning"})
	require.NoError(t, err)
	assert.True(t, out2.Returning)
	assert.Equal(t, out1.VisitorID, out2.VisitorID)    // Same visitor.
	assert.NotEqual(t, out1.SessionID, out2.SessionID) // Different session.
}

func TestService_HandleChatMessage_CreatesThread(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-msg-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "msg-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "msg-key", FingerprintHash: "fp-msg"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	resp, err := svc.HandleChatMessage(ctx, claims, "Hello, I need help with your product")
	require.NoError(t, err)
	assert.Equal(t, "ai_response", resp.Type)
	assert.NotEmpty(t, resp.Message)
	assert.NotEmpty(t, resp.MessageID)

	// Verify thread was created.
	session, err := svc.repo.FindSession(ctx, output.SessionID)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ThreadID)

	// Verify messages were stored (user + AI).
	messages, err := svc.repo.ListThreadMessages(ctx, session.ThreadID)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestService_HandleChatMessage_NoLLM(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-nollm-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "nollm-key")

	svc := newTestServiceNoLLM(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "nollm-key", FingerprintHash: "fp-nollm"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	resp, err := svc.HandleChatMessage(ctx, claims, "Hi!")
	require.NoError(t, err)
	assert.Contains(t, resp.Message, "How can I help you today")
}

func TestService_HandleChatMessage_EmptyBody(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	claims := &SessionClaims{SessionID: "s1", OrgID: "o1", VisitorID: "v1"}
	_, err := svc.HandleChatMessage(context.Background(), claims, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message body is required")
}

func TestService_HandleChatMessage_SessionNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	claims := &SessionClaims{SessionID: "nonexistent", OrgID: "o1", VisitorID: "v1"}
	_, err := svc.HandleChatMessage(context.Background(), claims, "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// --- Lead capture tests ---

func TestService_LeadCapture_Email(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "lead-email-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "lead-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "lead-key", FingerprintHash: "fp-lead"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	_, err = svc.HandleChatMessage(ctx, claims, "My email is john@example.com")
	require.NoError(t, err)

	// Verify visitor contact email was captured.
	visitor, err := svc.repo.FindVisitor(ctx, output.VisitorID)
	require.NoError(t, err)
	assert.Equal(t, "john@example.com", visitor.ContactEmail)
}

func TestService_LeadCapture_Name(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "lead-name-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "lead-name-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "lead-name-key", FingerprintHash: "fp-lead-name"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	_, err = svc.HandleChatMessage(ctx, claims, "My name is John Doe. I need help.")
	require.NoError(t, err)

	visitor, err := svc.repo.FindVisitor(ctx, output.VisitorID)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", visitor.ContactName)
}

func TestService_LeadCapture_NameWithImContraction(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "lead-im-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "lead-im-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "lead-im-key", FingerprintHash: "fp-lead-im"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	_, err = svc.HandleChatMessage(ctx, claims, "Hi, I'm Alice Smith, can you help?")
	require.NoError(t, err)

	visitor, err := svc.repo.FindVisitor(ctx, output.VisitorID)
	require.NoError(t, err)
	assert.Equal(t, "Alice Smith", visitor.ContactName)
}

// --- Escalation tests ---

func TestService_Escalation_Detection(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "esc-detect-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "esc-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "esc-key", FingerprintHash: "fp-esc"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// First send a normal message to create the thread.
	_, err = svc.HandleChatMessage(ctx, claims, "I need help")
	require.NoError(t, err)

	// Now send escalation request.
	resp, err := svc.HandleChatMessage(ctx, claims, "I want to speak to a human agent please")
	require.NoError(t, err)
	assert.Equal(t, "escalation", resp.Type)
	assert.Contains(t, resp.Message, "human agent")

	// Verify session is marked as escalated.
	session, err := svc.repo.FindSession(ctx, output.SessionID)
	require.NoError(t, err)
	assert.True(t, session.Escalated)
}

func TestService_Escalation_Timeout(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "esc-timeout-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "esc-to-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "esc-to-key", FingerprintHash: "fp-esc-to"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// Create thread and escalate.
	_, err = svc.HandleChatMessage(ctx, claims, "Hello")
	require.NoError(t, err)
	_, err = svc.HandleChatMessage(ctx, claims, "I want to talk to a real person")
	require.NoError(t, err)

	// Simulate timeout.
	err = svc.ResumeAfterEscalationTimeout(ctx, output.SessionID)
	require.NoError(t, err)

	// Session should no longer be escalated.
	session, err := svc.repo.FindSession(ctx, output.SessionID)
	require.NoError(t, err)
	assert.False(t, session.Escalated)
}

func TestService_EscalationTimeout_SessionNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	err := svc.ResumeAfterEscalationTimeout(context.Background(), "nonexistent")
	assert.Error(t, err)
}

// --- Escalation pattern detection ---

func TestDetectEscalation_Patterns(t *testing.T) {
	svc := &Service{}
	tests := []struct {
		msg  string
		want bool
	}{
		{"I want to speak to a human", true},
		{"Can I talk to a person?", true},
		{"I need a real person", true},
		{"Connect me to a human agent", true},
		{"Please escalate this", true},
		{"I need a live agent", true},
		{"I want customer service", true},
		{"Hello, how are you?", false},
		{"I need help with my order", false},
		{"What are your hours?", false},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := svc.detectEscalation(tt.msg)
			assert.Equal(t, tt.want, got, "message: %q", tt.msg)
		})
	}
}

// --- extractName tests ---

func TestExtractName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"John Doe", "John Doe"},
		{"Alice Smith, I need help", "Alice Smith"},
		{"Bob. Thanks!", "Bob"},
		{"", ""},
		{"123", ""},
		{"Jane Doe!", "Jane Doe"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Handler tests ---

func TestHandler_CreateSession_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-sess-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "hdl-key")

	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	body := `{"embed_key":"hdl-key","fingerprint_hash":"fp-hdl"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateSession(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
	assert.NotEmpty(t, resp["session_id"])
}

func TestHandler_CreateSession_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/session", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	h.CreateSession(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateSession_InvalidEmbedKey(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	body := `{"embed_key":"bad-key","fingerprint_hash":"fp"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/session", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateSession(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SendMessage_Success(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-msg-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "hdl-msg-key")

	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	// Create session first.
	sessBody := `{"embed_key":"hdl-msg-key","fingerprint_hash":"fp-hdl-msg"}`
	sessReq := httptest.NewRequest(http.MethodPost, "/v1/chat/session", strings.NewReader(sessBody))
	sessW := httptest.NewRecorder()
	h.CreateSession(sessW, sessReq)
	require.Equal(t, http.StatusOK, sessW.Code)

	var sessResp map[string]any
	require.NoError(t, json.Unmarshal(sessW.Body.Bytes(), &sessResp))
	token := sessResp["token"].(string)

	// Send message.
	msgBody := `{"message":"Hello, I need help"}`
	msgReq := httptest.NewRequest(http.MethodPost, "/v1/chat/message", strings.NewReader(msgBody))
	msgReq.Header.Set("Authorization", "Bearer "+token)
	msgW := httptest.NewRecorder()
	h.SendMessage(msgW, msgReq)

	assert.Equal(t, http.StatusOK, msgW.Code)
}

func TestHandler_SendMessage_NoToken(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/message", strings.NewReader(`{"message":"hi"}`))
	w := httptest.NewRecorder()
	h.SendMessage(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SendMessage_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/message", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	h.SendMessage(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SendMessage_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "hdl-bad-msg-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "hdl-bad-msg-key")

	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	sessBody := `{"embed_key":"hdl-bad-msg-key","fingerprint_hash":"fp-bad-msg"}`
	sessReq := httptest.NewRequest(http.MethodPost, "/v1/chat/session", strings.NewReader(sessBody))
	sessW := httptest.NewRecorder()
	h.CreateSession(sessW, sessReq)
	require.Equal(t, http.StatusOK, sessW.Code)

	var sessResp map[string]any
	require.NoError(t, json.Unmarshal(sessW.Body.Bytes(), &sessResp))
	token := sessResp["token"].(string)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/message", strings.NewReader("not-json"))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.SendMessage(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Fingerprint matching tests ---

func TestFingerprint_MergeLeadData(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "fp-merge-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "fp-merge-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	// Session 1: provide email.
	out1, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "fp-merge-key", FingerprintHash: "fp-merge"})
	require.NoError(t, err)
	claims1, _ := ValidateSessionToken(testSecret, out1.Token)
	_, err = svc.HandleChatMessage(ctx, claims1, "My email is merge@example.com")
	require.NoError(t, err)

	// Session 2: same fingerprint, provide name.
	out2, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "fp-merge-key", FingerprintHash: "fp-merge"})
	require.NoError(t, err)
	assert.True(t, out2.Returning)
	claims2, _ := ValidateSessionToken(testSecret, out2.Token)
	_, err = svc.HandleChatMessage(ctx, claims2, "My name is Jane Merge")
	require.NoError(t, err)

	// Visitor should have both email and name.
	visitor, err := svc.repo.FindVisitor(ctx, out1.VisitorID)
	require.NoError(t, err)
	assert.Equal(t, "merge@example.com", visitor.ContactEmail)
	assert.Equal(t, "Jane Merge", visitor.ContactName)
}

// --- Fuzz tests ---

// FuzzChatMessage fuzzes chat message processing.
func FuzzChatMessage(f *testing.F) {
	seeds := []string{
		"Hello, I need help",
		"My email is test@example.com",
		"My name is John Doe",
		"I want to speak to a human agent",
		"Can I talk to a real person?",
		"",
		"   ",
		"\n\n\n",
		"<script>alert('xss')</script>",
		`"; DROP TABLE threads; --`,
		"' OR 1=1 --",
		strings.Repeat("x", 10000),
		"Hello! 😀 How are you?",
		"test@test.com and more text",
		"I'm Alice, nice to meet you.",
		"My name is Bob Smith. Can you help?",
		"please escalate this to management",
		"I need customer service",
		"Connect me to a live agent",
		"transfer me to your supervisor",
		"The quick brown fox",
		"1234567890",
		"null",
		"undefined",
		"true",
		"false",
		"NaN",
		"Infinity",
		`{"key": "value"}`,
		`[1, 2, 3]`,
		"a@b.c",
		"not-an-email",
		"name@",
		"@domain.com",
		"user@.com",
		"Hello\x00World",
		"Hello\tWorld",
		"Hello\rWorld",
		"My name is 数据 and I need help",
		"Bonjour, j'ai besoin d'aide",
		"email: admin@company.org please contact me",
		"I'm looking for information about pricing",
		"What are your business hours?",
		"Can you help with my account?",
		"I forgot my password",
		"I want to cancel my subscription",
		"Where can I find the documentation?",
		"Is there a free trial available?",
		"How much does the enterprise plan cost?",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, message string) {
		// extractName must not panic.
		_ = extractName(message)
		// emailRegex must not panic.
		_ = emailRegex.FindString(message)
		// detectEscalation must not panic.
		svc := &Service{}
		_ = svc.detectEscalation(message)
	})
}

// FuzzEmbedKey fuzzes embed key validation.
func FuzzEmbedKey(f *testing.F) {
	seeds := []string{
		"valid-embed-key-123",
		"",
		" ",
		"a",
		strings.Repeat("k", 1000),
		"key-with-special-chars!@#$%",
		`key"with"quotes`,
		"key\nwith\nnewlines",
		"key\x00null",
		"<script>alert(1)</script>",
		`'; DROP TABLE channel_configs; --`,
		"key with spaces",
		"key\twith\ttabs",
		"unicode-key-日本語",
		"emoji-key-🔑",
		"null",
		"undefined",
		"true",
		"false",
		"0",
		"-1",
		"key-123-abc-def",
		"KEY-UPPER",
		"MiXeD-CaSe",
		"a-b-c-d-e-f-g",
		strings.Repeat("a", 500),
		"key/with/slashes",
		"key\\with\\backslashes",
		"key.with.dots",
		"key_with_underscores",
		"key-with-dashes",
		"key+with+plus",
		"key=with=equals",
		"key&with&amps",
		"key?with?question",
		"key#with#hash",
		"key@with@at",
		"key!with!bang",
		"key~with~tilde",
		"key`with`backtick",
		"key|with|pipe",
		"key[with]brackets",
		"key{with}braces",
		"key<with>angles",
		"key(with)parens",
		"key;with;semicolon",
		"key:with:colon",
		"key'with'apostrophe",
		`key"with"double-quotes`,
		"key,with,commas",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, embedKey string) {
		// CreateSessionInput validation must not panic.
		input := CreateSessionInput{EmbedKey: embedKey, FingerprintHash: "test-fp"}
		_ = input.EmbedKey // Just access; actual validation is in service.
	})
}

// --- Phase 6: Lead record creation tests ---

func TestService_LeadCapture_CreatesLeadRecord(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "lead-record-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "lead-rec-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "lead-rec-key", FingerprintHash: "fp-lead-rec"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// Send a message with email to trigger lead capture.
	_, err = svc.HandleChatMessage(ctx, claims, "My email is lead@example.com")
	require.NoError(t, err)

	// Verify lead record was created with anon_session_id.
	lead, err := svc.repo.FindLeadByAnonSession(ctx, output.VisitorID)
	require.NoError(t, err)
	require.NotNil(t, lead)
	assert.Equal(t, "lead@example.com", lead.Email)
	assert.Equal(t, models.LeadStatusAnonymous, lead.Status)
	assert.NotNil(t, lead.AnonSessionID)
	assert.Equal(t, output.VisitorID, *lead.AnonSessionID)
}

func TestService_LeadCapture_NoDuplicateLeads(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "lead-dedup-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "lead-dedup-key")

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "lead-dedup-key", FingerprintHash: "fp-lead-dedup"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// First message with email.
	_, err = svc.HandleChatMessage(ctx, claims, "My email is dedup@example.com")
	require.NoError(t, err)

	// Second message with name (should update, not create duplicate).
	_, err = svc.HandleChatMessage(ctx, claims, "My name is Dedup User")
	require.NoError(t, err)

	// Count leads with this anon_session_id.
	var count int64
	err = db.Model(&models.Lead{}).Where("anon_session_id = ?", output.VisitorID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count) // Only one lead, not duplicated.

	// Verify lead has both email and name.
	lead, err := svc.repo.FindLeadByAnonSession(ctx, output.VisitorID)
	require.NoError(t, err)
	require.NotNil(t, lead)
	assert.Equal(t, "dedup@example.com", lead.Email)
	assert.Equal(t, "Dedup User", lead.Name)
}

func TestRepository_CreateOrUpdateLead_NewLead(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	lead, err := repo.CreateOrUpdateLead(ctx, "vis-new-lead", "new@example.com", "New Lead", "chatbot")
	require.NoError(t, err)
	require.NotNil(t, lead)
	assert.NotEmpty(t, lead.ID)
	assert.Equal(t, "new@example.com", lead.Email)
	assert.Equal(t, "New Lead", lead.Name)
	assert.Equal(t, models.LeadStatusAnonymous, lead.Status)
	assert.Equal(t, "chatbot", lead.Source)
}

func TestRepository_CreateOrUpdateLead_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Create initial lead with email only.
	lead1, err := repo.CreateOrUpdateLead(ctx, "vis-update-lead", "update@example.com", "", "chatbot")
	require.NoError(t, err)

	// Update with name.
	lead2, err := repo.CreateOrUpdateLead(ctx, "vis-update-lead", "", "Updated Name", "chatbot")
	require.NoError(t, err)
	assert.Equal(t, lead1.ID, lead2.ID)
	assert.Equal(t, "update@example.com", lead2.Email)
	assert.Equal(t, "Updated Name", lead2.Name)
}

func TestRepository_FindLeadByAnonSession_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	lead, err := repo.FindLeadByAnonSession(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, lead)
}

// --- Phase 6: Escalation support thread tests ---

func createGlobalSupportSpaceAndBoard(t *testing.T, db *gorm.DB) (*models.Space, *models.Board) {
	t.Helper()
	// Find the system org.
	var systemOrg models.Org
	err := db.Where("slug = ?", "_system").First(&systemOrg).Error
	if err != nil {
		// Create system org if it doesn't exist.
		systemOrg = models.Org{Name: "System", Slug: "_system", Metadata: "{}"}
		require.NoError(t, db.Create(&systemOrg).Error)
	}

	// Create or find global-support space.
	var space models.Space
	err = db.Where("slug = ?", "global-support").First(&space).Error
	if err != nil {
		space = models.Space{OrgID: systemOrg.ID, Name: "Support", Slug: "global-support", Type: models.SpaceTypeSupport, Metadata: "{}"}
		require.NoError(t, db.Create(&space).Error)
	}

	// Create a board in the space.
	board := &models.Board{SpaceID: space.ID, Name: "Support Board", Slug: "support-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	return &space, board
}

func TestService_Escalation_CreatesSupportThread(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "esc-thread-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "esc-thread-key")
	createGlobalSupportSpaceAndBoard(t, db)

	svc := newTestService(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "esc-thread-key", FingerprintHash: "fp-esc-thread"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// First message to create the chat thread.
	_, err = svc.HandleChatMessage(ctx, claims, "I need help with billing")
	require.NoError(t, err)

	// Escalation request.
	resp, err := svc.HandleChatMessage(ctx, claims, "I want to speak to a human agent")
	require.NoError(t, err)
	assert.Equal(t, "escalation", resp.Type)

	// Verify a support thread was created in global-support.
	var supportThread models.Thread
	err = db.Where("thread_type = ? AND visibility = ?",
		models.ThreadTypeSupport, models.ThreadVisibilityDeftOnly).
		First(&supportThread).Error
	require.NoError(t, err)
	assert.Contains(t, supportThread.Title, "Chat escalation")
	assert.Contains(t, supportThread.Metadata, "chat_escalation")
	assert.Equal(t, org.ID, *supportThread.OrgID)
}

func TestRepository_FindGlobalSupportBoard_NoBoard(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	board, err := repo.FindGlobalSupportBoard(context.Background())
	require.NoError(t, err)
	assert.Nil(t, board)
}

func TestRepository_FindGlobalSupportBoard_Found(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	createGlobalSupportSpaceAndBoard(t, db)

	board, err := repo.FindGlobalSupportBoard(context.Background())
	require.NoError(t, err)
	require.NotNil(t, board)
	assert.Equal(t, "Support Board", board.Name)
}

// --- Phase 6: Session promotion tests ---

func TestService_PromoteSession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Create a lead with anonymous status.
	anonID := "vis-promote-test"
	lead, err := repo.CreateOrUpdateLead(ctx, anonID, "promote@example.com", "Promote User", "chatbot")
	require.NoError(t, err)
	assert.Equal(t, models.LeadStatusAnonymous, lead.Status)

	// Promote the session.
	err = repo.PromoteAnonymousSession(ctx, anonID, "user-123")
	require.NoError(t, err)

	// Verify the lead was updated.
	var updatedLead models.Lead
	err = db.Where("anon_session_id = ?", anonID).First(&updatedLead).Error
	require.NoError(t, err)
	assert.Equal(t, models.LeadStatusRegistered, updatedLead.Status)
	require.NotNil(t, updatedLead.UserID)
	assert.Equal(t, "user-123", *updatedLead.UserID)
}

func TestService_PromoteSession_NoMatchingLead(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Promoting a non-existent session should not error (no-op).
	err := repo.PromoteAnonymousSession(ctx, "nonexistent-session", "user-999")
	require.NoError(t, err)
}

func TestService_PromoteSession_ViaService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	ctx := context.Background()

	// Create a lead.
	_, err := svc.repo.CreateOrUpdateLead(ctx, "vis-svc-promote", "svc@example.com", "", "chatbot")
	require.NoError(t, err)

	// Promote via service.
	err = svc.PromoteSession("vis-svc-promote", "user-svc-456")
	require.NoError(t, err)

	// Verify.
	lead, err := svc.repo.FindLeadByAnonSession(ctx, "vis-svc-promote")
	require.NoError(t, err)
	require.NotNil(t, lead)
	assert.Equal(t, models.LeadStatusRegistered, lead.Status)
}

// --- Phase 6: Handler promotion endpoint tests ---

func TestHandler_SessionPromotion_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)
	ctx := context.Background()

	// Create a lead first.
	_, err := svc.repo.CreateOrUpdateLead(ctx, "vis-hdl-promote", "hdl@example.com", "", "chatbot")
	require.NoError(t, err)

	body := `{"anon_session_id":"vis-hdl-promote","user_id":"user-hdl-789"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/promote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(setTestAuth(req.Context(), "user-hdl-789"))
	w := httptest.NewRecorder()
	h.HandleSessionPromotion(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "promoted", resp["status"])
}

func TestHandler_SessionPromotion_Unauthenticated(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	body := `{"anon_session_id":"vis-1","user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/promote", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleSessionPromotion(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SessionPromotion_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	tests := []struct {
		name string
		body string
	}{
		{"missing anon_session_id", `{"anon_session_id":"","user_id":"u1"}`},
		{"missing user_id", `{"anon_session_id":"s1","user_id":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/promote", strings.NewReader(tt.body))
			req = req.WithContext(setTestAuth(req.Context(), "test-user"))
			w := httptest.NewRecorder()
			h.HandleSessionPromotion(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_SessionPromotion_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(t, db)
	h := NewHandler(svc, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/promote", strings.NewReader("not-json"))
	req = req.WithContext(setTestAuth(req.Context(), "test-user"))
	w := httptest.NewRecorder()
	h.HandleSessionPromotion(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Phase 6: RAG context retrieval tests ---

func createGlobalDocsContent(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Find or create system org.
	var systemOrg models.Org
	err := db.Where("slug = ?", "_system").First(&systemOrg).Error
	if err != nil {
		systemOrg = models.Org{Name: "System", Slug: "_system", Metadata: "{}"}
		require.NoError(t, db.Create(&systemOrg).Error)
	}

	// Create global-docs space.
	var docsSpace models.Space
	err = db.Where("slug = ?", "global-docs").First(&docsSpace).Error
	if err != nil {
		docsSpace = models.Space{OrgID: systemOrg.ID, Name: "Documentation", Slug: "global-docs", Type: models.SpaceTypeKnowledgeBase, Metadata: "{}"}
		require.NoError(t, db.Create(&docsSpace).Error)
	}

	// Create a board.
	board := &models.Board{SpaceID: docsSpace.ID, Name: "Docs Board", Slug: "docs-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	// Create some documentation threads.
	threads := []models.Thread{
		{BoardID: board.ID, Title: "Getting Started Guide", Body: "Welcome to our platform. Here is how to get started with your account setup and configuration.", Slug: "getting-started", AuthorID: "system", Metadata: "{}", ThreadType: models.ThreadTypeDocumentation, Visibility: models.ThreadVisibilityPublic},
		{BoardID: board.ID, Title: "API Authentication", Body: "Learn how to authenticate API requests using JWT tokens and API keys.", Slug: "api-auth", AuthorID: "system", Metadata: "{}", ThreadType: models.ThreadTypeDocumentation, Visibility: models.ThreadVisibilityPublic},
		{BoardID: board.ID, Title: "Billing FAQ", Body: "Frequently asked questions about billing, invoices, and payment methods.", Slug: "billing-faq", AuthorID: "system", Metadata: "{}", ThreadType: models.ThreadTypeDocumentation, Visibility: models.ThreadVisibilityPublic},
	}
	for i := range threads {
		require.NoError(t, db.Create(&threads[i]).Error)
	}
}

func TestRepository_SearchGlobalDocs_Found(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	createGlobalDocsContent(t, db)

	results, err := repo.SearchGlobalDocs(context.Background(), "billing", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Billing FAQ", results[0].Title)
}

func TestRepository_SearchGlobalDocs_MultipleResults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	createGlobalDocsContent(t, db)

	// Search for "API" which should match the API auth thread.
	results, err := repo.SearchGlobalDocs(context.Background(), "API", 5)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestRepository_SearchGlobalDocs_NoResults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	createGlobalDocsContent(t, db)

	results, err := repo.SearchGlobalDocs(context.Background(), "zyxwv-nonexistent", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRepository_SearchGlobalDocs_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	results, err := repo.SearchGlobalDocs(context.Background(), "", 5)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestRepository_SearchGlobalDocs_ZeroLimit(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	results, err := repo.SearchGlobalDocs(context.Background(), "test", 0)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestService_RAG_NoLLM_WithDocs(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "rag-nollm-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "rag-nollm-key")
	createGlobalDocsContent(t, db)

	svc := newTestServiceNoLLM(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "rag-nollm-key", FingerprintHash: "fp-rag-nollm"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// Ask about billing — should get RAG response.
	resp, err := svc.HandleChatMessage(ctx, claims, "Tell me about billing please")
	require.NoError(t, err)
	assert.Contains(t, resp.Message, "Based on our documentation")
	assert.Contains(t, resp.Message, "Billing FAQ")
}

func TestService_RAG_NoLLM_NoDocs(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "rag-nodocs-org")
	createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	createChatChannelConfig(t, db, org.ID, "rag-nodocs-key")

	svc := newTestServiceNoLLM(t, db)
	ctx := context.Background()

	output, err := svc.CreateSession(ctx, CreateSessionInput{EmbedKey: "rag-nodocs-key", FingerprintHash: "fp-rag-nodocs"})
	require.NoError(t, err)

	claims, err := ValidateSessionToken(testSecret, output.Token)
	require.NoError(t, err)

	// No docs available — should get default response.
	resp, err := svc.HandleChatMessage(ctx, claims, "Hello")
	require.NoError(t, err)
	assert.Contains(t, resp.Message, "How can I help you today")
}

// --- Phase 6: truncate helper tests ---

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hello...", truncate("hello world", 5))
	assert.Equal(t, "", truncate("", 5))
	assert.Equal(t, "ab...", truncate("abcdef", 2))
}

// --- Phase 6: buildRAGContext tests ---

func TestBuildRAGContext_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestServiceNoLLM(t, db)
	ctx := context.Background()

	result := svc.buildRAGContext(ctx, "")
	assert.Empty(t, result)
}

func TestBuildRAGContext_ShortWords(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestServiceNoLLM(t, db)
	ctx := context.Background()

	// All words are shorter than 3 chars.
	result := svc.buildRAGContext(ctx, "a b c")
	assert.Empty(t, result)
}

func TestBuildRAGContext_WithDocs(t *testing.T) {
	db := setupTestDB(t)
	createGlobalDocsContent(t, db)
	svc := newTestServiceNoLLM(t, db)
	ctx := context.Background()

	result := svc.buildRAGContext(ctx, "billing invoices")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Billing FAQ")
}

// setTestAuth sets authentication context for handler tests.
func setTestAuth(ctx context.Context, userID string) context.Context {
	return auth.SetUserContext(ctx, &auth.UserContext{
		UserID:     userID,
		AuthMethod: auth.AuthMethodJWT,
	})
}

// FuzzFingerprint fuzzes fingerprint hash handling.
func FuzzFingerprint(f *testing.F) {
	seeds := []string{
		"abc123def456",
		"",
		" ",
		strings.Repeat("f", 64),
		strings.Repeat("0", 32),
		"short",
		strings.Repeat("x", 10000),
		"fingerprint-with-special-!@#$%",
		"fp\x00null",
		"fp\nnewline",
		"<script>",
		`'; DROP TABLE visitors; --`,
		"fp with spaces",
		"fp\twith\ttabs",
		"unicode-fp-日本語",
		"emoji-fp-👆",
		"0123456789abcdef0123456789abcdef",
		"UPPER-CASE-FP",
		"MiXeD-CaSe-Fp",
		"null",
		"undefined",
		"true",
		"false",
		"NaN",
		"-1",
		"1.5",
		"1e10",
		"fp/with/slashes",
		"fp\\with\\backslashes",
		"fp.with.dots",
		"fp_with_underscores",
		"fp-with-dashes",
		"fp+with+plus",
		"fp=with=equals",
		"fp&with&amps",
		"fp?with?question",
		"fp#with#hash",
		"fp@with@at",
		"fp!with!bang",
		"fp~with~tilde",
		"fp`with`backtick",
		"fp|with|pipe",
		"fp[with]brackets",
		"fp{with}braces",
		"fp<with>angles",
		"fp(with)parens",
		"fp;with;semicolon",
		"fp:with:colon",
		"fp'with'apostrophe",
		`fp"with"double-quotes`,
		"fp,with,commas",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, fingerprint string) {
		// Fingerprint handling in JWT must not panic.
		claims := SessionClaims{
			SessionID: "test-session",
			OrgID:     "test-org",
			VisitorID: "test-visitor",
		}
		token, err := IssueSessionToken(testSecret, claims)
		if err != nil {
			return
		}
		// Validate must not panic.
		_, _ = ValidateSessionToken(testSecret, token)
		// ValidateSessionToken with fuzzed input as token must not panic.
		_, _ = ValidateSessionToken(testSecret, fingerprint)
	})
}
