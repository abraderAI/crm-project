// Package health provides health check endpoint handlers.
package health

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"
)

// HealthResponse is the response payload for health check endpoints.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Handler holds dependencies for health check endpoints.
type Handler struct {
	DB *gorm.DB
}

// NewHandler creates a new health handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// Healthz returns a simple liveness check (always 200 if the server is up).
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// Readyz returns a readiness check including database connectivity.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)

	// Check SQLite connectivity.
	sqlDB, err := h.DB.DB()
	if err != nil {
		checks["database"] = "error: " + err.Error()
		writeJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status: "unavailable",
			Checks: checks,
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		checks["database"] = "error: " + err.Error()
		writeJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status: "unavailable",
			Checks: checks,
		})
		return
	}

	checks["database"] = "ok"
	writeJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
		Checks: checks,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
