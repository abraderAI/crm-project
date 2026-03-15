package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	srv := httptest.NewServer(handler)
	c := NewWithHTTPClient(srv.URL, "X-API-Key", "test-key", srv.Client())
	return srv, c
}

func TestListOrgs(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		resp := ListResponse{
			Data:     json.RawMessage(`[{"id":"org-1","name":"Test Org"}]`),
			PageInfo: &PageInfo{HasMore: false},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	orgs, pi, err := c.ListOrgs(nil)
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
	assert.Equal(t, "org-1", orgs[0]["id"])
	assert.False(t, pi.HasMore)
}

func TestListOrgsWithPagination(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		assert.Equal(t, "cursor-abc", r.URL.Query().Get("cursor"))
		resp := ListResponse{
			Data:     json.RawMessage(`[{"id":"org-2"}]`),
			PageInfo: &PageInfo{NextCursor: "next-cursor", HasMore: true},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	orgs, pi, err := c.ListOrgs(&ListParams{Cursor: "cursor-abc", Limit: 5})
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
	assert.True(t, pi.HasMore)
	assert.Equal(t, "next-cursor", pi.NextCursor)
}

func TestGetOrg(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs/my-org", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "org-1", "name": "My Org"})
	})
	defer srv.Close()

	org, err := c.GetOrg("my-org")
	require.NoError(t, err)
	assert.Equal(t, "My Org", org["name"])
}

func TestCreateOrg(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/orgs", r.URL.Path)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Org", body["name"])
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-org", "name": "New Org"})
	})
	defer srv.Close()

	org, err := c.CreateOrg(map[string]any{"name": "New Org"})
	require.NoError(t, err)
	assert.Equal(t, "new-org", org["id"])
}

func TestUpdateOrg(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "org-1", "name": "Updated"})
	})
	defer srv.Close()

	org, err := c.UpdateOrg("org-1", map[string]any{"name": "Updated"})
	require.NoError(t, err)
	assert.Equal(t, "Updated", org["name"])
}

func TestListSpaces(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs/org-1/spaces", r.URL.Path)
		resp := ListResponse{Data: json.RawMessage(`[{"id":"sp-1"}]`), PageInfo: &PageInfo{}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	spaces, _, err := c.ListSpaces("org-1", nil)
	require.NoError(t, err)
	assert.Len(t, spaces, 1)
}

func TestGetSpace(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs/org-1/spaces/sp-1", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "sp-1"})
	})
	defer srv.Close()

	space, err := c.GetSpace("org-1", "sp-1")
	require.NoError(t, err)
	assert.Equal(t, "sp-1", space["id"])
}

func TestListBoards(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs/org-1/spaces/sp-1/boards", r.URL.Path)
		resp := ListResponse{Data: json.RawMessage(`[{"id":"bd-1"}]`), PageInfo: &PageInfo{}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	boards, _, err := c.ListBoards("org-1", "sp-1", nil)
	require.NoError(t, err)
	assert.Len(t, boards, 1)
}

func TestListThreads(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs/org-1/spaces/sp-1/boards/bd-1/threads", r.URL.Path)
		resp := ListResponse{Data: json.RawMessage(`[{"id":"th-1","title":"Lead 1"}]`), PageInfo: &PageInfo{}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	threads, _, err := c.ListThreads("org-1", "sp-1", "bd-1", nil)
	require.NoError(t, err)
	assert.Len(t, threads, 1)
	assert.Equal(t, "Lead 1", threads[0]["title"])
}

func TestGetThread(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/orgs/o/spaces/s/boards/b/threads/t", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t", "title": "Thread"})
	})
	defer srv.Close()

	thread, err := c.GetThread("o", "s", "b", "t")
	require.NoError(t, err)
	assert.Equal(t, "Thread", thread["title"])
}

func TestCreateThread(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-th", "title": "New Lead"})
	})
	defer srv.Close()

	thread, err := c.CreateThread("o", "s", "b", map[string]any{"title": "New Lead"})
	require.NoError(t, err)
	assert.Equal(t, "new-th", thread["id"])
}

func TestListMessages(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/messages")
		resp := ListResponse{Data: json.RawMessage(`[{"id":"msg-1","body":"Hello"}]`), PageInfo: &PageInfo{}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	msgs, _, err := c.ListMessages("o", "s", "b", "t", nil)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}

func TestSearch(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, "test query", r.URL.Query().Get("q"))
		resp := ListResponse{Data: json.RawMessage(`[{"id":"result-1"}]`), PageInfo: &PageInfo{}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, _, err := c.Search("test query", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestTransitionStage(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/stage")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "qualified", body["stage"])
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t", "stage": "qualified"})
	})
	defer srv.Close()

	result, err := c.TransitionStage("o", "s", "b", "t", "qualified")
	require.NoError(t, err)
	assert.Equal(t, "qualified", result["stage"])
}

func TestAPIErrorResponse(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "https://httpstatuses.com/404",
			"title":  "Not Found",
			"status": 404,
			"detail": "org not found",
		})
	})
	defer srv.Close()

	_, err := c.GetOrg("missing")
	assert.Error(t, err)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Status)
	assert.Contains(t, apiErr.Error(), "Not Found")
	assert.Contains(t, apiErr.Error(), "org not found")
}

func TestAPIErrorNoTitle(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	})
	defer srv.Close()

	_, err := c.GetOrg("bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestBuildListURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		params   *ListParams
		contains []string
	}{
		{"nil params", "/v1/orgs", nil, []string{"/v1/orgs"}},
		{"with cursor", "/v1/orgs", &ListParams{Cursor: "abc"}, []string{"cursor=abc"}},
		{"with limit", "/v1/orgs", &ListParams{Limit: 10}, []string{"limit=10"}},
		{"with query", "/v1/search", &ListParams{Query: map[string]string{"q": "test"}}, []string{"q=test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildListURL(tt.path, tt.params)
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
		})
	}
}

func TestListAllPagination(t *testing.T) {
	calls := 0
	result, err := ListAll(func(cursor string) ([]string, *PageInfo, error) {
		calls++
		if calls == 1 {
			assert.Empty(t, cursor)
			return []string{"a", "b"}, &PageInfo{NextCursor: "page2", HasMore: true}, nil
		}
		assert.Equal(t, "page2", cursor)
		return []string{"c"}, &PageInfo{HasMore: false}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result)
	assert.Equal(t, 2, calls)
}

func TestRawGet(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"raw":"data"}`))
	})
	defer srv.Close()

	data, err := c.RawGet("/v1/test")
	require.NoError(t, err)
	assert.Contains(t, string(data), "raw")
}

func TestCreateSpaceAndBoard(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-1"})
	})
	defer srv.Close()

	space, err := c.CreateSpace("o", map[string]any{"name": "S"})
	require.NoError(t, err)
	assert.Equal(t, "new-1", space["id"])

	board, err := c.CreateBoard("o", "s", map[string]any{"name": "B"})
	require.NoError(t, err)
	assert.Equal(t, "new-1", board["id"])
}

func TestUpdateSpaceAndBoard(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "up-1", "name": "Updated"})
	})
	defer srv.Close()

	space, err := c.UpdateSpace("o", "s", map[string]any{"name": "Updated"})
	require.NoError(t, err)
	assert.Equal(t, "Updated", space["name"])

	board, err := c.UpdateBoard("o", "s", "b", map[string]any{"name": "Updated"})
	require.NoError(t, err)
	assert.Equal(t, "Updated", board["name"])
}

func TestUpdateThread(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t", "title": "Updated"})
	})
	defer srv.Close()

	thread, err := c.UpdateThread("o", "s", "b", "t", map[string]any{"title": "Updated"})
	require.NoError(t, err)
	assert.Equal(t, "Updated", thread["title"])
}

func TestGetBoard(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "b-1"})
	})
	defer srv.Close()

	board, err := c.GetBoard("o", "s", "b-1")
	require.NoError(t, err)
	assert.Equal(t, "b-1", board["id"])
}

func TestGetMessage(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "msg-1", "body": "Hello"})
	})
	defer srv.Close()

	msg, err := c.GetMessage("o", "s", "b", "t", "msg-1")
	require.NoError(t, err)
	assert.Equal(t, "Hello", msg["body"])
}

func TestAuthHeaderInjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-jwt", r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, "Authorization", "Bearer my-jwt", srv.Client())
	_, err := c.GetOrg("test")
	assert.NoError(t, err)
}

func TestDirectArrayResponse(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Return a direct array instead of a wrapped response.
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}, {"id": "2"}})
	})
	defer srv.Close()

	orgs, _, err := c.ListOrgs(nil)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

func TestSearchWithParams(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "hello", r.URL.Query().Get("q"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		resp := ListResponse{Data: json.RawMessage(`[]`), PageInfo: &PageInfo{}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	_, _, err := c.Search("hello", &ListParams{Limit: 5})
	require.NoError(t, err)
}
