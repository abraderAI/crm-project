package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abraderAI/crm-project/api/internal/cli/client"
)

// FuzzUserQuery fuzzes the agent's ability to handle arbitrary user queries.
func FuzzUserQuery(f *testing.F) {
	// 50+ seed corpus entries.
	seeds := []string{
		"show me all leads",
		"search for acme corporation",
		"list messages for thread abc",
		"update lead status to qualified",
		"get thread details for t-123",
		"search contacts named John",
		"list activities for last week",
		"move deal to closed won",
		"create new lead for company XYZ",
		"show pipeline stages",
		"what orgs do I have access to",
		"get org settings",
		"find leads with high priority",
		"show recent messages",
		"update deal stage to negotiation",
		"search all for important",
		"list boards in sales space",
		"get space details",
		"create a new thread",
		"show me contacts from last month",
		"",
		"a",
		"SELECT * FROM users",
		"<script>alert('xss')</script>",
		"'; DROP TABLE threads; --",
		"null",
		"undefined",
		"true",
		"false",
		"0",
		"-1",
		"999999999999999999999999999999",
		"🎯 emoji query",
		"very " + string(make([]byte, 1000)),
		"query with\nnewlines\n",
		"query with\ttabs",
		"query with \"quotes\"",
		"query with 'single quotes'",
		"query with {json: true}",
		"query with [array]",
		"path/traversal/../../../etc/passwd",
		"%00 null byte",
		"\\x00\\x01\\x02",
		"\r\n CRLF injection",
		"日本語クエリ",
		"Ñoño español",
		"Ünïcödë tëst",
		"Arabic: مرحبا",
		"  leading spaces",
		"trailing spaces  ",
		"  both  spaces  ",
		"multi  word  query  with  spaces",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, query string) {
		llm := &fuzzMockLLM{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"data":      []any{},
				"page_info": map[string]any{"has_more": false},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		apiClient := client.NewWithHTTPClient(srv.URL, "X-API-Key", "key", srv.Client())
		ag := New(llm, apiClient, "test-org")

		// Should not panic.
		response, _, err := ag.Process(nil, query)
		if err == nil {
			_ = response
		}
	})
}

// FuzzAPIResponse fuzzes the agent's handling of arbitrary API responses.
func FuzzAPIResponse(f *testing.F) {
	// 50+ seed corpus entries of API response bodies.
	seeds := []string{
		`{"data":[],"page_info":{"has_more":false}}`,
		`{"data":[{"id":"1","name":"Test"}],"page_info":{"has_more":false}}`,
		`{"data":[{"id":"1","title":"Lead","status":"new"}],"page_info":{"has_more":true,"next_cursor":"abc"}}`,
		`{"type":"https://httpstatuses.com/404","title":"Not Found","status":404,"detail":"not found"}`,
		`{"type":"https://httpstatuses.com/500","title":"Internal Server Error","status":500}`,
		`{"type":"https://httpstatuses.com/400","title":"Bad Request","status":400,"detail":"invalid input"}`,
		`{}`,
		`[]`,
		`null`,
		`""`,
		`{"data":null}`,
		`{"data":"not an array"}`,
		`{"data":123}`,
		`{"data":true}`,
		`{"unexpected":"field"}`,
		`{"data":[{"nested":{"deep":{"value":"test"}}}]}`,
		`{"data":[{"metadata":"{\"stage\":\"new_lead\"}"}]}`,
		`{"data":[{}]}`,
		`{"data":[{"id":""}]}`,
		`{"data":[{"id":null}]}`,
		`{"page_info":{"has_more":true}}`,
		`{"data":[],"page_info":null}`,
		`{"data":[],"page_info":{"has_more":false,"next_cursor":""}}`,
		`not json at all`,
		`<html>error page</html>`,
		``,
		`{`,
		`}`,
		`[`,
		`]`,
		`{"data":[{"title":"` + string(make([]byte, 500)) + `"}]}`,
		`{"data":[{"id":"1"},{"id":"2"},{"id":"3"},{"id":"4"},{"id":"5"}],"page_info":{"has_more":false}}`,
		`{"data":[{"special":"chars: <>&\"'"}]}`,
		`{"data":[{"unicode":"日本語テスト"}]}`,
		`{"data":[{"emoji":"🎯🚀💡"}]}`,
		`{"data":[{"number":42}]}`,
		`{"data":[{"float":3.14159}]}`,
		`{"data":[{"bool":true}]}`,
		`{"data":[{"array":[1,2,3]}]}`,
		`{"data":[{"null_field":null}]}`,
		`{"data":[{"empty_string":""}]}`,
		`{"data":[{"whitespace":"   "}]}`,
		`{"data":[{"very_long_key_name_that_goes_on_and_on_and_on":"value"}]}`,
		`{"data":[{"a":"1","b":"2","c":"3","d":"4","e":"5","f":"6","g":"7","h":"8","i":"9","j":"10"}]}`,
		`{"data":[{"metadata":"{}"}]}`,
		`{"data":[{"metadata":"{\"key\":\"value\"}"}]}`,
		`{"data":[{"id":"org-1","slug":"my-org","name":"Test Org","description":"A test org","metadata":"{}","created_at":"2024-01-01T00:00:00Z"}]}`,
		`{"data":[{"id":"thread-1","title":"Lead","body":"Description","author_id":"user-1","board_id":"board-1","metadata":"{\"stage\":\"qualified\",\"score\":85}"}]}`,
		`{"status":"ok","version":"v1"}`,
		`{"error":"unauthorized"}`,
		`{"message":"rate limited","retry_after":60}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, responseBody string) {
		llm := &fuzzToolCallLLM{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(responseBody))
		}))
		defer srv.Close()

		apiClient := client.NewWithHTTPClient(srv.URL, "X-API-Key", "key", srv.Client())
		ag := New(llm, apiClient, "test-org")

		// Should not panic regardless of API response content.
		response, _, err := ag.Process(nil, "search for test")
		if err == nil {
			_ = response
		}
	})
}

// fuzzMockLLM returns a direct text response for fuzzing user queries.
type fuzzMockLLM struct{}

func (m *fuzzMockLLM) Chat(messages []Message, tools []ToolSchema) (*Message, error) {
	return &Message{Role: "assistant", Content: "response"}, nil
}

// fuzzToolCallLLM returns a search tool call then a text response, for fuzzing API responses.
type fuzzToolCallLLM struct {
	callCount int
}

func (m *fuzzToolCallLLM) Chat(messages []Message, tools []ToolSchema) (*Message, error) {
	m.callCount++
	if m.callCount == 1 {
		return &Message{
			Role:      "assistant",
			ToolCalls: []ToolCall{{Name: "search_all", Arguments: map[string]any{"query": "test"}}},
		}, nil
	}
	return &Message{Role: "assistant", Content: "done"}, nil
}
