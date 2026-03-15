// Package chat provides interactive REPL and one-shot query modes for the CLI.
package chat

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/abraderAI/crm-project/api/internal/cli/agent"
)

// Session maintains conversation state across queries.
type Session struct {
	agent   *agent.Agent
	history []agent.Message
	input   io.Reader
	output  io.Writer
}

// New creates a new chat session.
func New(ag *agent.Agent, input io.Reader, output io.Writer) *Session {
	return &Session{
		agent:  ag,
		input:  input,
		output: output,
	}
}

// Ask processes a single query and returns the response.
// This is used for one-shot mode (deft ask 'query').
func (s *Session) Ask(query string) (string, error) {
	response, history, err := s.agent.Process(s.history, query)
	if err != nil {
		return "", err
	}
	s.history = history
	return response, nil
}

// RunREPL starts an interactive REPL loop.
// Returns when the user types "exit", "quit", or sends EOF (Ctrl+D).
func (s *Session) RunREPL() error {
	scanner := bufio.NewScanner(s.input)

	fmt.Fprintln(s.output, "DEFT CRM AI Assistant")
	fmt.Fprintln(s.output, "Type your questions in natural language. Type 'exit' or 'quit' to leave.")
	fmt.Fprintln(s.output)

	for {
		fmt.Fprint(s.output, "deft> ")

		if !scanner.Scan() {
			// EOF or error.
			fmt.Fprintln(s.output)
			return scanner.Err()
		}

		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue
		}

		lower := strings.ToLower(query)
		if lower == "exit" || lower == "quit" {
			fmt.Fprintln(s.output, "Goodbye!")
			return nil
		}

		response, err := s.Ask(query)
		if err != nil {
			fmt.Fprintf(s.output, "Error: %s\n\n", err)
			continue
		}

		fmt.Fprintln(s.output, response)
		fmt.Fprintln(s.output)
	}
}

// History returns the current conversation history.
func (s *Session) History() []agent.Message {
	return s.history
}

// ClearHistory resets the conversation history.
func (s *Session) ClearHistory() {
	s.history = nil
}
