package tier

import (
	"encoding/json"
	"net/http"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for tier and home-preferences endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new tier Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetTier handles GET /api/me/tier.
// Returns the authenticated user's resolved tier information.
func (h *Handler) GetTier(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	userID := ""
	if uc != nil {
		userID = uc.UserID
	}

	result, err := h.service.ResolveTier(userID)
	if err != nil {
		apierrors.InternalError(w, "failed to resolve tier")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// homePreferencesRequest is the expected body for PUT /api/me/home-preferences.
type homePreferencesRequest struct {
	Tier   int                       `json:"tier"`
	Layout []models.WidgetPreference `json:"layout"`
}

// GetHomePreferences handles GET /api/me/home-preferences.
// Returns the user's saved home layout preferences or null.
func (h *Handler) GetHomePreferences(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	prefs, err := h.service.GetHomePreferences(uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to retrieve home preferences")
		return
	}

	if prefs == nil {
		response.JSON(w, http.StatusOK, nil)
		return
	}

	response.JSON(w, http.StatusOK, prefs)
}

// PutHomePreferences handles PUT /api/me/home-preferences.
// Validates and stores the user's home layout preferences.
func (h *Handler) PutHomePreferences(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var req homePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	// Validate layout has at least one widget.
	if len(req.Layout) == 0 {
		apierrors.ValidationError(w, "layout must contain at least one widget", []apierrors.FieldError{
			{Field: "layout", Message: "must contain at least one widget"},
		})
		return
	}

	// Validate all widget IDs are non-empty.
	seen := make(map[string]bool)
	for _, wp := range req.Layout {
		if wp.WidgetID == "" {
			apierrors.ValidationError(w, "widget_id must not be empty", []apierrors.FieldError{
				{Field: "layout.widget_id", Message: "must not be empty"},
			})
			return
		}
		if seen[wp.WidgetID] {
			apierrors.ValidationError(w, "duplicate widget_id: "+wp.WidgetID, []apierrors.FieldError{
				{Field: "layout.widget_id", Message: "duplicate widget_id: " + wp.WidgetID},
			})
			return
		}
		seen[wp.WidgetID] = true
	}

	// Validate tier.
	if !Tier(req.Tier).IsValid() {
		apierrors.ValidationError(w, "invalid tier value", []apierrors.FieldError{
			{Field: "tier", Message: "must be between 1 and 6"},
		})
		return
	}

	// Serialize layout to JSON.
	layoutJSON, err := json.Marshal(req.Layout)
	if err != nil {
		apierrors.InternalError(w, "failed to encode layout")
		return
	}

	prefs := &models.UserHomePreferences{
		UserID: uc.UserID,
		Tier:   req.Tier,
		Layout: string(layoutJSON),
	}

	if err := h.service.SaveHomePreferences(prefs); err != nil {
		apierrors.InternalError(w, "failed to save home preferences")
		return
	}

	response.JSON(w, http.StatusOK, prefs)
}
