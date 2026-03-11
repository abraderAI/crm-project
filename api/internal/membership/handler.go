package membership

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// MemberInput is the request body for adding/updating a member.
type MemberInput struct {
	UserID string      `json:"user_id"`
	Role   models.Role `json:"role"`
}

// Handler provides HTTP handlers for Membership operations.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new Membership handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// --- Org Membership Handlers ---

// AddOrgMember handles POST /v1/orgs/{org}/members.
func (h *Handler) AddOrgMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if input.UserID == "" {
		apierrors.ValidationError(w, "user_id is required", nil)
		return
	}
	if input.Role == "" {
		input.Role = models.RoleViewer
	}
	if !input.Role.IsValid() {
		apierrors.ValidationError(w, "invalid role", nil)
		return
	}

	m := &models.OrgMembership{OrgID: orgID, UserID: input.UserID, Role: input.Role}
	if err := h.repo.AddOrgMember(r.Context(), m); err != nil {
		apierrors.Conflict(w, "member already exists or invalid data")
		return
	}
	response.Created(w, m)
}

// ListOrgMembers handles GET /v1/orgs/{org}/members.
func (h *Handler) ListOrgMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	members, err := h.repo.ListOrgMembers(r.Context(), orgID)
	if err != nil {
		apierrors.InternalError(w, "failed to list members")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": members})
}

// UpdateOrgMember handles PATCH /v1/orgs/{org}/members/{userID}.
func (h *Handler) UpdateOrgMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	userID := chi.URLParam(r, "userID")

	var input struct {
		Role models.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if !input.Role.IsValid() {
		apierrors.ValidationError(w, "invalid role", nil)
		return
	}

	m, err := h.repo.GetOrgMember(r.Context(), orgID, userID)
	if err != nil {
		apierrors.InternalError(w, "failed to get member")
		return
	}
	if m == nil {
		apierrors.NotFound(w, "member not found")
		return
	}
	m.Role = input.Role
	if err := h.repo.UpdateOrgMember(r.Context(), m); err != nil {
		apierrors.InternalError(w, "failed to update member")
		return
	}
	response.JSON(w, http.StatusOK, m)
}

// RemoveOrgMember handles DELETE /v1/orgs/{org}/members/{userID}.
func (h *Handler) RemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	userID := chi.URLParam(r, "userID")

	// Check if this is the last owner.
	m, err := h.repo.GetOrgMember(r.Context(), orgID, userID)
	if err != nil {
		apierrors.InternalError(w, "failed to check member")
		return
	}
	if m == nil {
		apierrors.NotFound(w, "member not found")
		return
	}
	if m.Role == models.RoleOwner {
		count, err := h.repo.CountOrgOwners(r.Context(), orgID)
		if err != nil {
			apierrors.InternalError(w, "failed to count owners")
			return
		}
		if count <= 1 {
			apierrors.ValidationError(w, "cannot remove the last owner", nil)
			return
		}
	}

	if err := h.repo.RemoveOrgMember(r.Context(), orgID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apierrors.NotFound(w, "member not found")
			return
		}
		apierrors.InternalError(w, "failed to remove member")
		return
	}
	response.NoContent(w)
}

// --- Space Membership Handlers ---

// AddSpaceMember handles POST /v1/orgs/{org}/spaces/{space}/members.
func (h *Handler) AddSpaceMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if input.UserID == "" {
		apierrors.ValidationError(w, "user_id is required", nil)
		return
	}
	if input.Role == "" {
		input.Role = models.RoleViewer
	}
	if !input.Role.IsValid() {
		apierrors.ValidationError(w, "invalid role", nil)
		return
	}

	m := &models.SpaceMembership{SpaceID: spaceID, UserID: input.UserID, Role: input.Role}
	if err := h.repo.AddSpaceMember(r.Context(), m); err != nil {
		apierrors.Conflict(w, "member already exists or invalid data")
		return
	}
	response.Created(w, m)
}

// ListSpaceMembers handles GET /v1/orgs/{org}/spaces/{space}/members.
func (h *Handler) ListSpaceMembers(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	members, err := h.repo.ListSpaceMembers(r.Context(), spaceID)
	if err != nil {
		apierrors.InternalError(w, "failed to list members")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": members})
}

// RemoveSpaceMember handles DELETE /v1/orgs/{org}/spaces/{space}/members/{userID}.
func (h *Handler) RemoveSpaceMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	userID := chi.URLParam(r, "userID")

	if err := h.repo.RemoveSpaceMember(r.Context(), spaceID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apierrors.NotFound(w, "member not found")
			return
		}
		apierrors.InternalError(w, "failed to remove member")
		return
	}
	response.NoContent(w)
}

// --- Board Membership Handlers ---

// AddBoardMember handles POST .../boards/{board}/members.
func (h *Handler) AddBoardMember(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if input.UserID == "" {
		apierrors.ValidationError(w, "user_id is required", nil)
		return
	}
	if input.Role == "" {
		input.Role = models.RoleViewer
	}
	if !input.Role.IsValid() {
		apierrors.ValidationError(w, "invalid role", nil)
		return
	}

	m := &models.BoardMembership{BoardID: boardID, UserID: input.UserID, Role: input.Role}
	if err := h.repo.AddBoardMember(r.Context(), m); err != nil {
		apierrors.Conflict(w, "member already exists or invalid data")
		return
	}
	response.Created(w, m)
}

// ListBoardMembers handles GET .../boards/{board}/members.
func (h *Handler) ListBoardMembers(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	members, err := h.repo.ListBoardMembers(r.Context(), boardID)
	if err != nil {
		apierrors.InternalError(w, "failed to list members")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": members})
}

// RemoveBoardMember handles DELETE .../boards/{board}/members/{userID}.
func (h *Handler) RemoveBoardMember(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	userID := chi.URLParam(r, "userID")

	if err := h.repo.RemoveBoardMember(r.Context(), boardID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apierrors.NotFound(w, "member not found")
			return
		}
		apierrors.InternalError(w, "failed to remove member")
		return
	}
	response.NoContent(w)
}
