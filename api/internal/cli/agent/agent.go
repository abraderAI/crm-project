// Package agent provides LLM function calling for the CLI, mapping natural language
// queries to CRM API operations via tool function schemas.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abraderAI/crm-project/api/internal/cli/client"
)

// ToolCall represents a function call returned by the LLM.
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolResult holds the result of executing a tool call.
type ToolResult struct {
	ToolName string `json:"tool_name"`
	Data     any    `json:"data,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Message represents a conversation message for the LLM.
type Message struct {
	Role       string      `json:"role"` // "user", "assistant", "tool"
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolSchema defines a function schema for the LLM.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// LLMProvider abstracts the LLM interaction for function calling.
type LLMProvider interface {
	// Chat sends messages and tool schemas, returning the LLM response (which may include tool calls).
	Chat(messages []Message, tools []ToolSchema) (*Message, error)
}

// Agent orchestrates LLM function calling against the CRM API.
type Agent struct {
	llm        LLMProvider
	apiClient  *client.Client
	defaultOrg string
}

// New creates a new Agent.
func New(llm LLMProvider, apiClient *client.Client, defaultOrg string) *Agent {
	return &Agent{
		llm:        llm,
		apiClient:  apiClient,
		defaultOrg: defaultOrg,
	}
}

// MaxToolRounds limits the number of consecutive tool call rounds to prevent infinite loops.
const MaxToolRounds = 10

// Process sends a user query to the LLM with tool schemas, executes any tool calls,
// and returns the final text response. Supports multi-step function call chains.
func (a *Agent) Process(history []Message, userQuery string) (string, []Message, error) {
	messages := make([]Message, len(history))
	copy(messages, history)
	messages = append(messages, Message{Role: "user", Content: userQuery})

	tools := GetToolSchemas()

	for round := 0; round < MaxToolRounds; round++ {
		resp, err := a.llm.Chat(messages, tools)
		if err != nil {
			return "", nil, fmt.Errorf("LLM chat error: %w", err)
		}

		messages = append(messages, *resp)

		if len(resp.ToolCalls) == 0 {
			// No tool calls — return the text response.
			return resp.Content, messages, nil
		}

		// Execute each tool call and feed results back.
		for _, tc := range resp.ToolCalls {
			result := a.executeTool(tc)
			messages = append(messages, Message{
				Role:       "tool",
				ToolResult: &result,
			})
		}
	}

	return "I was unable to complete your request within the allowed number of steps.", messages, nil
}

// executeTool dispatches a tool call to the appropriate API client method.
func (a *Agent) executeTool(tc ToolCall) ToolResult {
	result := ToolResult{ToolName: tc.Name}

	switch tc.Name {
	case "search_leads":
		data, err := a.searchLeads(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "get_lead":
		data, err := a.getLead(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "create_lead":
		data, err := a.createLead(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "update_lead":
		data, err := a.updateLead(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "list_messages":
		data, err := a.listMessages(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "get_thread":
		data, err := a.getThread(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "search_contacts":
		data, err := a.searchContacts(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "list_activities":
		data, err := a.listActivities(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "update_deal_stage":
		data, err := a.updateDealStage(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "search_all":
		data, err := a.searchAll(tc.Arguments)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "list_orgs":
		data, _, err := a.apiClient.ListOrgs(nil)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	case "get_org":
		orgRef := stringArg(tc.Arguments, "org_id")
		if orgRef == "" {
			orgRef = a.defaultOrg
		}
		data, err := a.apiClient.GetOrg(orgRef)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Data = data
		}

	default:
		result.Error = fmt.Sprintf("unknown tool: %s", tc.Name)
	}

	return result
}

func (a *Agent) searchLeads(args map[string]any) (any, error) {
	query := stringArg(args, "query")
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	data, _, err := a.apiClient.Search(query, &client.ListParams{
		Limit: intArg(args, "limit", 20),
	})
	return data, err
}

func (a *Agent) getLead(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	thread := stringArg(args, "thread_id")
	if thread == "" {
		return nil, fmt.Errorf("thread_id is required")
	}
	return a.apiClient.GetThread(org, space, board, thread)
}

func (a *Agent) createLead(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	body := mapArg(args, "body")
	if body == nil {
		body = map[string]any{}
	}
	if title := stringArg(args, "title"); title != "" {
		body["title"] = title
	}
	return a.apiClient.CreateThread(org, space, board, body)
}

func (a *Agent) updateLead(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	thread := stringArg(args, "thread_id")
	body := mapArg(args, "body")
	if body == nil {
		body = map[string]any{}
	}
	return a.apiClient.UpdateThread(org, space, board, thread, body)
}

func (a *Agent) listMessages(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	thread := stringArg(args, "thread_id")
	data, _, err := a.apiClient.ListMessages(org, space, board, thread, &client.ListParams{
		Limit: intArg(args, "limit", 20),
	})
	return data, err
}

func (a *Agent) getThread(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	thread := stringArg(args, "thread_id")
	return a.apiClient.GetThread(org, space, board, thread)
}

func (a *Agent) searchContacts(args map[string]any) (any, error) {
	query := stringArg(args, "query")
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	params := &client.ListParams{
		Limit: intArg(args, "limit", 20),
		Query: map[string]string{"type": "thread"},
	}
	data, _, err := a.apiClient.Search(query, params)
	return data, err
}

func (a *Agent) listActivities(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	thread := stringArg(args, "thread_id")
	data, _, err := a.apiClient.ListMessages(org, space, board, thread, &client.ListParams{
		Limit: intArg(args, "limit", 50),
	})
	return data, err
}

func (a *Agent) updateDealStage(args map[string]any) (any, error) {
	org := a.orgArg(args)
	space := stringArg(args, "space_id")
	board := stringArg(args, "board_id")
	thread := stringArg(args, "thread_id")
	stage := stringArg(args, "stage")
	if stage == "" {
		return nil, fmt.Errorf("stage is required")
	}
	return a.apiClient.TransitionStage(org, space, board, thread, stage)
}

func (a *Agent) searchAll(args map[string]any) (any, error) {
	query := stringArg(args, "query")
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	data, _, err := a.apiClient.Search(query, &client.ListParams{
		Limit: intArg(args, "limit", 20),
	})
	return data, err
}

func (a *Agent) orgArg(args map[string]any) string {
	org := stringArg(args, "org_id")
	if org == "" {
		return a.defaultOrg
	}
	return org
}

// --- Argument helpers ---

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func intArg(args map[string]any, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return int(i)
			}
		}
	}
	return defaultVal
}

func mapArg(args map[string]any, key string) map[string]any {
	if v, ok := args[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

// GetToolSchemas returns the set of tool function schemas for LLM function calling.
func GetToolSchemas() []ToolSchema {
	return []ToolSchema{
		{
			Name:        "search_leads",
			Description: "Search for leads/threads across the CRM by keyword or metadata",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query text"},
					"limit": map[string]any{"type": "integer", "description": "Max results to return"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_lead",
			Description: "Get a specific lead/thread by its IDs",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":    map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id":  map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id":  map[string]any{"type": "string", "description": "Board ID or slug"},
					"thread_id": map[string]any{"type": "string", "description": "Thread ID or slug"},
				},
				"required": []string{"thread_id"},
			},
		},
		{
			Name:        "create_lead",
			Description: "Create a new lead/thread in a board",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":   map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id": map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id": map[string]any{"type": "string", "description": "Board ID or slug"},
					"title":    map[string]any{"type": "string", "description": "Lead title"},
					"body":     map[string]any{"type": "object", "description": "Full request body"},
				},
				"required": []string{"space_id", "board_id", "title"},
			},
		},
		{
			Name:        "update_lead",
			Description: "Update an existing lead/thread",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":    map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id":  map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id":  map[string]any{"type": "string", "description": "Board ID or slug"},
					"thread_id": map[string]any{"type": "string", "description": "Thread ID or slug"},
					"body":      map[string]any{"type": "object", "description": "Fields to update"},
				},
				"required": []string{"thread_id", "body"},
			},
		},
		{
			Name:        "list_messages",
			Description: "List messages/activities in a thread",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":    map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id":  map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id":  map[string]any{"type": "string", "description": "Board ID or slug"},
					"thread_id": map[string]any{"type": "string", "description": "Thread ID or slug"},
					"limit":     map[string]any{"type": "integer", "description": "Max results"},
				},
				"required": []string{"thread_id"},
			},
		},
		{
			Name:        "get_thread",
			Description: "Get full details of a thread including metadata",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":    map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id":  map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id":  map[string]any{"type": "string", "description": "Board ID or slug"},
					"thread_id": map[string]any{"type": "string", "description": "Thread ID or slug"},
				},
				"required": []string{"thread_id"},
			},
		},
		{
			Name:        "search_contacts",
			Description: "Search for contacts across the CRM",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Contact search query"},
					"limit": map[string]any{"type": "integer", "description": "Max results"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_activities",
			Description: "List recent activities/messages for a thread",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":    map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id":  map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id":  map[string]any{"type": "string", "description": "Board ID or slug"},
					"thread_id": map[string]any{"type": "string", "description": "Thread ID or slug"},
					"limit":     map[string]any{"type": "integer", "description": "Max results"},
				},
				"required": []string{"thread_id"},
			},
		},
		{
			Name:        "update_deal_stage",
			Description: "Transition a deal/lead to a new pipeline stage",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id":    map[string]any{"type": "string", "description": "Org ID or slug"},
					"space_id":  map[string]any{"type": "string", "description": "Space ID or slug"},
					"board_id":  map[string]any{"type": "string", "description": "Board ID or slug"},
					"thread_id": map[string]any{"type": "string", "description": "Thread ID or slug"},
					"stage":     map[string]any{"type": "string", "description": "Target pipeline stage"},
				},
				"required": []string{"thread_id", "stage"},
			},
		},
		{
			Name:        "search_all",
			Description: "Search across all entity types (orgs, spaces, boards, threads, messages)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query"},
					"limit": map[string]any{"type": "integer", "description": "Max results"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_orgs",
			Description: "List all accessible organizations",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_org",
			Description: "Get details of a specific organization",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org_id": map[string]any{"type": "string", "description": "Org ID or slug"},
				},
			},
		},
	}
}
