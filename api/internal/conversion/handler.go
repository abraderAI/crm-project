package conversion

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for conversion endpoints.
type Handler struct {
	service      *Service
	auditService *audit.Service
}

// NewHandler creates a new conversion Handler.
func NewHandler(service *Service, auditService *audit.Service) *Handler {
	return &Handler{service: service, auditService: auditService}
}

// selfServiceRequest is the request body for POST /v1/me/upgrade.
type selfServiceRequest struct {
	OrgName string `json:"org_name"`
}

// SelfServiceUpgrade handles POST /v1/me/upgrade.
// Creates an org for the authenticated user and promotes them to Tier 3.
func (h *Handler) SelfServiceUpgrade(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var req selfServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if req.OrgName == "" {
		apierrors.ValidationError(w, "org_name is required", []apierrors.FieldError{
			{Field: "org_name", Message: "must not be empty"},
		})
		return
	}

	result, err := h.service.SelfServiceUpgrade(r.Context(), uc.UserID, req.OrgName)
	if err != nil {
		switch err.Error() {
		case "user not found":
			apierrors.NotFound(w, "user not found")
		case "user already belongs to an organization":
			apierrors.BadRequest(w, "user already belongs to an organization")
		default:
			apierrors.InternalError(w, "failed to upgrade")
		}
		return
	}

	audit.CreateAuditEntry(r.Context(), h.auditService, "upgrade", "user", uc.UserID, nil,
		map[string]string{"org_id": result.Org.ID, "tier": "3"})

	response.JSON(w, http.StatusOK, result)
}

// salesConvertRequest is the request body for POST /v1/admin/leads/{lead_id}/convert.
type salesConvertRequest struct {
	OrgName string `json:"org_name"`
}

// SalesConvert handles POST /v1/admin/leads/{lead_id}/convert.
// Allows a DEFT sales member to convert a lead to Tier 3.
func (h *Handler) SalesConvert(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	// Verify the actor is a DEFT org member.
	isDeft, err := h.service.IsDeftOrgMember(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to verify DEFT membership")
		return
	}
	if !isDeft {
		apierrors.Forbidden(w, "only DEFT org members can convert leads")
		return
	}

	leadID := chi.URLParam(r, "lead_id")
	if leadID == "" {
		apierrors.BadRequest(w, "lead_id is required")
		return
	}

	var req salesConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if req.OrgName == "" {
		apierrors.ValidationError(w, "org_name is required", []apierrors.FieldError{
			{Field: "org_name", Message: "must not be empty"},
		})
		return
	}

	result, err := h.service.SalesConvert(r.Context(), leadID, req.OrgName, uc.UserID)
	if err != nil {
		switch err.Error() {
		case "lead not found":
			apierrors.NotFound(w, "lead not found")
		case "lead is already converted":
			apierrors.BadRequest(w, "lead is already converted")
		default:
			apierrors.InternalError(w, "failed to convert lead")
		}
		return
	}

	audit.CreateAuditEntry(r.Context(), h.auditService, "convert_lead", "lead", leadID, nil,
		map[string]string{"org_id": result.Org.ID, "actor": uc.UserID})

	response.JSON(w, http.StatusOK, result)
}

// adminPromoteRequest is the request body for POST /v1/admin/users/{user_id}/promote.
type adminPromoteRequest struct {
	OrgName string `json:"org_name"`
}

// AdminPromote handles POST /v1/admin/users/{user_id}/promote.
// Allows a platform admin to promote a user to Tier 3 by assigning them to an org.
func (h *Handler) AdminPromote(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	var req adminPromoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if req.OrgName == "" {
		apierrors.ValidationError(w, "org_name is required", []apierrors.FieldError{
			{Field: "org_name", Message: "must not be empty"},
		})
		return
	}

	result, err := h.service.AdminPromote(r.Context(), userID, req.OrgName)
	if err != nil {
		switch err.Error() {
		case "user not found":
			apierrors.NotFound(w, "user not found")
		default:
			apierrors.InternalError(w, "failed to promote user")
		}
		return
	}

	audit.CreateAuditEntry(r.Context(), h.auditService, "promote_user", "user", userID, nil,
		map[string]string{"org_id": result.Org.ID, "admin": uc.UserID, "tier": "3"})

	response.JSON(w, http.StatusOK, result)
}
