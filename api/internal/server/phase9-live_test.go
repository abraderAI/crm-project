package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
)

// phase9AuthReq is a helper similar to authReq but allows specifying the user.
func phase9AuthReq(t *testing.T, env *liveAuthEnv, method, url, body, userID string) *http.Response {
	t.Helper()
	token := env.SignToken(auth.JWTClaims{
		Subject:   userID,
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// setupPhase9Hierarchy creates a full hierarchy and returns IDs.
func setupPhase9Hierarchy(t *testing.T, env *liveAuthEnv) (orgID, spaceID, boardID, board2ID, threadID, thread2ID string) {
	t.Helper()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Phase9 Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID = decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Community","type":"community"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID = decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Feature Requests"}`)
	defer func() { _ = resp.Body.Close() }()
	boardID = decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"General Discussion"}`)
	defer func() { _ = resp.Body.Close() }()
	board2ID = decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Dark Mode","body":"Please add dark mode"}`)
	defer func() { _ = resp.Body.Close() }()
	threadID = decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Light Mode","body":"Please add light mode"}`)
	defer func() { _ = resp.Body.Close() }()
	thread2ID = decodeJSON(t, resp)["id"].(string)

	return
}

func threadURL(baseURL, orgID, spaceID, boardID, threadID string) string {
	return baseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/" + threadID
}

// --- Phase 9 Live API Tests ---

// TestLive_Phase9_VoteToggle tests voting on a thread and verifying VoteScore in GET response.
func TestLive_Phase9_VoteToggle(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	// Vote on.
	resp := authReq(t, env, "POST", base+"/vote", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	voteData := decodeJSON(t, resp)
	assert.Equal(t, true, voteData["voted"])
	score := int(voteData["vote_score"].(float64))
	assert.GreaterOrEqual(t, score, 1)

	// Verify VoteScore in GET response.
	resp = authReq(t, env, "GET", base, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	threadData := decodeJSON(t, resp)
	assert.Equal(t, float64(score), threadData["vote_score"])

	// Vote off (toggle).
	resp = authReq(t, env, "POST", base+"/vote", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	voteOffData := decodeJSON(t, resp)
	assert.Equal(t, false, voteOffData["voted"])
	assert.Equal(t, float64(0), voteOffData["vote_score"])
}

// TestLive_Phase9_VoteMultipleUsers tests votes from multiple users.
func TestLive_Phase9_VoteMultipleUsers(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	// User 1 votes.
	resp := phase9AuthReq(t, env, "POST", base+"/vote", "", "user1")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// User 2 votes.
	resp = phase9AuthReq(t, env, "POST", base+"/vote", "", "user2")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	data := decodeJSON(t, resp)
	assert.GreaterOrEqual(t, int(data["vote_score"].(float64)), 2)
}

// TestLive_Phase9_FlagWorkflow tests the full flag lifecycle: create → list → resolve.
func TestLive_Phase9_FlagWorkflow(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, _ := setupPhase9Hierarchy(t, env)
	_ = spaceID
	_ = boardID

	// Create flag.
	flagBody := `{"thread_id":"` + threadID + `","reason":"spam content"}`
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/flags", flagBody)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	flagData := decodeJSON(t, resp)
	flagID := flagData["id"].(string)
	assert.Equal(t, "open", flagData["status"])
	assert.Equal(t, "spam content", flagData["reason"])

	// List flags — should return the one we created.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/flags", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	listData := decodeJSON(t, resp)
	flags := listData["data"].([]any)
	assert.GreaterOrEqual(t, len(flags), 1)

	// Resolve flag.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/flags/"+flagID+"/resolve", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resolvedData := decodeJSON(t, resp)
	assert.Equal(t, "resolved", resolvedData["status"])

	// List flags again — resolved flags should not appear.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/flags", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	emptyList := decodeJSON(t, resp)
	emptyFlags := emptyList["data"].([]any)
	assert.Empty(t, emptyFlags)
}

// TestLive_Phase9_FlagDismiss tests flag dismissal.
func TestLive_Phase9_FlagDismiss(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, _, _, _, threadID, _ := setupPhase9Hierarchy(t, env)

	flagBody := `{"thread_id":"` + threadID + `","reason":"not an issue"}`
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/flags", flagBody)
	defer func() { _ = resp.Body.Close() }()
	flagID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/flags/"+flagID+"/dismiss", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	data := decodeJSON(t, resp)
	assert.Equal(t, "dismissed", data["status"])
}

// TestLive_Phase9_MoveThread tests moving a thread to a different board.
func TestLive_Phase9_MoveThread(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, board2ID, threadID, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	// Move thread to board2.
	moveBody := `{"target_board_id":"` + board2ID + `"}`
	resp := authReq(t, env, "POST", base+"/move", moveBody)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	movedData := decodeJSON(t, resp)
	assert.Equal(t, board2ID, movedData["board_id"])

	// GET thread at new board location succeeds.
	newBase := threadURL(env.BaseURL, orgID, spaceID, board2ID, threadID)
	resp = authReq(t, env, "GET", newBase, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// GET thread at old board location returns 404.
	resp = authReq(t, env, "GET", base, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Phase9_MergeThread tests merging one thread into another.
func TestLive_Phase9_MergeThread(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, thread2ID := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	// Add a message to the source thread.
	msgBody := `{"body":"Message in source thread","type":"comment"}`
	resp := authReq(t, env, "POST", base+"/messages", msgBody)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Merge source into target.
	mergeBody := `{"target_thread_id":"` + thread2ID + `"}`
	resp = authReq(t, env, "POST", base+"/merge", mergeBody)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Source thread should return 404.
	resp = authReq(t, env, "GET", base, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Target thread should still exist.
	targetBase := threadURL(env.BaseURL, orgID, spaceID, boardID, thread2ID)
	resp = authReq(t, env, "GET", targetBase, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestLive_Phase9_HideUnhide tests hiding and unhiding a thread.
func TestLive_Phase9_HideUnhide(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	// Hide.
	resp := authReq(t, env, "POST", base+"/hide", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	hideData := decodeJSON(t, resp)
	assert.Equal(t, true, hideData["is_hidden"])

	// Unhide.
	resp = authReq(t, env, "POST", base+"/unhide", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	unhideData := decodeJSON(t, resp)
	assert.Equal(t, false, unhideData["is_hidden"])
}

// TestLive_Phase9_VoteOnNonexistentThread tests voting on a nonexistent thread.
func TestLive_Phase9_VoteOnNonexistentThread(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, _, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, "nonexistent-thread-id")

	resp := authReq(t, env, "POST", base+"/vote", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Phase9_FlagNotFound tests resolving a nonexistent flag.
func TestLive_Phase9_FlagNotFound(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, _, _, _, _, _ := setupPhase9Hierarchy(t, env)

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/flags/nonexistent-flag/resolve", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Phase9_WeightTable tests the vote weight table endpoint.
func TestLive_Phase9_WeightTable(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "GET", env.BaseURL+"/v1/vote/weights", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "role_weights")
	assert.Contains(t, body, "tier_bonuses")
	assert.Contains(t, body, "default_weight")
}

// TestLive_Phase9_AuthRequired tests that Phase 9 endpoints require auth.
func TestLive_Phase9_AuthRequired(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"vote", "POST", base + "/vote"},
		{"hide", "POST", base + "/hide"},
		{"unhide", "POST", base + "/unhide"},
		{"move", "POST", base + "/move"},
		{"merge", "POST", base + "/merge"},
		{"flag create", "POST", env.BaseURL + "/v1/orgs/" + orgID + "/flags"},
		{"flag list", "GET", env.BaseURL + "/v1/orgs/" + orgID + "/flags"},
		{"weight table", "GET", env.BaseURL + "/v1/vote/weights"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

// TestLive_Phase9_ResponseHeaders verifies Phase 9 endpoints return proper headers.
func TestLive_Phase9_ResponseHeaders(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID, spaceID, boardID, _, threadID, _ := setupPhase9Hierarchy(t, env)
	base := threadURL(env.BaseURL, orgID, spaceID, boardID, threadID)

	resp := authReq(t, env, "POST", base+"/vote", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}
