// Package agent provides LLM function calling for the CLI, mapping natural language
// queries to CRM API operations via tool function schemas.
package agent

import (
	"fmt"

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
