// Package pagination provides cursor-based pagination helpers using UUIDv7 encoding.
package pagination

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

const (
	// DefaultLimit is the default number of items per page.
	DefaultLimit = 50
	// MaxLimit is the maximum allowed number of items per page.
	MaxLimit = 100

	cursorPrefix = "cursor:"
)

// Params holds parsed pagination parameters from a request.
type Params struct {
	Cursor string
	Limit  int
}

// PageInfo holds pagination metadata for a response.
type PageInfo struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// Parse extracts cursor and limit from query parameters.
// Defaults to DefaultLimit if limit is not provided or invalid.
// Clamps limit to [1, MaxLimit].
func Parse(r *http.Request) Params {
	q := r.URL.Query()

	limit := DefaultLimit
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	if limit < 1 {
		limit = 1
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return Params{
		Cursor: q.Get("cursor"),
		Limit:  limit,
	}
}

// EncodeCursor encodes a UUIDv7 into a base64 cursor string.
func EncodeCursor(id uuid.UUID) string {
	raw := cursorPrefix + id.String()
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a base64 cursor string back to a UUIDv7.
// Returns uuid.Nil and an error if the cursor is invalid.
func DecodeCursor(cursor string) (uuid.UUID, error) {
	if cursor == "" {
		return uuid.Nil, nil
	}

	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	str := string(raw)
	if len(str) < len(cursorPrefix) || str[:len(cursorPrefix)] != cursorPrefix {
		return uuid.Nil, fmt.Errorf("invalid cursor format: missing prefix")
	}

	id, err := uuid.Parse(str[len(cursorPrefix):])
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid cursor UUID: %w", err)
	}

	return id, nil
}
