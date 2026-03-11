package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteProblem(t *testing.T) {
	w := httptest.NewRecorder()
	problem := ProblemDetail{
		Type:   "https://httpstatuses.com/400",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: "test detail",
	}

	WriteProblem(w, problem)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var got ProblemDetail
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, problem.Type, got.Type)
	assert.Equal(t, problem.Title, got.Title)
	assert.Equal(t, problem.Status, got.Status)
	assert.Equal(t, problem.Detail, got.Detail)
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "resource missing")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Not Found", got.Title)
	assert.Equal(t, 404, got.Status)
	assert.Equal(t, "resource missing", got.Detail)
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "invalid token")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Unauthorized", got.Title)
	assert.Equal(t, 401, got.Status)
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "access denied")

	assert.Equal(t, http.StatusForbidden, w.Code)
	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Forbidden", got.Title)
	assert.Equal(t, 403, got.Status)
}

func TestValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	fieldErrors := []FieldError{
		{Field: "name", Message: "must not be empty"},
		{Field: "email", Message: "invalid format"},
	}
	ValidationError(w, "validation failed", fieldErrors)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Bad Request", got.Title)
	assert.Len(t, got.Errors, 2)
	assert.Equal(t, "name", got.Errors[0].Field)
	assert.Equal(t, "must not be empty", got.Errors[0].Message)
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "malformed input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Bad Request", got.Title)
	assert.Equal(t, "malformed input", got.Detail)
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	Conflict(w, "slug already exists")

	assert.Equal(t, http.StatusConflict, w.Code)
	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Conflict", got.Title)
	assert.Equal(t, 409, got.Status)
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w, "something went wrong")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Internal Server Error", got.Title)
	assert.Equal(t, 500, got.Status)
}

func TestProblemDetail_OmitEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	WriteProblem(w, ProblemDetail{
		Type:   "https://httpstatuses.com/404",
		Title:  "Not Found",
		Status: 404,
	})

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	_, hasDetail := raw["detail"]
	assert.False(t, hasDetail, "detail should be omitted when empty")
	_, hasInstance := raw["instance"]
	assert.False(t, hasInstance, "instance should be omitted when empty")
	_, hasErrors := raw["errors"]
	assert.False(t, hasErrors, "errors should be omitted when nil")
}

func TestValidationError_EmptyFieldErrors(t *testing.T) {
	w := httptest.NewRecorder()
	ValidationError(w, "no fields", nil)

	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Nil(t, got.Errors)
}

func TestWriteProblem_WithInstance(t *testing.T) {
	w := httptest.NewRecorder()
	WriteProblem(w, ProblemDetail{
		Type:     "https://httpstatuses.com/500",
		Title:    "Internal Server Error",
		Status:   500,
		Detail:   "test",
		Instance: "/v1/orgs/123",
	})

	var got ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "/v1/orgs/123", got.Instance)
}
