// Package output provides terminal output formatting for the CLI including
// table, JSON, and detail views with optional color.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tw "github.com/olekukonko/tablewriter"
)

// Format specifies the output format.
type Format string

const (
	// FormatTable renders data as an ASCII table.
	FormatTable Format = "table"
	// FormatJSON renders data as raw JSON.
	FormatJSON Format = "json"
)

// Formatter handles output rendering.
type Formatter struct {
	writer io.Writer
	format Format
	color  bool
}

// New creates a new Formatter.
func New(w io.Writer, format Format) *Formatter {
	color := true
	if os.Getenv("NO_COLOR") != "" {
		color = false
	}
	return &Formatter{writer: w, format: format, color: color}
}

// NewWithColor creates a Formatter with explicit color setting.
func NewWithColor(w io.Writer, format Format, color bool) *Formatter {
	return &Formatter{writer: w, format: format, color: color}
}

// RenderList renders a list of entities.
func (f *Formatter) RenderList(entities []map[string]any) error {
	if f.format == FormatJSON {
		return f.renderJSON(entities)
	}
	return f.renderTable(entities)
}

// RenderDetail renders a single entity in detail view.
func (f *Formatter) RenderDetail(entity map[string]any) error {
	if f.format == FormatJSON {
		return f.renderJSON(entity)
	}
	return f.renderDetailView(entity)
}

// RenderText renders plain text output.
func (f *Formatter) RenderText(text string) {
	if f.color {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
		fmt.Fprintln(f.writer, style.Render(text))
	} else {
		fmt.Fprintln(f.writer, text)
	}
}

// RenderError renders an error message.
func (f *Formatter) RenderError(msg string) {
	if f.color {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		fmt.Fprintln(f.writer, style.Render("Error: "+msg))
	} else {
		fmt.Fprintln(f.writer, "Error: "+msg)
	}
}

// RenderSuccess renders a success message.
func (f *Formatter) RenderSuccess(msg string) {
	if f.color {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		fmt.Fprintln(f.writer, style.Render(msg))
	} else {
		fmt.Fprintln(f.writer, msg)
	}
}

func (f *Formatter) renderJSON(data any) error {
	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func (f *Formatter) renderTable(entities []map[string]any) error {
	if len(entities) == 0 {
		fmt.Fprintln(f.writer, "No results found.")
		return nil
	}

	// Collect all keys, prioritize common fields.
	keys := collectKeys(entities)

	table := tw.NewTable(f.writer)
	headers := make([]string, len(keys))
	for i, k := range keys {
		headers[i] = strings.ToUpper(k)
	}
	table.Header(headers)

	for _, entity := range entities {
		row := make([]string, len(keys))
		for i, k := range keys {
			row[i] = formatValue(entity[k])
		}
		_ = table.Append(row)
	}

	_ = table.Render()
	return nil
}

func (f *Formatter) renderDetailView(entity map[string]any) error {
	if len(entity) == 0 {
		fmt.Fprintln(f.writer, "No data.")
		return nil
	}

	keys := sortedKeys(entity)
	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	for _, k := range keys {
		label := k
		if f.color {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
			label = style.Render(k)
		}
		val := formatValue(entity[k])
		padding := strings.Repeat(" ", maxKeyLen-len(k))
		fmt.Fprintf(f.writer, "%s%s : %s\n", label, padding, val)
	}
	return nil
}

// collectKeys extracts unique keys from entities, prioritizing common CRM fields.
func collectKeys(entities []map[string]any) []string {
	priority := []string{"id", "name", "title", "slug", "type", "status", "stage", "created_at"}
	seen := make(map[string]bool)
	var keys []string

	// Add priority keys first if they exist.
	for _, pk := range priority {
		for _, e := range entities {
			if _, ok := e[pk]; ok && !seen[pk] {
				keys = append(keys, pk)
				seen[pk] = true
				break
			}
		}
	}

	// Add remaining keys.
	for _, e := range entities {
		for k := range e {
			if !seen[k] {
				keys = append(keys, k)
				seen[k] = true
			}
		}
	}
	return keys
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		if len(val) > 80 {
			return val[:77] + "..."
		}
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case map[string]any:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		s := string(data)
		if len(s) > 80 {
			return s[:77] + "..."
		}
		return s
	default:
		s := fmt.Sprintf("%v", val)
		if len(s) > 80 {
			return s[:77] + "..."
		}
		return s
	}
}
