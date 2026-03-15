package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abraderAI/crm-project/api/internal/cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLM implements LLMProvider for testing.
type mockLLM struct {
	responses []Message
	callCount int
}

func (m *mockLLM) Chat(messages []Message, tools []ToolSchema) (*Message, error) {
	if m.callCount >= len(m.responses) {
		return &Message{Role: "assistant", Content: "done"}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func newTestAPIClient(handler http.HandlerFunc) (*httptest.Server, *client.Client) {
	srv := httptest.NewServer(handler)
	c := client.NewWithHTTPClient(srv.URL, "X-API-Key", "test-key", srv.Client())
	return srv, c
}

func TestProcessDirectResponse(t *testing.T) {
	llm := &mockLLM{
		responses: []Message{
			{Role: "assistant", Content: "Here are your leads."},
		},
	}
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	defer srv.Close()

	ag := New(llm, apiClient, "test-org")
	response, history, err := ag.Process(nil, "show me leads")
	require.NoError(t, err)
	assert.Equal(t, "Here are your leads.", response)
	assert.Len(t, history, 2) // user + assistant
}

func TestProcessWithToolCall(t *testing.T) {
	llm := &mockLLM{
		responses: []Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{Name: "search_leads", Arguments: map[string]any{"query": "acme"}},
				},
			},
			{Role: "assistant", Content: "Found 1 lead matching 'acme'."},
		},
	}
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{map[string]any{"id": "lead-1", "title": "Acme Corp"}},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(llm, apiClient, "test-org")
	response, _, err := ag.Process(nil, "search for acme")
	require.NoError(t, err)
	assert.Contains(t, response, "acme")
}

func TestProcessMultiStepToolCalls(t *testing.T) {
	llm := &mockLLM{
		responses: []Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{Name: "list_orgs", Arguments: map[string]any{}},
				},
			},
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{Name: "get_org", Arguments: map[string]any{"org_id": "org-1"}},
				},
			},
			{Role: "assistant", Content: "Org details retrieved."},
		},
	}
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/orgs" {
			resp := map[string]any{
				"data":      []any{map[string]any{"id": "org-1"}},
				"page_info": map[string]any{"has_more": false},
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "org-1", "name": "Test"})
		}
	})
	defer srv.Close()

	ag := New(llm, apiClient, "test-org")
	response, history, err := ag.Process(nil, "get org details")
	require.NoError(t, err)
	assert.Equal(t, "Org details retrieved.", response)
	// user + toolcall1 + toolresult1 + toolcall2 + toolresult2 + final
	assert.True(t, len(history) >= 5)
}

func TestExecuteToolSearchLeads(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{map[string]any{"id": "t-1"}},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "search_leads", Arguments: map[string]any{"query": "test"}})
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Data)
}

func TestExecuteToolSearchLeadsMissingQuery(t *testing.T) {
	ag := New(nil, nil, "org")
	result := ag.executeTool(ToolCall{Name: "search_leads", Arguments: map[string]any{}})
	assert.Contains(t, result.Error, "query is required")
}

func TestExecuteToolGetLead(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t-1", "title": "Lead"})
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "get_lead", Arguments: map[string]any{
		"thread_id": "t-1", "space_id": "s", "board_id": "b",
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolGetLeadMissingThread(t *testing.T) {
	ag := New(nil, nil, "org")
	result := ag.executeTool(ToolCall{Name: "get_lead", Arguments: map[string]any{}})
	assert.Contains(t, result.Error, "thread_id is required")
}

func TestExecuteToolCreateLead(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-t"})
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "create_lead", Arguments: map[string]any{
		"space_id": "s", "board_id": "b", "title": "New Lead",
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolUpdateLead(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t-1"})
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "update_lead", Arguments: map[string]any{
		"thread_id": "t-1", "space_id": "s", "board_id": "b",
		"body": map[string]any{"title": "Updated"},
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolListMessages(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{map[string]any{"id": "msg-1"}},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "list_messages", Arguments: map[string]any{
		"thread_id": "t-1", "space_id": "s", "board_id": "b",
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolGetThread(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t-1"})
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "get_thread", Arguments: map[string]any{
		"thread_id": "t-1", "space_id": "s", "board_id": "b",
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolSearchContacts(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "search_contacts", Arguments: map[string]any{"query": "john"}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolSearchContactsMissingQuery(t *testing.T) {
	ag := New(nil, nil, "org")
	result := ag.executeTool(ToolCall{Name: "search_contacts", Arguments: map[string]any{}})
	assert.Contains(t, result.Error, "query is required")
}

func TestExecuteToolListActivities(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "list_activities", Arguments: map[string]any{
		"thread_id": "t-1", "space_id": "s", "board_id": "b",
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolUpdateDealStage(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t-1", "stage": "qualified"})
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "update_deal_stage", Arguments: map[string]any{
		"thread_id": "t-1", "space_id": "s", "board_id": "b", "stage": "qualified",
	}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolUpdateDealStageMissing(t *testing.T) {
	ag := New(nil, nil, "org")
	result := ag.executeTool(ToolCall{Name: "update_deal_stage", Arguments: map[string]any{
		"thread_id": "t-1",
	}})
	assert.Contains(t, result.Error, "stage is required")
}

func TestExecuteToolSearchAll(t *testing.T) {
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(nil, apiClient, "org")
	result := ag.executeTool(ToolCall{Name: "search_all", Arguments: map[string]any{"query": "test"}})
	assert.Empty(t, result.Error)
}

func TestExecuteToolSearchAllMissingQuery(t *testing.T) {
	ag := New(nil, nil, "org")
	result := ag.executeTool(ToolCall{Name: "search_all", Arguments: map[string]any{}})
	assert.Contains(t, result.Error, "query is required")
}

func TestExecuteToolUnknown(t *testing.T) {
	ag := New(nil, nil, "org")
	result := ag.executeTool(ToolCall{Name: "unknown_tool", Arguments: map[string]any{}})
	assert.Contains(t, result.Error, "unknown tool")
}

func TestGetToolSchemas(t *testing.T) {
	schemas := GetToolSchemas()
	assert.True(t, len(schemas) >= 10)

	names := make(map[string]bool)
	for _, s := range schemas {
		names[s.Name] = true
		assert.NotEmpty(t, s.Description)
		assert.NotNil(t, s.Parameters)
	}

	expectedTools := []string{
		"search_leads", "get_lead", "create_lead", "update_lead",
		"list_messages", "get_thread", "search_contacts", "list_activities",
		"update_deal_stage", "search_all", "list_orgs", "get_org",
	}
	for _, name := range expectedTools {
		assert.True(t, names[name], "missing tool: %s", name)
	}
}

func TestStringArg(t *testing.T) {
	args := map[string]any{"key": "value", "num": 42, "spaces": "  trimmed  "}
	assert.Equal(t, "value", stringArg(args, "key"))
	assert.Equal(t, "", stringArg(args, "missing"))
	assert.Equal(t, "", stringArg(args, "num"))
	assert.Equal(t, "trimmed", stringArg(args, "spaces"))
}

func TestIntArg(t *testing.T) {
	args := map[string]any{"float": 42.0, "int": 10, "str": "not a number"}
	assert.Equal(t, 42, intArg(args, "float", 0))
	assert.Equal(t, 10, intArg(args, "int", 0))
	assert.Equal(t, 5, intArg(args, "missing", 5))
	assert.Equal(t, 5, intArg(args, "str", 5))
}

func TestMapArg(t *testing.T) {
	m := map[string]any{"title": "Updated"}
	args := map[string]any{"body": m, "other": "string"}
	assert.Equal(t, m, mapArg(args, "body"))
	assert.Nil(t, mapArg(args, "other"))
	assert.Nil(t, mapArg(args, "missing"))
}

func TestOrgArgDefault(t *testing.T) {
	ag := New(nil, nil, "default-org")
	assert.Equal(t, "default-org", ag.orgArg(map[string]any{}))
	assert.Equal(t, "custom", ag.orgArg(map[string]any{"org_id": "custom"}))
}

func TestProcessLLMError(t *testing.T) {
	llm := &errorLLM{}
	ag := New(llm, nil, "org")
	_, _, err := ag.Process(nil, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM chat error")
}

type errorLLM struct{}

func (e *errorLLM) Chat(messages []Message, tools []ToolSchema) (*Message, error) {
	return nil, fmt.Errorf("LLM unavailable")
}

func TestProcessMaxRoundsExceeded(t *testing.T) {
	// LLM always returns tool calls, never a text response.
	llm := &infiniteToolCallLLM{}
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data":      []any{},
			"page_info": map[string]any{"has_more": false},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ag := New(llm, apiClient, "org")
	response, _, err := ag.Process(nil, "test")
	require.NoError(t, err)
	assert.Contains(t, response, "unable to complete")
}

type infiniteToolCallLLM struct{}

func (i *infiniteToolCallLLM) Chat(messages []Message, tools []ToolSchema) (*Message, error) {
	return &Message{
		Role:      "assistant",
		ToolCalls: []ToolCall{{Name: "search_all", Arguments: map[string]any{"query": "test"}}},
	}, nil
}

func TestProcessPreservesHistory(t *testing.T) {
	llm := &mockLLM{
		responses: []Message{
			{Role: "assistant", Content: "First response"},
		},
	}
	srv, apiClient := newTestAPIClient(func(w http.ResponseWriter, r *http.Request) {})
	defer srv.Close()

	ag := New(llm, apiClient, "org")
	history := []Message{{Role: "user", Content: "previous"}, {Role: "assistant", Content: "old response"}}
	_, newHistory, err := ag.Process(history, "new query")
	require.NoError(t, err)
	// Should include previous history + new user + new assistant
	assert.True(t, len(newHistory) > len(history))
}
