package globalspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/vote"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Stub vote toggler ---

type stubVoteToggler struct {
	result *vote.VoteResult
	err    error
}

func (s *stubVoteToggler) Toggle(_ context.Context, threadID, _ string, _ models.Role, _ string) (*vote.VoteResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result != nil {
		return s.result, nil
	}
	return &vote.VoteResult{Voted: true, VoteScore: 1, Weight: 1}, nil
}

// globalSpaceRouterFull extends the existing router with message and vote routes.
func globalSpaceRouterFull(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/global-spaces/{space}/threads", h.ListThreads)
	r.Post("/global-spaces/{space}/threads", h.CreateThread)
	r.Get("/global-spaces/{space}/threads/{slug}", h.GetThread)
	r.Patch("/global-spaces/{space}/threads/{slug}", h.UpdateThread)
	r.Get("/global-spaces/{space}/threads/{slug}/attachments", h.ListAttachments)
	r.Post("/global-spaces/{space}/threads/{slug}/attachments", h.UploadAttachment)
	r.Get("/global-spaces/{space}/threads/{slug}/messages", h.ListMessages)
	r.Post("/global-spaces/{space}/threads/{slug}/messages", h.CreateMessage)
	r.Post("/global-spaces/{space}/threads/{slug}/vote", h.ToggleVote)
	return r
}

// --- SetVoteService ---

func TestService_SetVoteService(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)

	stub := &stubVoteToggler{}
	svc.SetVoteService(stub)
	assert.Equal(t, stub, svc.voteSvc)

	// Can be set to nil without panic.
	svc.SetVoteService(nil)
	assert.Nil(t, svc.voteSvc)
}

// --- Service: ListMessages ---

func TestService_ListMessages(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "msg-author", CreateInput{Title: "Message Thread"})
	require.NoError(t, err)

	// Seed user shadow for author enrichment.
	require.NoError(t, db.Create(&models.UserShadow{
		ClerkUserID: "msg-author",
		Email:       "msg@example.com",
		DisplayName: "Message Author",
		LastSeenAt:  time.Now(),
		SyncedAt:    time.Now(),
	}).Error)

	// Seed an org for org name enrichment.
	org := &models.Org{Name: "Message Org", Slug: "msg-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: org.ID, UserID: "msg-author", Role: models.RoleContributor,
	}).Error)

	t.Run("empty list when no messages", func(t *testing.T) {
		msgs, pi, err := svc.ListMessages(ctx, "global-support", th.Slug, pagination.Params{Limit: 50})
		require.NoError(t, err)
		require.NotNil(t, msgs)
		assert.Empty(t, msgs)
		assert.False(t, pi.HasMore)
	})

	// Create a message directly.
	msg, err := svc.CreateMessage(ctx, "global-support", th.Slug, "msg-author", CreateMessageInput{Body: "Hello forum"})
	require.NoError(t, err)
	require.NotNil(t, msg)

	t.Run("returns messages with author enrichment", func(t *testing.T) {
		msgs, pi, err := svc.ListMessages(ctx, "global-support", th.Slug, pagination.Params{Limit: 50})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "Hello forum", msgs[0].Body)
		assert.Equal(t, "Message Author", msgs[0].AuthorName)
		assert.Equal(t, "Message Org", msgs[0].AuthorOrg)
		assert.False(t, pi.HasMore)
	})

	t.Run("returns nil for unknown space", func(t *testing.T) {
		msgs, pi, err := svc.ListMessages(ctx, "no-such-space", th.Slug, pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Nil(t, msgs)
		assert.Nil(t, pi)
	})

	t.Run("returns nil for unknown thread", func(t *testing.T) {
		msgs, pi, err := svc.ListMessages(ctx, "global-support", "no-such-slug", pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Nil(t, msgs)
		assert.Nil(t, pi)
	})
}

// --- Service: CreateMessage ---

func TestService_CreateMessage(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "creator-u", CreateInput{Title: "Reply Thread"})
	require.NoError(t, err)

	t.Run("success creates message", func(t *testing.T) {
		msg, err := svc.CreateMessage(ctx, "global-support", th.Slug, "creator-u", CreateMessageInput{Body: "A reply"})
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, "A reply", msg.Body)
		assert.Equal(t, "creator-u", msg.AuthorID)
		assert.Equal(t, models.MessageTypeComment, msg.Type)
	})

	t.Run("empty body returns error", func(t *testing.T) {
		_, err := svc.CreateMessage(ctx, "global-support", th.Slug, "creator-u", CreateMessageInput{Body: ""})
		require.Error(t, err)
		assert.EqualError(t, err, "body is required")
	})

	t.Run("locked thread returns error", func(t *testing.T) {
		lockedTh, err := svc.CreateThread(ctx, "global-support", "creator-u", CreateInput{Title: "Locked Thread"})
		require.NoError(t, err)
		require.NoError(t, db.Model(&models.Thread{}).Where("id = ?", lockedTh.ID).Update("is_locked", true).Error)

		_, err = svc.CreateMessage(ctx, "global-support", lockedTh.Slug, "creator-u", CreateMessageInput{Body: "blocked"})
		require.Error(t, err)
		assert.EqualError(t, err, "thread is locked")
	})

	t.Run("unknown space returns nil", func(t *testing.T) {
		msg, err := svc.CreateMessage(ctx, "no-such-space", th.Slug, "creator-u", CreateMessageInput{Body: "orphan"})
		require.NoError(t, err)
		assert.Nil(t, msg)
	})

	t.Run("unknown thread returns nil", func(t *testing.T) {
		msg, err := svc.CreateMessage(ctx, "global-support", "no-such-slug", "creator-u", CreateMessageInput{Body: "orphan"})
		require.NoError(t, err)
		assert.Nil(t, msg)
	})
}

// --- Service: ToggleVote ---

func TestService_ToggleVote(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "voter-u", CreateInput{Title: "Votable Thread"})
	require.NoError(t, err)

	t.Run("vote service not configured returns error", func(t *testing.T) {
		_, err := svc.ToggleVote(ctx, "global-support", th.Slug, "voter-u")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vote service not available")
	})

	// Inject stub vote toggler.
	stub := &stubVoteToggler{}
	svc.SetVoteService(stub)

	t.Run("success returns vote result", func(t *testing.T) {
		result, err := svc.ToggleVote(ctx, "global-support", th.Slug, "voter-u")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Voted)
		assert.Equal(t, 1, result.VoteScore)
	})

	t.Run("unknown space returns nil", func(t *testing.T) {
		result, err := svc.ToggleVote(ctx, "no-such-space", th.Slug, "voter-u")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("unknown thread returns nil", func(t *testing.T) {
		result, err := svc.ToggleVote(ctx, "global-support", "no-such-slug", "voter-u")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("vote service error propagates", func(t *testing.T) {
		errStub := &stubVoteToggler{err: fmt.Errorf("vote backend unavailable")}
		svc.SetVoteService(errStub)
		_, err := svc.ToggleVote(ctx, "global-support", th.Slug, "voter-u")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vote backend unavailable")
	})
}

// --- Handler: ListMessages ---

func TestHandler_ListMessages(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	h := NewHandler(svc)
	r := globalSpaceRouterFull(h)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "list-msg-u", CreateInput{Title: "Messages List Thread"})
	require.NoError(t, err)

	t.Run("empty list returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/"+th.Slug+"/messages", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Empty(t, data)
	})

	// Create a message so we can see it listed.
	_, err = svc.CreateMessage(ctx, "global-support", th.Slug, "list-msg-u", CreateMessageInput{Body: "Test reply"})
	require.NoError(t, err)

	t.Run("returns created messages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/"+th.Slug+"/messages", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]any)
		assert.Len(t, data, 1)
	})

	t.Run("not found thread returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/global-support/threads/no-such-slug/messages", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("unknown space returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/global-spaces/no-such-space/threads/"+th.Slug+"/messages", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- Handler: CreateMessage ---

func TestHandler_CreateMessage(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	h := NewHandler(svc)
	r := globalSpaceRouterFull(h)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "create-msg-u", CreateInput{Title: "Create Message Thread"})
	require.NoError(t, err)

	t.Run("success returns 201", func(t *testing.T) {
		body := `{"body":"A reply to the thread"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "create-msg-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "A reply to the thread", resp["body"])
		assert.Equal(t, "create-msg-u", resp["author_id"])
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := `{"body":"anon"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/messages", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "create-msg-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty body returns validation error", func(t *testing.T) {
		body := `{"body":""}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "create-msg-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("locked thread returns 403", func(t *testing.T) {
		lockedTh, err := svc.CreateThread(ctx, "global-support", "create-msg-u", CreateInput{Title: "Locked For Messages"})
		require.NoError(t, err)
		require.NoError(t, db.Model(&models.Thread{}).Where("id = ?", lockedTh.ID).Update("is_locked", true).Error)

		body := `{"body":"blocked reply"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+lockedTh.Slug+"/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "create-msg-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("not found thread returns 404", func(t *testing.T) {
		body := `{"body":"orphan reply"}`
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/no-such-slug/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUser(req, "create-msg-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- Handler: ToggleVote ---

func TestHandler_ToggleVote(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)

	// Inject stub so vote service is available.
	stub := &stubVoteToggler{}
	svc.SetVoteService(stub)

	h := NewHandler(svc)
	r := globalSpaceRouterFull(h)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "vote-u", CreateInput{Title: "Votable"})
	require.NoError(t, err)

	t.Run("success returns 200 with vote result", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/vote", nil)
		req = withUser(req, "vote-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, true, resp["voted"])
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/vote", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("not found thread returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/no-such-slug/vote", nil)
		req = withUser(req, "vote-u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("vote service error returns 500", func(t *testing.T) {
		errSvc := newSvc(db)
		errSvc.SetVoteService(&stubVoteToggler{err: fmt.Errorf("backend error")})
		errH := NewHandler(errSvc)
		errR := globalSpaceRouterFull(errH)

		req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/vote", nil)
		req = withUser(req, "vote-u")
		w := httptest.NewRecorder()
		errR.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestHandler_ToggleVote_NoVoteService covers the case where no vote service
// has been injected (service returns "vote service not available" error).
func TestHandler_ToggleVote_NoVoteService(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db) // no vote service injected
	h := NewHandler(svc)
	r := globalSpaceRouterFull(h)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "no-vote-svc-u", CreateInput{Title: "No Vote Svc"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/vote", nil)
	req = withUser(req, "no-vote-svc-u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- EnrichMessages (via ListMessages with enriched shadows) ---

func TestService_EnrichMessages_NoShadows(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "anon-author", CreateInput{Title: "Anon Thread"})
	require.NoError(t, err)

	_, err = svc.CreateMessage(ctx, "global-support", th.Slug, "anon-author", CreateMessageInput{Body: "anon reply"})
	require.NoError(t, err)

	msgs, _, err := svc.ListMessages(ctx, "global-support", th.Slug, pagination.Params{Limit: 50})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	// Author has no shadow — name should be empty.
	assert.Equal(t, "", msgs[0].AuthorName)
	assert.Equal(t, "", msgs[0].AuthorOrg)
}

// TestRepository_CreateThread covers the repository-level CreateThread
// function that creates a thread record directly.
func TestRepository_CreateThread(t *testing.T) {
	db, boardID := setupDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	thread := &models.Thread{
		BoardID:  boardID,
		Title:    "Direct Repo Thread",
		Body:     "body",
		Slug:     "direct-repo-thread",
		Metadata: "{}",
		AuthorID: "repo-author",
	}
	err := repo.CreateThread(ctx, thread)
	require.NoError(t, err)
	assert.NotEmpty(t, thread.ID)

	// Verify it's in the DB.
	found, err := repo.FindThreadBySlug(ctx, boardID, "direct-repo-thread")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "Direct Repo Thread", found.Title)
}

// TestRepository_ListMessages covers the repository ListMessages function.
func TestRepository_ListMessages(t *testing.T) {
	db, _ := setupDB(t)
	repo := NewRepository(db)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "rm-author", CreateInput{Title: "Repo Messages"})
	require.NoError(t, err)

	// Create two messages directly via repository.
	msg1 := &models.Message{ThreadID: th.ID, Body: "msg one", AuthorID: "rm-author", Metadata: "{}", Type: models.MessageTypeComment}
	msg2 := &models.Message{ThreadID: th.ID, Body: "msg two", AuthorID: "rm-author", Metadata: "{}", Type: models.MessageTypeComment}
	require.NoError(t, repo.CreateMessage(ctx, msg1))
	require.NoError(t, repo.CreateMessage(ctx, msg2))

	t.Run("lists all messages for thread", func(t *testing.T) {
		msgs, pi, err := repo.ListMessages(ctx, th.ID, pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
		assert.False(t, pi.HasMore)
	})

	t.Run("pagination works", func(t *testing.T) {
		msgs, pi, err := repo.ListMessages(ctx, th.ID, pagination.Params{Limit: 1})
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
		assert.True(t, pi.HasMore)
	})

	t.Run("empty for unknown thread", func(t *testing.T) {
		msgs, pi, err := repo.ListMessages(ctx, "non-existent-id", pagination.Params{Limit: 50})
		require.NoError(t, err)
		assert.Empty(t, msgs)
		assert.False(t, pi.HasMore)
	})
}

// TestRepository_ListMessages_InvalidCursor covers the invalid cursor error path.
func TestRepository_ListMessages_InvalidCursor(t *testing.T) {
	db, _ := setupDB(t)
	repo := NewRepository(db)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "cursor-author", CreateInput{Title: "Cursor Thread"})
	require.NoError(t, err)

	_, _, err = repo.ListMessages(ctx, th.ID, pagination.Params{Limit: 25, Cursor: "not-a-valid-cursor"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor")
}

// TestCanSeeThread_DefaultScope verifies the default switch branch returns false
// for an unrecognised VisibilityScope value.
func TestCanSeeThread_DefaultScope(t *testing.T) {
	thread := &models.Thread{}
	cv := &CallerVisibility{Scope: VisibilityScope(99)} // unknown scope
	assert.False(t, canSeeThread(cv, thread))
}

// TestService_CreateMessage_EmptyBodyVariant tests the empty body path via handler
// to cover the handler-level validation branch.
func TestService_CreateMessage_NilBodyInHandler(t *testing.T) {
	db, _ := setupDB(t)
	svc := newSvc(db)
	h := NewHandler(svc)
	r := globalSpaceRouterFull(h)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "nil-body-u", CreateInput{Title: "Nil Body Thread"})
	require.NoError(t, err)

	// Missing body field entirely — JSON decodes to empty Body.
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/global-spaces/global-support/threads/"+th.Slug+"/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, "nil-body-u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRepository_CreateMessage covers the repository CreateMessage function.
func TestRepository_CreateMessage(t *testing.T) {
	db, _ := setupDB(t)
	repo := NewRepository(db)
	svc := newSvc(db)
	ctx := context.Background()

	th, err := svc.CreateThread(ctx, "global-support", "rcm-author", CreateInput{Title: "Repo Create Msg"})
	require.NoError(t, err)

	msg := &models.Message{
		ThreadID: th.ID,
		Body:     "repo-created message",
		AuthorID: "rcm-author",
		Metadata: "{}",
		Type:     models.MessageTypeComment,
	}
	err = repo.CreateMessage(ctx, msg)
	require.NoError(t, err)
	assert.NotEmpty(t, msg.ID)

	// Verify via list.
	msgs, _, err := repo.ListMessages(ctx, th.ID, pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "repo-created message", msgs[0].Body)
}
