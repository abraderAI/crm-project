package email

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/mail"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/channel"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
)

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

func makeMessage(headers map[string]string, body string) *mail.Message {
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k + ": " + v + "\r\n")
	}
	sb.WriteString("\r\n")
	sb.WriteString(body)
	msg, _ := mail.ReadMessage(strings.NewReader(sb.String()))
	return msg
}

func makeMultipartMessage(headers map[string]string, boundary string, parts []string) *mail.Message {
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k + ": " + v + "\r\n")
	}
	sb.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n")
	sb.WriteString("\r\n")
	for _, part := range parts {
		sb.WriteString("--" + boundary + "\r\n")
		sb.WriteString(part)
		sb.WriteString("\r\n")
	}
	sb.WriteString("--" + boundary + "--\r\n")
	msg, _ := mail.ReadMessage(strings.NewReader(sb.String()))
	return msg
}

// --- Parser tests ---

func TestParseEmail_PlainText(t *testing.T) {
	msg := makeMessage(map[string]string{
		"Message-ID":  "<msg001@example.com>",
		"From":        "Alice <alice@example.com>",
		"To":          "bob@example.com",
		"Subject":     "Hello Bob",
		"In-Reply-To": "<parent@example.com>",
		"References":  "<ref1@example.com> <ref2@example.com>",
	}, "Hello, this is a plain text email.")

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Equal(t, "msg001@example.com", parsed.MessageID)
	assert.Equal(t, "parent@example.com", parsed.InReplyTo)
	assert.Equal(t, []string{"ref1@example.com", "ref2@example.com"}, parsed.References)
	assert.Equal(t, "alice@example.com", parsed.From)
	assert.Equal(t, []string{"bob@example.com"}, parsed.To)
	assert.Equal(t, "Hello Bob", parsed.Subject)
	assert.Equal(t, "Hello, this is a plain text email.", parsed.Body)
	assert.Equal(t, "Hello, this is a plain text email.", parsed.PlainBody)
	assert.Empty(t, parsed.HTMLBody)
	assert.Empty(t, parsed.Attachments)
}

func TestParseEmail_HTMLOnly(t *testing.T) {
	msg := makeMessage(map[string]string{
		"From":         "sender@example.com",
		"Content-Type": "text/html; charset=utf-8",
	}, "<html><body><p>Hello &amp; welcome</p></body></html>")

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Contains(t, parsed.HTMLBody, "Hello &amp; welcome")
	assert.Contains(t, parsed.Body, "Hello & welcome")
	assert.Empty(t, parsed.PlainBody)
}

func TestParseEmail_Multipart(t *testing.T) {
	boundary := "testboundary123"
	parts := []string{
		"Content-Type: text/plain\r\n\r\nPlain text body",
		"Content-Type: text/html\r\n\r\n<p>HTML body</p>",
	}
	msg := makeMultipartMessage(map[string]string{
		"From":    "sender@example.com",
		"Subject": "Multipart Test",
	}, boundary, parts)

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Equal(t, "Plain text body", parsed.PlainBody)
	assert.Contains(t, parsed.HTMLBody, "HTML body")
	assert.Equal(t, "Plain text body", parsed.Body) // Prefers plain text.
}

func TestParseEmail_WithAttachment(t *testing.T) {
	boundary := "attboundary456"
	parts := []string{
		"Content-Type: text/plain\r\n\r\nBody text",
		"Content-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"document.pdf\"\r\n\r\nPDF content here",
	}
	msg := makeMultipartMessage(map[string]string{
		"From":    "sender@example.com",
		"Subject": "With Attachment",
	}, boundary, parts)

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Equal(t, "Body text", parsed.PlainBody)
	require.Len(t, parsed.Attachments, 1)
	assert.Equal(t, "document.pdf", parsed.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", parsed.Attachments[0].ContentType)
	assert.False(t, parsed.Attachments[0].IsInline)
}

func TestParseEmail_InlineImage(t *testing.T) {
	boundary := "inlineboundary789"
	parts := []string{
		"Content-Type: text/plain\r\n\r\nBody with image",
		"Content-Type: image/png\r\nContent-Disposition: inline\r\nContent-ID: <img001>\r\n\r\nPNG data bytes here",
	}
	msg := makeMultipartMessage(map[string]string{
		"From": "sender@example.com",
	}, boundary, parts)

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	require.Len(t, parsed.Attachments, 1)
	assert.True(t, parsed.Attachments[0].IsInline)
	assert.Equal(t, "img001", parsed.Attachments[0].ContentID)
}

func TestParseEmail_Base64Encoding(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("Decoded body content"))
	msg := makeMessage(map[string]string{
		"From":                      "sender@example.com",
		"Content-Type":              "text/plain",
		"Content-Transfer-Encoding": "base64",
	}, encoded)

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Equal(t, "Decoded body content", parsed.PlainBody)
}

func TestParseEmail_QuotedPrintable(t *testing.T) {
	msg := makeMessage(map[string]string{
		"From":                      "sender@example.com",
		"Content-Type":              "text/plain",
		"Content-Transfer-Encoding": "quoted-printable",
	}, "Hello =C3=A9 world")

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Contains(t, parsed.PlainBody, "Hello")
	assert.Contains(t, parsed.PlainBody, "world")
}

func TestParseEmail_NilMessage(t *testing.T) {
	_, err := ParseEmail(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestParseEmail_EmptyFrom(t *testing.T) {
	msg := makeMessage(map[string]string{
		"Subject": "No From",
	}, "Body")

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Empty(t, parsed.From)
}

func TestParseEmail_MultipleRecipients(t *testing.T) {
	msg := makeMessage(map[string]string{
		"From": "sender@example.com",
		"To":   "a@example.com, b@example.com",
		"Cc":   "c@example.com",
	}, "Body")

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Len(t, parsed.To, 2)
	assert.Len(t, parsed.CC, 1)
	assert.Contains(t, parsed.To, "a@example.com")
	assert.Contains(t, parsed.To, "b@example.com")
}

func TestParseEmail_ReplyChain(t *testing.T) {
	msg := makeMessage(map[string]string{
		"From":        "reply@example.com",
		"Message-ID":  "<reply-001@example.com>",
		"In-Reply-To": "<original-001@example.com>",
		"References":  "<original-001@example.com> <intermediate-001@example.com>",
		"Subject":     "Re: Original Subject",
	}, "Reply body text")

	parsed, err := ParseEmail(msg)
	require.NoError(t, err)
	assert.Equal(t, "reply-001@example.com", parsed.MessageID)
	assert.Equal(t, "original-001@example.com", parsed.InReplyTo)
	assert.Len(t, parsed.References, 2)
	assert.Equal(t, "original-001@example.com", parsed.References[0])
	assert.Equal(t, "intermediate-001@example.com", parsed.References[1])
}

// --- HTML stripping tests ---

func TestStripHTML_Basic(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"simple tags", "<p>Hello</p>", "Hello"},
		{"entities", "&amp; &lt; &gt;", "& < >"},
		{"style block", "<style>body{color:red}</style>Text", "Text"},
		{"script block", "<script>alert('xss')</script>Safe text", "Safe text"},
		{"br tag", "Line1<br>Line2", "Line1\nLine2"},
		{"nested", "<div><p>Nested <b>bold</b></p></div>", "Nested bold"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTML(tt.input)
			assert.Contains(t, result, tt.expect)
		})
	}
}

// --- Helper function tests ---

func TestCleanHeaderValue(t *testing.T) {
	tests := []struct{ input, expect string }{
		{"<msg@example.com>", "msg@example.com"},
		{"msg@example.com", "msg@example.com"},
		{"  <msg@example.com>  ", "msg@example.com"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expect, cleanHeaderValue(tt.input))
	}
}

func TestParseReferences(t *testing.T) {
	assert.Nil(t, parseReferences(""))
	assert.Equal(t, []string{"a@b.com"}, parseReferences("<a@b.com>"))
	assert.Equal(t, []string{"a@b.com", "c@d.com"}, parseReferences("<a@b.com> <c@d.com>"))
}

func TestExtractEmailAddress(t *testing.T) {
	assert.Equal(t, "alice@example.com", extractEmailAddress("Alice <alice@example.com>"))
	assert.Equal(t, "alice@example.com", extractEmailAddress("alice@example.com"))
	assert.Equal(t, "", extractEmailAddress(""))
}

func TestParseAddressList(t *testing.T) {
	assert.Nil(t, parseAddressList(""))
	addrs := parseAddressList("a@b.com, c@d.com")
	assert.Len(t, addrs, 2)
}

// --- Thread matcher tests ---

func TestThreadMatcher_MatchByMessageID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "tm-msgid-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	// Create existing thread with message_ids in metadata.
	existingThread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Existing Thread",
		Slug:     "existing-thread",
		Metadata: `{"message_ids":["original@example.com"],"contact_email":"alice@example.com"}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(existingThread).Error)

	matcher := NewThreadMatcher(db)
	parsed := &ParsedEmail{
		InReplyTo: "original@example.com",
		From:      "bob@example.com",
		Subject:   "Re: Test",
		Body:      "Reply body",
	}

	result, err := matcher.Match(context.Background(), org.ID, parsed, models.RoutingActionSalesLead)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, "message_id", result.MatchBy)
	assert.Equal(t, existingThread.ID, result.Thread.ID)
}

func TestThreadMatcher_MatchByReferences(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "tm-refs-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	existingThread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Thread with References",
		Slug:     "thread-refs",
		Metadata: `{"message_ids":["ref-msg-001@example.com"]}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(existingThread).Error)

	matcher := NewThreadMatcher(db)
	parsed := &ParsedEmail{
		References: []string{"ref-msg-001@example.com"},
		From:       "carol@example.com",
		Subject:    "Re: Referenced",
		Body:       "Response",
	}

	result, err := matcher.Match(context.Background(), org.ID, parsed, models.RoutingActionSalesLead)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, "message_id", result.MatchBy)
}

func TestThreadMatcher_MatchBySenderEmail(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "tm-sender-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	existingThread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Sender Lead",
		Slug:     "sender-lead",
		Metadata: `{"contact_email":"dave@example.com","email_address":"dave@example.com"}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(existingThread).Error)

	matcher := NewThreadMatcher(db)
	parsed := &ParsedEmail{
		From:    "dave@example.com",
		Subject: "Follow-up",
		Body:    "Another message from Dave",
	}

	result, err := matcher.Match(context.Background(), org.ID, parsed, models.RoutingActionSalesLead)
	require.NoError(t, err)
	assert.False(t, result.IsNew)
	assert.Equal(t, "sender_email", result.MatchBy)
	assert.Equal(t, existingThread.ID, result.Thread.ID)
}

func TestThreadMatcher_CreateNewLead(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "tm-newlead-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	matcher := NewThreadMatcher(db)
	parsed := &ParsedEmail{
		MessageID: "new-msg@example.com",
		From:      "newcontact@example.com",
		Subject:   "New Inquiry",
		Body:      "I want to learn more about your product.",
	}

	result, err := matcher.Match(context.Background(), org.ID, parsed, models.RoutingActionSalesLead)
	require.NoError(t, err)
	assert.True(t, result.IsNew)
	assert.Equal(t, "new_lead", result.MatchBy)
	assert.NotNil(t, result.Thread)
	assert.Contains(t, result.Thread.Title, "New Inquiry")
	assert.Contains(t, result.Thread.Metadata, "newcontact@example.com")
	assert.Contains(t, result.Thread.Metadata, "new-msg@example.com")
}

func TestThreadMatcher_NilParsedEmail(t *testing.T) {
	db := setupTestDB(t)
	matcher := NewThreadMatcher(db)
	_, err := matcher.Match(context.Background(), "org-1", nil, models.RoutingActionSalesLead)
	assert.Error(t, err)
}

func TestThreadMatcher_AppendMessageID(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "tm-append-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Thread for Append",
		Slug:     "thread-append",
		Metadata: `{"message_ids":["first@example.com"]}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(thread).Error)

	matcher := NewThreadMatcher(db)
	err := matcher.AppendMessageID(context.Background(), thread, "second@example.com")
	require.NoError(t, err)

	ids := ExtractMessageIDs(thread.Metadata)
	assert.Contains(t, ids, "first@example.com")
	assert.Contains(t, ids, "second@example.com")

	// Duplicate should be a no-op.
	err = matcher.AppendMessageID(context.Background(), thread, "second@example.com")
	require.NoError(t, err)
	ids = ExtractMessageIDs(thread.Metadata)
	assert.Len(t, ids, 2)

	// Empty message ID is a no-op.
	err = matcher.AppendMessageID(context.Background(), thread, "")
	require.NoError(t, err)
}

// --- Attachment handler tests ---

func TestAttachmentHandler_ProcessAttachments(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "att-proc-org")
	storage := NewMockStorageProvider()
	handler := NewAttachmentHandler(db, storage)

	attachments := []ParsedAttachment{
		{Filename: "doc.pdf", ContentType: "application/pdf", Data: []byte("pdf content")},
		{Filename: "img.png", ContentType: "image/png", Data: make([]byte, 20*1024)}, // 20KB, should be stored
	}

	// We need a message to link to.
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	thread := &models.Thread{BoardID: board.ID, Title: "Test", Slug: "test-att", Metadata: "{}", AuthorID: "system"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "test", AuthorID: "system", Type: models.MessageTypeEmail, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	uploads, err := handler.ProcessAttachments(context.Background(), org.ID, msg.ID, attachments)
	require.NoError(t, err)
	assert.Len(t, uploads, 2)
	assert.Equal(t, "doc.pdf", uploads[0].Filename)
	assert.Equal(t, "image/png", uploads[1].ContentType)
}

func TestAttachmentHandler_SkipsSmallInlineImages(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "att-skip-org")
	storage := NewMockStorageProvider()
	handler := NewAttachmentHandler(db, storage)

	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	thread := &models.Thread{BoardID: board.ID, Title: "Test", Slug: "test-skip", Metadata: "{}", AuthorID: "system"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "test", AuthorID: "system", Type: models.MessageTypeEmail, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	attachments := []ParsedAttachment{
		// Inline image under 10KB — should be skipped.
		{Filename: "tiny.gif", ContentType: "image/gif", Data: make([]byte, 5*1024), IsInline: true, ContentID: "cid1"},
		// Regular attachment — should be stored.
		{Filename: "report.pdf", ContentType: "application/pdf", Data: []byte("pdf data")},
	}

	uploads, err := handler.ProcessAttachments(context.Background(), org.ID, msg.ID, attachments)
	require.NoError(t, err)
	assert.Len(t, uploads, 1)
	assert.Equal(t, "report.pdf", uploads[0].Filename)
}

func TestAttachmentHandler_EmptyAttachments(t *testing.T) {
	db := setupTestDB(t)
	storage := NewMockStorageProvider()
	handler := NewAttachmentHandler(db, storage)

	uploads, err := handler.ProcessAttachments(context.Background(), "org", "msg", nil)
	require.NoError(t, err)
	assert.Nil(t, uploads)
}

func TestAttachmentHandler_LargeInlineImageStored(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "att-large-org")
	storage := NewMockStorageProvider()
	handler := NewAttachmentHandler(db, storage)

	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)
	thread := &models.Thread{BoardID: board.ID, Title: "Test", Slug: "test-large", Metadata: "{}", AuthorID: "system"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "test", AuthorID: "system", Type: models.MessageTypeEmail, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	attachments := []ParsedAttachment{
		// Inline image OVER 10KB — should be stored.
		{Filename: "big.png", ContentType: "image/png", Data: make([]byte, 15*1024), IsInline: true, ContentID: "cid2"},
	}

	uploads, err := handler.ProcessAttachments(context.Background(), org.ID, msg.ID, attachments)
	require.NoError(t, err)
	assert.Len(t, uploads, 1)
}

// --- Metadata tests ---

func TestBuildMessageMetadata(t *testing.T) {
	parsed := &ParsedEmail{
		MessageID:  "msg@example.com",
		InReplyTo:  "parent@example.com",
		References: []string{"ref1@example.com"},
		From:       "sender@example.com",
		To:         []string{"recipient@example.com"},
		CC:         []string{"cc@example.com"},
		Subject:    "Test Subject",
	}
	meta := BuildMessageMetadata(parsed, "event-123")
	assert.Contains(t, meta, "msg@example.com")
	assert.Contains(t, meta, "parent@example.com")
	assert.Contains(t, meta, "sender@example.com")
	assert.Contains(t, meta, "event-123")
	assert.Contains(t, meta, "email")
}

func TestBuildThreadMetadata(t *testing.T) {
	parsed := &ParsedEmail{
		MessageID: "msg@example.com",
		From:      "sender@example.com",
	}
	meta := BuildThreadMetadata(parsed)
	assert.Contains(t, meta, "inbound_email")
	assert.Contains(t, meta, "sender@example.com")
	assert.Contains(t, meta, "msg@example.com")
}

func TestUpdateThreadMetadataWithMessageID(t *testing.T) {
	// Empty metadata.
	updated, err := UpdateThreadMetadataWithMessageID("{}", "new@example.com")
	require.NoError(t, err)
	assert.Contains(t, updated, "new@example.com")

	// Existing IDs.
	updated, err = UpdateThreadMetadataWithMessageID(`{"message_ids":["existing@example.com"]}`, "new@example.com")
	require.NoError(t, err)
	assert.Contains(t, updated, "existing@example.com")
	assert.Contains(t, updated, "new@example.com")

	// Duplicate — no change.
	updated, err = UpdateThreadMetadataWithMessageID(`{"message_ids":["dup@example.com"]}`, "dup@example.com")
	require.NoError(t, err)
	ids := ExtractMessageIDs(updated)
	assert.Len(t, ids, 1)

	// Empty message ID — no change.
	updated, err = UpdateThreadMetadataWithMessageID(`{"key":"val"}`, "")
	require.NoError(t, err)
	assert.Contains(t, updated, "key")
}

func TestExtractMessageIDs(t *testing.T) {
	assert.Nil(t, ExtractMessageIDs(""))
	assert.Nil(t, ExtractMessageIDs("{}"))
	assert.Nil(t, ExtractMessageIDs("invalid json"))
	ids := ExtractMessageIDs(`{"message_ids":["a@b","c@d"]}`)
	assert.Equal(t, []string{"a@b", "c@d"}, ids)
}

func TestExtractEmailAddressFromMeta(t *testing.T) {
	assert.Equal(t, "", ExtractEmailAddressFromMeta(""))
	assert.Equal(t, "", ExtractEmailAddressFromMeta("{}"))
	assert.Equal(t, "a@b.com", ExtractEmailAddressFromMeta(`{"email_address":"a@b.com"}`))
	assert.Equal(t, "c@d.com", ExtractEmailAddressFromMeta(`{"contact_email":"c@d.com"}`))
}

func TestSanitizeEmailHeader(t *testing.T) {
	assert.Equal(t, "hello world", SanitizeEmailHeader("hello  \x00  world"))
	assert.Equal(t, "clean", SanitizeEmailHeader("clean"))
	assert.Equal(t, "", SanitizeEmailHeader(""))
}

// --- OAuth tests ---

func TestOAuthCredentials_Validate(t *testing.T) {
	tests := []struct {
		name    string
		creds   OAuthCredentials
		wantErr bool
	}{
		{"valid", OAuthCredentials{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"}, false},
		{"missing client_id", OAuthCredentials{ClientSecret: "secret", RefreshToken: "token"}, true},
		{"missing client_secret", OAuthCredentials{ClientID: "id", RefreshToken: "token"}, true},
		{"missing refresh_token", OAuthCredentials{ClientID: "id", ClientSecret: "secret"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGoogleOAuthService_TokenCaching(t *testing.T) {
	callCount := 0
	refreshFn := func(_ OAuthCredentials) (string, time.Duration, error) {
		callCount++
		return fmt.Sprintf("token-%d", callCount), 3600 * time.Second, nil
	}

	creds := OAuthCredentials{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"}
	svc := newGoogleOAuthServiceForTest(creds, refreshFn, time.Now)

	// First call should refresh.
	token1, err := svc.GetXOAUTH2Token("user@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, token1)
	assert.Equal(t, 1, callCount)

	// Second call should use cache.
	token2, err := svc.GetXOAUTH2Token("user@example.com")
	require.NoError(t, err)
	assert.Equal(t, token1, token2)
	assert.Equal(t, 1, callCount) // No additional refresh.
}

func TestGoogleOAuthService_TokenRefreshOnExpiry(t *testing.T) {
	now := time.Now()
	callCount := 0
	refreshFn := func(_ OAuthCredentials) (string, time.Duration, error) {
		callCount++
		return fmt.Sprintf("token-%d", callCount), 30 * time.Second, nil // Expires in 30s
	}

	creds := OAuthCredentials{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"}
	svc := newGoogleOAuthServiceForTest(creds, refreshFn, func() time.Time { return now })

	// First call — refreshes.
	_, err := svc.GetXOAUTH2Token("user@example.com")
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Token expires in 30s, but we check with 60s buffer, so it should already need refresh.
	token2, err := svc.GetXOAUTH2Token("user@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, token2)
	assert.Equal(t, 2, callCount) // Refreshed because 30s TTL < 60s buffer.
}

func TestGoogleOAuthService_RefreshError(t *testing.T) {
	refreshFn := func(_ OAuthCredentials) (string, time.Duration, error) {
		return "", 0, fmt.Errorf("network error")
	}

	creds := OAuthCredentials{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"}
	svc := newGoogleOAuthServiceForTest(creds, refreshFn, time.Now)

	_, err := svc.GetXOAUTH2Token("user@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestBuildXOAUTH2String(t *testing.T) {
	token := buildXOAUTH2String("user@gmail.com", "access-token-123")
	decoded, err := base64.StdEncoding.DecodeString(token)
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "user=user@gmail.com")
	assert.Contains(t, string(decoded), "auth=Bearer access-token-123")
}

func TestParseOAuthCredentials(t *testing.T) {
	_, err := ParseOAuthCredentials("")
	assert.Error(t, err)

	_, err = ParseOAuthCredentials("{}")
	assert.Error(t, err)

	creds, err := ParseOAuthCredentials(`{"client_id":"id","client_secret":"secret","oauth_refresh_token":"token"}`)
	require.NoError(t, err)
	assert.Equal(t, "id", creds.ClientID)
	assert.Equal(t, "secret", creds.ClientSecret)
	assert.Equal(t, "token", creds.RefreshToken)
}

func TestMockOAuthService(t *testing.T) {
	mock := &MockOAuthService{Token: "mock-token"}
	token, err := mock.GetXOAUTH2Token("user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "mock-token", token)
	assert.Equal(t, 1, mock.GetCallCount)
}

// --- IMAP Provider tests ---

func TestMockIMAPProvider_ConnectAndFetch(t *testing.T) {
	mock := NewMockIMAPProvider()
	msg := makeMessage(map[string]string{"From": "test@example.com"}, "Body")
	mock.AddMessage(1, msg)

	err := mock.Connect(channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user"})
	require.NoError(t, err)
	assert.True(t, mock.Connected)
	assert.Equal(t, 1, mock.ConnectCalls)

	fetched, err := mock.FetchMessage(context.Background(), 1)
	require.NoError(t, err)
	assert.NotNil(t, fetched)

	_, err = mock.FetchMessage(context.Background(), 999)
	assert.Error(t, err)

	err = mock.Close()
	require.NoError(t, err)
	assert.False(t, mock.Connected)
}

func TestMockIMAPProvider_ConnectFunc(t *testing.T) {
	mock := NewMockIMAPProvider()
	mock.ConnectFunc = func(_ channel.EmailConfig) error {
		return fmt.Errorf("connection refused")
	}

	err := mock.Connect(channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user"})
	assert.Error(t, err)
	assert.False(t, mock.Connected)
}

// --- Connection Pool tests ---

func TestConnectionPool_GetAndRemove(t *testing.T) {
	factory := func() IMAPProvider { return NewMockIMAPProvider() }
	pool := NewConnectionPool(factory)

	p1 := pool.Get("org-1")
	assert.NotNil(t, p1)
	assert.Equal(t, 1, pool.Size())

	// Same org returns same provider.
	p2 := pool.Get("org-1")
	assert.Same(t, p1, p2)

	// Different org creates new.
	p3 := pool.Get("org-2")
	assert.NotSame(t, p1, p3)
	assert.Equal(t, 2, pool.Size())

	// Remove.
	err := pool.Remove("org-1")
	require.NoError(t, err)
	assert.Equal(t, 1, pool.Size())

	// Remove non-existent is no-op.
	err = pool.Remove("org-999")
	require.NoError(t, err)
}

func TestConnectionPool_CloseAll(t *testing.T) {
	factory := func() IMAPProvider { return NewMockIMAPProvider() }
	pool := NewConnectionPool(factory)

	pool.Get("org-1")
	pool.Get("org-2")
	assert.Equal(t, 2, pool.Size())

	pool.CloseAll()
	assert.Equal(t, 0, pool.Size())
}

// --- IDLE Manager tests ---

func TestIDLEManager_StartAndStop(t *testing.T) {
	mock := NewMockIMAPProvider()
	// StartIDLE blocks until context is cancelled (simulated by returning immediately).
	mock.StartIDLEFunc = func(_ string, _ func(uint32)) error {
		return nil // Simulate clean close — will cause reconnect loop.
	}

	cfg := IDLEManagerConfig{
		OrgID:       "org-idle-1",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user", Mailbox: "INBOX"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}

	mgr := NewIDLEManager(cfg)
	mgr.Start()

	// Give it a moment to connect.
	time.Sleep(50 * time.Millisecond)

	mgr.Stop()
	assert.Equal(t, ConnectionStateDisconnected, mgr.State())
}

func TestIDLEManager_ReconnectOnError(t *testing.T) {
	connectCount := 0
	mock := NewMockIMAPProvider()
	mock.ConnectFunc = func(_ channel.EmailConfig) error {
		connectCount++
		if connectCount <= 2 {
			return fmt.Errorf("connection failed attempt %d", connectCount)
		}
		return nil
	}
	mock.StartIDLEFunc = func(_ string, _ func(uint32)) error {
		// Return nil after success to trigger loop restart.
		return nil
	}

	cfg := IDLEManagerConfig{
		OrgID:       "org-idle-reconnect",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}

	// Use a no-op sleep to skip backoff delays in tests.
	mgr := newIDLEManagerForTest(cfg, func(_ time.Duration) {})
	mgr.Start()
	time.Sleep(200 * time.Millisecond)
	mgr.Stop()

	assert.GreaterOrEqual(t, connectCount, 2, "should have attempted multiple connects")
}

func TestIDLEManager_HealthReport(t *testing.T) {
	mock := NewMockIMAPProvider()
	cfg := IDLEManagerConfig{
		OrgID:       "org-health-report",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}

	mgr := NewIDLEManager(cfg)
	report := mgr.HealthReport()
	assert.Equal(t, "org-health-report", report["org_id"])
	assert.Equal(t, "disconnected", report["state"])
}

func TestComputeIDLEBackoff(t *testing.T) {
	// Attempt 0 should return base delay.
	d := computeIDLEBackoff(0)
	assert.Equal(t, IDLEBaseDelay, d)

	// Attempt 1 should be around base delay.
	d = computeIDLEBackoff(1)
	assert.GreaterOrEqual(t, d, time.Duration(float64(IDLEBaseDelay)*0.75))
	assert.LessOrEqual(t, d, time.Duration(float64(IDLEBaseDelay)*1.25))

	// High attempts should be capped at max.
	d = computeIDLEBackoff(100)
	assert.LessOrEqual(t, d, IDLEMaxDelay+time.Duration(float64(IDLEMaxDelay)*IDLEJitterFraction))
}

// --- IDLE Manager Registry tests ---

func TestIDLEManagerRegistry(t *testing.T) {
	registry := NewIDLEManagerRegistry()
	assert.Equal(t, 0, registry.Size())

	mock := NewMockIMAPProvider()
	mock.StartIDLEFunc = func(_ string, _ func(uint32)) error { return nil }
	cfg := IDLEManagerConfig{
		OrgID:       "org-reg-1",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}

	mgr := NewIDLEManager(cfg)
	registry.Register("org-reg-1", mgr)
	assert.Equal(t, 1, registry.Size())

	got := registry.Get("org-reg-1")
	assert.NotNil(t, got)

	// Deregister.
	registry.Deregister("org-reg-1")
	assert.Equal(t, 0, registry.Size())
	assert.Nil(t, registry.Get("org-reg-1"))
}

func TestIDLEManagerRegistry_StopAll(t *testing.T) {
	registry := NewIDLEManagerRegistry()

	for i := 0; i < 3; i++ {
		mock := NewMockIMAPProvider()
		mock.StartIDLEFunc = func(_ string, _ func(uint32)) error { return nil }
		cfg := IDLEManagerConfig{
			OrgID:       fmt.Sprintf("org-stopall-%d", i),
			EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "user"},
			Provider:    mock,
			OnMessage:   func(_ uint32) {},
		}
		registry.Register(cfg.OrgID, NewIDLEManager(cfg))
	}
	assert.Equal(t, 3, registry.Size())

	registry.StopAll()
	assert.Equal(t, 0, registry.Size())
}

func TestFormatError(t *testing.T) {
	assert.Equal(t, "", FormatError("org-1", nil))
	assert.Contains(t, FormatError("org-1", fmt.Errorf("test error")), "org-1")
	assert.Contains(t, FormatError("org-1", fmt.Errorf("test error")), "test error")
}

// --- Email Service tests ---

func TestService_ProcessInbound_NewLead(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-newlead-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	storage := NewMockStorageProvider()
	svc := NewService(db, storage, nil)

	msg := makeMessage(map[string]string{
		"Message-ID": "<new-lead@example.com>",
		"From":       "prospect@example.com",
		"Subject":    "Product Inquiry",
	}, "I'm interested in your product.")

	result, err := svc.ProcessInbound(context.Background(), org.ID, models.RoutingActionSalesLead, msg)
	require.NoError(t, err)
	assert.True(t, result.IsNewLead)
	assert.Equal(t, "new_lead", result.MatchBy)
	assert.NotNil(t, result.Thread)
	assert.NotNil(t, result.Message)
	assert.Equal(t, models.MessageTypeEmail, result.Message.Type)
	assert.Contains(t, result.Message.Metadata, "new-lead@example.com")
}

func TestService_ProcessInbound_ExistingThread(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-existing-org")
	_, board := createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	// Create existing thread.
	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    "Existing Lead",
		Slug:     "existing-lead",
		Metadata: `{"message_ids":["original@example.com"],"contact_email":"alice@example.com","email_address":"alice@example.com"}`,
		AuthorID: "system",
	}
	require.NoError(t, db.Create(thread).Error)

	storage := NewMockStorageProvider()
	svc := NewService(db, storage, nil)

	msg := makeMessage(map[string]string{
		"Message-ID":  "<reply@example.com>",
		"From":        "alice@example.com",
		"In-Reply-To": "<original@example.com>",
		"Subject":     "Re: Existing Lead",
	}, "Follow-up message.")

	result, err := svc.ProcessInbound(context.Background(), org.ID, models.RoutingActionSalesLead, msg)
	require.NoError(t, err)
	assert.False(t, result.IsNewLead)
	assert.Equal(t, thread.ID, result.Thread.ID)
}

func TestService_ProcessInbound_WithAttachments(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-att-org")
	_, _ = createTestSpaceAndBoard(t, db, org.ID, models.SpaceTypeCRM)

	storage := NewMockStorageProvider()
	svc := NewService(db, storage, nil)

	boundary := "svcboundary"
	parts := []string{
		"Content-Type: text/plain\r\n\r\nBody text",
		"Content-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"contract.pdf\"\r\n\r\nPDF bytes",
	}
	msg := makeMultipartMessage(map[string]string{
		"Message-ID": "<att-msg@example.com>",
		"From":       "client@example.com",
		"Subject":    "Contract attached",
	}, boundary, parts)

	result, err := svc.ProcessInbound(context.Background(), org.ID, models.RoutingActionSalesLead, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Message)
	assert.Len(t, result.Uploads, 1)
	assert.Equal(t, "contract.pdf", result.Uploads[0].Filename)
}

func TestService_ProcessInbound_NilMessage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, NewMockStorageProvider(), nil)
	_, err := svc.ProcessInbound(context.Background(), "org-1", models.RoutingActionSalesLead, nil)
	assert.Error(t, err)
}

func TestService_ProcessInbound_EmptyOrgID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, NewMockStorageProvider(), nil)
	msg := makeMessage(map[string]string{"From": "test@example.com"}, "body")
	_, err := svc.ProcessInbound(context.Background(), "", models.RoutingActionSalesLead, msg)
	assert.Error(t, err)
}

func TestService_Normalize(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, NewMockStorageProvider(), nil)

	rawEmail := "Message-ID: <norm@example.com>\r\nFrom: sender@example.com\r\nSubject: Test\r\n\r\nBody content"
	evt, err := svc.Normalize("org-1", []byte(rawEmail))
	require.NoError(t, err)
	assert.Equal(t, models.ChannelTypeEmail, evt.ChannelType)
	assert.Equal(t, "org-1", evt.OrgID)
	assert.Equal(t, "norm@example.com", evt.ExternalID)
	assert.Equal(t, "sender@example.com", evt.SenderIdentifier)
	assert.Equal(t, "Test", evt.Subject)
	assert.Contains(t, evt.Body, "Body content")
}

func TestService_Normalize_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, NewMockStorageProvider(), nil)

	_, err := svc.Normalize("org-1", nil)
	assert.Error(t, err)

	_, err = svc.Normalize("org-1", []byte{})
	assert.Error(t, err)
}

// --- isImageContentType tests ---

func TestIsImageContentType(t *testing.T) {
	assert.True(t, isImageContentType("image/png"))
	assert.True(t, isImageContentType("image/jpeg"))
	assert.True(t, isImageContentType("Image/GIF"))
	assert.False(t, isImageContentType("application/pdf"))
	assert.False(t, isImageContentType("text/plain"))
}

// --- LiveIMAPProvider unit tests (no real IMAP connection) ---

func TestLiveIMAPProvider_NotConnected_Errors(t *testing.T) {
	p := NewLiveIMAPProvider(nil) // nil logger falls back to slog.Default()

	// FetchMessage without connecting should return an error.
	_, err := p.FetchMessage(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	// StartIDLE without connecting should return an error.
	err = p.StartIDLE("INBOX", func(_ uint32) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	// Close without connecting should be a no-op.
	err = p.Close()
	assert.NoError(t, err)
}

func TestLiveIMAPProvider_DrainNotify(t *testing.T) {
	p := NewLiveIMAPProvider(nil)

	// Drain empty channel — should return immediately.
	p.drainNotify()

	// Pre-load notifications then drain.
	for i := 0; i < 5; i++ {
		select {
		case p.notifyCh <- struct{}{}:
		default:
		}
	}
	p.drainNotify()
	assert.Equal(t, 0, len(p.notifyCh))
}

// --- InboxWatcher unit tests ---

func TestInboxWatcher_StartWithNoInboxes(t *testing.T) {
	db := setupTestDB(t)
	storage := NewMockStorageProvider()
	watcher := NewInboxWatcher(db, storage, nil)

	// With no EmailInbox records, Start should succeed and register 0 managers.
	err := watcher.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, watcher.registry.Size())

	watcher.Stop()
	assert.Equal(t, 0, watcher.registry.Size())
}

func TestInboxWatcher_RestartInbox_Disabled(t *testing.T) {
	db := setupTestDB(t)
	storage := NewMockStorageProvider()
	watcher := NewInboxWatcher(db, storage, nil)

	// Register a mock manager first.
	mock := NewMockIMAPProvider()
	mock.StartIDLEFunc = func(_ string, _ func(uint32)) error { return nil }
	cfg := IDLEManagerConfig{
		OrgID:       "test-org",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "u"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}
	watcher.registry.Register("inbox-id-1", NewIDLEManager(cfg))
	assert.Equal(t, 1, watcher.registry.Size())

	// RestartInbox with Enabled=false should deregister but not re-add.
	watcher.RestartInbox(models.EmailInbox{BaseModel: models.BaseModel{ID: "inbox-id-1"}, Enabled: false})
	assert.Equal(t, 0, watcher.registry.Size())
}

// --- Routing action tests ---

func TestRoutingActionToSpaceType(t *testing.T) {
	assert.Equal(t, models.SpaceTypeSupport, routingActionToSpaceType(models.RoutingActionSupportTicket))
	assert.Equal(t, models.SpaceTypeCRM, routingActionToSpaceType(models.RoutingActionSalesLead))
	assert.Equal(t, models.SpaceTypeGeneral, routingActionToSpaceType(models.RoutingActionGeneral))
	// Unknown falls back to CRM.
	assert.Equal(t, models.SpaceTypeCRM, routingActionToSpaceType("unknown"))
}

func TestService_ProcessInbound_SupportTicketRouting(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "svc-support-org")
	// Create CRM and support spaces with distinct slugs to avoid UNIQUE constraint.
	crmSpace := &models.Space{OrgID: org.ID, Name: "CRM", Slug: "crm-space", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, db.Create(crmSpace).Error)
	crmBoard := &models.Board{SpaceID: crmSpace.ID, Name: "CRM Board", Slug: "crm-board", Metadata: "{}"}
	require.NoError(t, db.Create(crmBoard).Error)

	supportSpace := &models.Space{OrgID: org.ID, Name: "Support", Slug: "support-space", Type: models.SpaceTypeSupport, Metadata: "{}"}
	require.NoError(t, db.Create(supportSpace).Error)
	supportBoard := &models.Board{SpaceID: supportSpace.ID, Name: "Support Board", Slug: "support-board", Metadata: "{}"}
	require.NoError(t, db.Create(supportBoard).Error)

	svc := NewService(db, NewMockStorageProvider(), nil)
	msg := makeMessage(map[string]string{
		"From":    "customer@example.com",
		"Subject": "Help needed",
	}, "I need support please.")

	result, err := svc.ProcessInbound(context.Background(), org.ID, models.RoutingActionSupportTicket, msg)
	require.NoError(t, err)
	assert.True(t, result.IsNewLead)
	assert.NotNil(t, result.Thread)

	// The thread should be in the support space.
	var space models.Space
	var board models.Board
	require.NoError(t, db.First(&board, "id = ?", result.Thread.BoardID).Error)
	require.NoError(t, db.First(&space, "id = ?", board.SpaceID).Error)
	assert.Equal(t, models.SpaceTypeSupport, space.Type)
}

// --- IDLEManager accessor tests ---

func TestIDLEManager_Accessors(t *testing.T) {
	mock := NewMockIMAPProvider()
	cfg := IDLEManagerConfig{
		OrgID:       "org-acc",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "u"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}
	mgr := NewIDLEManager(cfg)

	// Initial state: no error, zero reconnect attempts, not healthy.
	assert.Nil(t, mgr.LastError())
	assert.Equal(t, 0, mgr.ReconnectAttempts())
	assert.False(t, mgr.IsHealthy()) // disconnected state
}

func TestIDLEManagerRegistry_HealthReportsAndIsHealthy(t *testing.T) {
	registry := NewIDLEManagerRegistry()

	mock := NewMockIMAPProvider()
	mock.StartIDLEFunc = func(_ string, _ func(uint32)) error { return nil }
	cfg := IDLEManagerConfig{
		OrgID:       "org-hr",
		EmailConfig: channel.EmailConfig{IMAPHost: "mail.test", IMAPPort: 993, Username: "u"},
		Provider:    mock,
		OnMessage:   func(_ uint32) {},
	}
	mgr := NewIDLEManager(cfg)
	registry.Register("org-hr", mgr)

	// HealthReports returns one report per manager.
	reports := registry.HealthReports()
	assert.Len(t, reports, 1)
	assert.Equal(t, "org-hr", reports[0]["org_id"])

	// IsHealthy returns false for disconnected org and for unknown org.
	assert.False(t, registry.IsHealthy("org-hr"))
	assert.False(t, registry.IsHealthy("unknown-org"))

	mgr.Stop()
}

// --- NewGoogleOAuthService tests ---

func TestNewGoogleOAuthService_Valid(t *testing.T) {
	creds := OAuthCredentials{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"}
	svc, err := NewGoogleOAuthService(creds)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewGoogleOAuthService_Invalid(t *testing.T) {
	_, err := NewGoogleOAuthService(OAuthCredentials{})
	assert.Error(t, err)
}

// --- MockStorageProvider Get/Delete tests ---

func TestMockStorageProvider_GetAndDelete(t *testing.T) {
	storage := NewMockStorageProvider()

	// Store a file.
	path, err := storage.Store("test.pdf", strings.NewReader("content"))
	require.NoError(t, err)

	// Get it back.
	r, err := storage.Get(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })

	// Get non-existent file → error.
	_, err = storage.Get("missing.pdf")
	assert.Error(t, err)

	// Delete the file.
	err = storage.Delete(path)
	require.NoError(t, err)

	// After delete, Get should fail.
	_, err = storage.Get(path)
	assert.Error(t, err)

	// Delete error propagation.
	storage.DeleteErr = fmt.Errorf("disk error")
	err = storage.Delete("any-path")
	assert.Error(t, err)
}

// --- defaultRefreshFunc test ---

func TestDefaultRefreshFunc_ReturnsError(t *testing.T) {
	_, _, err := defaultRefreshFunc(OAuthCredentials{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

// --- mockClock tests ---

func TestMockClock(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := newMockClock(start)
	assert.Equal(t, start, clock.Now())

	clock.Advance(10 * time.Second)
	assert.Equal(t, start.Add(10*time.Second), clock.Now())
}
