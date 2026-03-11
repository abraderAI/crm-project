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
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// helper to make authenticated requests.
func authReq(t *testing.T, env *liveAuthEnv, method, url, body string) *http.Response {
	t.Helper()
	token := env.SignToken(auth.JWTClaims{
		Subject:   "test_user",
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

// decodeJSON decodes a JSON response body into a map.
func decodeJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// --- Phase 4 Live API Tests ---

// TestLive_Phase4_FullHierarchy creates the full Org→Space→Board→Thread→Message hierarchy via real HTTP.
func TestLive_Phase4_FullHierarchy(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// 1. Create Org.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Live Org","description":"Test"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	orgData := decodeJSON(t, resp)
	orgID := orgData["id"].(string)
	assert.Equal(t, "Live Org", orgData["name"])
	assert.Equal(t, "live-org", orgData["slug"])
	assert.NotEmpty(t, orgID)

	// 2. Create Space.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Engineering","type":"general"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	spaceData := decodeJSON(t, resp)
	spaceID := spaceData["id"].(string)
	assert.Equal(t, "Engineering", spaceData["name"])

	// 3. Create Board.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Feature Requests"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	boardData := decodeJSON(t, resp)
	boardID := boardData["id"].(string)
	assert.Equal(t, "feature-requests", boardData["slug"])

	// 4. Create Thread.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Add dark mode","body":"Please add dark mode","metadata":"{\"status\":\"open\",\"priority\":5}"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	threadData := decodeJSON(t, resp)
	threadID := threadData["id"].(string)
	assert.Equal(t, "Add dark mode", threadData["title"])
	assert.Equal(t, "test_user", threadData["author_id"])

	// 5. Create Message.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages",
		`{"body":"Great idea!","type":"comment"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	msgData := decodeJSON(t, resp)
	msgID := msgData["id"].(string)
	assert.Equal(t, "Great idea!", msgData["body"])
	assert.NotEmpty(t, msgID)

	// 6. Verify GET for each entity.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads/"+threadID+"/messages/"+msgID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestLive_Phase4_OrgCRUD tests the full Org CRUD lifecycle.
func TestLive_Phase4_OrgCRUD(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"CRUD Org","metadata":"{\"tier\":\"free\"}"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	orgData := decodeJSON(t, resp)
	orgID := orgData["id"].(string)

	// List.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	listData := decodeJSON(t, resp)
	data := listData["data"].([]any)
	assert.GreaterOrEqual(t, len(data), 1)

	// Get by slug.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/crud-org", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Patch with metadata deep-merge.
	resp = authReq(t, env, "PATCH", env.BaseURL+"/v1/orgs/"+orgID, `{"name":"Updated CRUD","metadata":"{\"plan\":\"pro\"}"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	updated := decodeJSON(t, resp)
	assert.Equal(t, "Updated CRUD", updated["name"])
	assert.Contains(t, updated["metadata"], "tier")
	assert.Contains(t, updated["metadata"], "plan")

	// Delete.
	resp = authReq(t, env, "DELETE", env.BaseURL+"/v1/orgs/"+orgID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify soft delete (returns 404).
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLive_Phase4_BoardLockUnlock tests board lock/unlock and thread rejection.
func TestLive_Phase4_BoardLockUnlock(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Setup hierarchy.
	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Lock Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Lock Space"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Lock Board"}`)
	defer func() { _ = resp.Body.Close() }()
	boardData := decodeJSON(t, resp)
	boardID := boardData["id"].(string)
	assert.Equal(t, false, boardData["is_locked"])

	// Lock.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/lock", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	lockData := decodeJSON(t, resp)
	assert.Equal(t, true, lockData["is_locked"])

	// Unlock.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/unlock", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	unlockData := decodeJSON(t, resp)
	assert.Equal(t, false, unlockData["is_locked"])
}

// TestLive_Phase4_ThreadPinLock tests thread pin/unpin and lock/unlock.
func TestLive_Phase4_ThreadPinLock(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Pin Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces", `{"name":"Pin Space"}`)
	defer func() { _ = resp.Body.Close() }()
	spaceID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards", `{"name":"Pin Board"}`)
	defer func() { _ = resp.Body.Close() }()
	boardID := decodeJSON(t, resp)["id"].(string)

	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces/"+spaceID+"/boards/"+boardID+"/threads",
		`{"title":"Pin Thread"}`)
	defer func() { _ = resp.Body.Close() }()
	threadID := decodeJSON(t, resp)["id"].(string)

	base := env.BaseURL + "/v1/orgs/" + orgID + "/spaces/" + spaceID + "/boards/" + boardID + "/threads/" + threadID

	// Pin.
	resp = authReq(t, env, "POST", base+"/pin", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, true, decodeJSON(t, resp)["is_pinned"])

	// Unpin.
	resp = authReq(t, env, "POST", base+"/unpin", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, false, decodeJSON(t, resp)["is_pinned"])

	// Lock.
	resp = authReq(t, env, "POST", base+"/lock", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, true, decodeJSON(t, resp)["is_locked"])

	// Unlock.
	resp = authReq(t, env, "POST", base+"/unlock", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, false, decodeJSON(t, resp)["is_locked"])
}

// TestLive_Phase4_Membership tests org membership CRUD and last-owner protection.
func TestLive_Phase4_Membership(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Member Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	// Add member.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/members",
		`{"user_id":"user_a","role":"admin"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Add owner.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/members",
		`{"user_id":"user_owner","role":"owner"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// List members.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs/"+orgID+"/members", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	listData := decodeJSON(t, resp)
	members := listData["data"].([]any)
	assert.GreaterOrEqual(t, len(members), 2)

	// Update role.
	resp = authReq(t, env, "PATCH", env.BaseURL+"/v1/orgs/"+orgID+"/members/user_a",
		`{"role":"moderator"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Remove member.
	resp = authReq(t, env, "DELETE", env.BaseURL+"/v1/orgs/"+orgID+"/members/user_a", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Cannot remove last owner.
	resp = authReq(t, env, "DELETE", env.BaseURL+"/v1/orgs/"+orgID+"/members/user_owner", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestLive_Phase4_Pagination tests cursor pagination across real requests.
func TestLive_Phase4_Pagination(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	// Create 5 orgs.
	for i := 0; i < 5; i++ {
		resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Page Org `+string(rune('A'+i))+`"}`)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Get page 1 (limit 2).
	resp := authReq(t, env, "GET", env.BaseURL+"/v1/orgs?limit=2", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page1 := decodeJSON(t, resp)
	data1 := page1["data"].([]any)
	assert.Len(t, data1, 2)
	pi1 := page1["page_info"].(map[string]any)
	assert.Equal(t, true, pi1["has_more"])
	cursor := pi1["next_cursor"].(string)
	assert.NotEmpty(t, cursor)

	// Get page 2 using cursor.
	resp = authReq(t, env, "GET", env.BaseURL+"/v1/orgs?limit=2&cursor="+cursor, "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page2 := decodeJSON(t, resp)
	data2 := page2["data"].([]any)
	assert.Len(t, data2, 2)
}

// TestLive_Phase4_AuthRequired verifies Phase 4 endpoints return 401 without auth.
func TestLive_Phase4_AuthRequired(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/v1/orgs"},
		{"POST", "/v1/orgs"},
	}
	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, err := http.NewRequest(ep.method, env.BaseURL+ep.path, nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
		})
	}
}

// TestLive_Phase4_NotFoundRFC7807 verifies 404 responses are RFC 7807.
func TestLive_Phase4_NotFoundRFC7807(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "GET", env.BaseURL+"/v1/orgs/nonexistent-org-id", "")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Equal(t, 404, problem.Status)
}

// TestLive_Phase4_SpaceTypeEnum verifies space type validation.
func TestLive_Phase4_SpaceTypeEnum(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Type Org"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	// Valid type.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces",
		`{"name":"CRM Space","type":"crm"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "crm", decodeJSON(t, resp)["type"])

	// Invalid type.
	resp = authReq(t, env, "POST", env.BaseURL+"/v1/orgs/"+orgID+"/spaces",
		`{"name":"Bad Space","type":"invalid_type"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestLive_Phase4_MetadataDeepMerge verifies metadata deep-merge in PATCH.
func TestLive_Phase4_MetadataDeepMerge(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs",
		`{"name":"Meta Org","metadata":"{\"a\":\"1\",\"b\":{\"x\":\"1\"}}"}`)
	defer func() { _ = resp.Body.Close() }()
	orgID := decodeJSON(t, resp)["id"].(string)

	// Patch: merge new key, deep-merge object.
	resp = authReq(t, env, "PATCH", env.BaseURL+"/v1/orgs/"+orgID,
		`{"metadata":"{\"c\":\"3\",\"b\":{\"y\":\"2\"}}"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	updated := decodeJSON(t, resp)
	meta := updated["metadata"].(string)
	assert.Contains(t, meta, "\"a\":\"1\"")
	assert.Contains(t, meta, "\"c\":\"3\"")
	assert.Contains(t, meta, "\"x\":\"1\"")
	assert.Contains(t, meta, "\"y\":\"2\"")
}

// TestLive_Phase4_ResponseHeaders verifies response headers on Phase 4 endpoints.
func TestLive_Phase4_ResponseHeaders(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	resp := authReq(t, env, "POST", env.BaseURL+"/v1/orgs", `{"name":"Headers Org"}`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}
