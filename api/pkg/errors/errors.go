// Package errors provides RFC 7807 Problem Details error response helpers.
package errors

import (
	"encoding/json"
	"net/http"
)

// ProblemDetail represents an RFC 7807 Problem Details response.
type ProblemDetail struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteProblem writes an RFC 7807 Problem Details JSON response.
func WriteProblem(w http.ResponseWriter, problem ProblemDetail) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

// NotFound writes a 404 Not Found problem response.
func NotFound(w http.ResponseWriter, detail string) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/404",
		Title:  "Not Found",
		Status: http.StatusNotFound,
		Detail: detail,
	})
}

// Unauthorized writes a 401 Unauthorized problem response.
func Unauthorized(w http.ResponseWriter, detail string) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/401",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: detail,
	})
}

// Forbidden writes a 403 Forbidden problem response.
func Forbidden(w http.ResponseWriter, detail string) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/403",
		Title:  "Forbidden",
		Status: http.StatusForbidden,
		Detail: detail,
	})
}

// ValidationError writes a 400 Bad Request problem response with field errors.
func ValidationError(w http.ResponseWriter, detail string, fieldErrors []FieldError) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/400",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: detail,
		Errors: fieldErrors,
	})
}

// BadRequest writes a 400 Bad Request problem response.
func BadRequest(w http.ResponseWriter, detail string) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/400",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: detail,
	})
}

// Conflict writes a 409 Conflict problem response.
func Conflict(w http.ResponseWriter, detail string) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/409",
		Title:  "Conflict",
		Status: http.StatusConflict,
		Detail: detail,
	})
}

// InternalError writes a 500 Internal Server Error problem response.
func InternalError(w http.ResponseWriter, detail string) {
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/500",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: detail,
	})
}
