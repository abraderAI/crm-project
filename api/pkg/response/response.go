// Package response provides shared JSON response helpers for HTTP handlers.
package response

import (
	"encoding/json"
	"net/http"

	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// ListResponse is a standard paginated list response.
type ListResponse struct {
	Data     any                  `json:"data"`
	PageInfo *pagination.PageInfo `json:"page_info"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Created writes a 201 JSON response.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
