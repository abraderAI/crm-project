package chat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abraderAI/crm-project/api/internal/cli/agent"
	"github.com/abraderAI/crm-project/api/internal/cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLLM struct {
	response string
}

func (m *mockLLM) Chat(messages []agent.Message, tools []agent.ToolSchema) (*agent.Message, error) {
	return &agent.Message{Role: "assistant", Content: m.response}, nil
}

func newTestSession(input string, llmResponse string) (*Session, *bytes.Buffer) {
	llm := &mockLLM{response: llmResponse}
	apiClient := client.New("http://localhost:9999", "", "")
	ag := agent.New(llm, apiClient, "test-org")
	output := &bytes.Buffer{}
	session := New(ag, strings.NewReader(input), output)
	return session, output
}

func TestAskOneShotQuery(t *testing.T) {
	session, _ := newTestSession("", "Here are your leads.")
	response, err := session.Ask("show me all leads")
	require.NoError(t, err)
	assert.Equal(t, "Here are your leads.", response)
}

func TestAskPreservesHistory(t *testing.T) {
	session, _ := newTestSession("", "Response 1")
	_, err := session.Ask("first query")
	require.NoError(t, err)
	assert.Len(t, session.History(), 2)

	_, err = session.Ask("second query")
	require.NoError(t, err)
	assert.True(t, len(session.History()) >= 3)
}

func TestRunREPLExit(t *testing.T) {
	session, output := newTestSession("exit\n", "")
	err := session.RunREPL()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Goodbye!")
}

func TestRunREPLQuit(t *testing.T) {
	session, output := newTestSession("quit\n", "")
	err := session.RunREPL()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Goodbye!")
}

func TestRunREPLEOF(t *testing.T) {
	session, _ := newTestSession("", "")
	err := session.RunREPL()
	assert.NoError(t, err)
}

func TestRunREPLWithQuery(t *testing.T) {
	session, output := newTestSession("show leads\nexit\n", "Found 5 leads.")
	err := session.RunREPL()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Found 5 leads.")
	assert.Contains(t, output.String(), "Goodbye!")
}

func TestRunREPLEmptyLines(t *testing.T) {
	session, output := newTestSession("\n\n\nexit\n", "")
	err := session.RunREPL()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Goodbye!")
}

func TestRunREPLBanner(t *testing.T) {
	session, output := newTestSession("exit\n", "")
	_ = session.RunREPL()
	assert.Contains(t, output.String(), "DEFT CRM AI Assistant")
	assert.Contains(t, output.String(), "exit")
}

func TestRunREPLPrompt(t *testing.T) {
	session, output := newTestSession("exit\n", "")
	_ = session.RunREPL()
	assert.Contains(t, output.String(), "deft> ")
}

func TestClearHistory(t *testing.T) {
	session, _ := newTestSession("", "response")
	_, _ = session.Ask("query")
	assert.True(t, len(session.History()) > 0)

	session.ClearHistory()
	assert.Len(t, session.History(), 0)
}

func TestHistoryEmpty(t *testing.T) {
	session, _ := newTestSession("", "")
	assert.Nil(t, session.History())
}

func TestMultipleQueriesInREPL(t *testing.T) {
	input := "query 1\nquery 2\nquery 3\nexit\n"
	session, output := newTestSession(input, "answer")
	err := session.RunREPL()
	assert.NoError(t, err)
	// Should have 3 answers.
	assert.Equal(t, 3, strings.Count(output.String(), "answer"))
}
