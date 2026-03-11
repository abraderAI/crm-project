// Package search provides full-text search across all entities using FTS5.
package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// SearchResult represents a single search result with snippet and rank.
type SearchResult struct {
	EntityType string  `json:"entity_type"`
	EntityID   string  `json:"entity_id"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
	Rank       float64 `json:"rank"`
}

// Repository handles FTS5 search queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new search repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ftsTarget defines a FTS5 table and how to extract results.
type ftsTarget struct {
	ftsTable   string
	srcTable   string
	entityType string
	titleCol   string // column to use as title in results
	snippetCol int    // column index for snippet() function
}

var allTargets = []ftsTarget{
	{"orgs_fts", "orgs", "org", "name", 0},
	{"spaces_fts", "spaces", "space", "name", 0},
	{"boards_fts", "boards", "board", "name", 0},
	{"threads_fts", "threads", "thread", "title", 0},
	{"messages_fts", "messages", "message", "body", 0},
}

// Search performs FTS5 search across all (or filtered) entity types.
func (r *Repository) Search(ctx context.Context, query string, entityTypes []string, params pagination.Params) ([]SearchResult, *pagination.PageInfo, error) {
	if query == "" {
		return nil, &pagination.PageInfo{}, nil
	}

	// Sanitize query for FTS5.
	sanitized := sanitizeFTSQuery(query)
	if sanitized == "" {
		return nil, &pagination.PageInfo{}, nil
	}

	targets := filterTargets(entityTypes)
	if len(targets) == 0 {
		return nil, &pagination.PageInfo{}, nil
	}

	var allResults []SearchResult
	for _, tgt := range targets {
		results, err := r.searchTable(ctx, tgt, sanitized)
		if err != nil {
			return nil, nil, fmt.Errorf("searching %s: %w", tgt.entityType, err)
		}
		allResults = append(allResults, results...)
	}

	// Sort by rank (lower = more relevant in FTS5).
	sortByRank(allResults)

	// Apply cursor pagination.
	pageInfo := &pagination.PageInfo{}
	startIdx := 0
	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err == nil {
			// Find the position after the cursor.
			cursorStr := cursorID.String()
			for i, r := range allResults {
				if r.EntityID == cursorStr {
					startIdx = i + 1
					break
				}
			}
		}
	}

	if startIdx >= len(allResults) {
		return nil, pageInfo, nil
	}

	endIdx := startIdx + params.Limit
	if endIdx > len(allResults) {
		endIdx = len(allResults)
	} else if endIdx < len(allResults) {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(allResults[endIdx-1].EntityID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
	}

	return allResults[startIdx:endIdx], pageInfo, nil
}

// searchTable queries a single FTS5 table.
func (r *Repository) searchTable(ctx context.Context, tgt ftsTarget, query string) ([]SearchResult, error) {
	sql := fmt.Sprintf(
		`SELECT s.id, snippet(%s, %d, '<mark>', '</mark>', '...', 32) as snip, rank
		 FROM %s f
		 JOIN %s s ON s.rowid = f.rowid
		 WHERE %s MATCH ? AND s.deleted_at IS NULL
		 ORDER BY rank
		 LIMIT 100`,
		tgt.ftsTable, tgt.snippetCol,
		tgt.ftsTable, tgt.srcTable,
		tgt.ftsTable,
	)

	rows, err := r.db.WithContext(ctx).Raw(sql, query).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var id, snip string
		var rank float64
		if err := rows.Scan(&id, &snip, &rank); err != nil {
			return nil, err
		}
		results = append(results, SearchResult{
			EntityType: tgt.entityType,
			EntityID:   id,
			Title:      snip,
			Snippet:    snip,
			Rank:       rank,
		})
	}
	return results, rows.Err()
}

// filterTargets filters FTS targets by entity type if specified.
func filterTargets(entityTypes []string) []ftsTarget {
	if len(entityTypes) == 0 {
		return allTargets
	}
	typeSet := make(map[string]bool, len(entityTypes))
	for _, t := range entityTypes {
		typeSet[strings.ToLower(t)] = true
	}
	var filtered []ftsTarget
	for _, tgt := range allTargets {
		if typeSet[tgt.entityType] {
			filtered = append(filtered, tgt)
		}
	}
	return filtered
}

// sanitizeFTSQuery escapes special characters for FTS5 queries.
func sanitizeFTSQuery(query string) string {
	// Remove FTS5 operators and special chars to prevent injection.
	replacer := strings.NewReplacer(
		"\"", "",
		"'", "",
		"*", "",
		"(", "",
		")", "",
		":", "",
		"^", "",
		"+", "",
		"-", " ",
		"AND", "",
		"OR", "",
		"NOT", "",
		"NEAR", "",
	)
	cleaned := replacer.Replace(query)
	// Split into words and rejoin.
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}
	// Quote each word for safety.
	var quoted []string
	for _, w := range words {
		if w != "" {
			quoted = append(quoted, "\""+w+"\"")
		}
	}
	return strings.Join(quoted, " ")
}

// sortByRank sorts results by FTS5 rank (ascending = more relevant).
func sortByRank(results []SearchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Rank < results[j-1].Rank; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}
