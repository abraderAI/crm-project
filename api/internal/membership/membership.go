// Package membership provides CRUD operations for org/space/board memberships.
package membership

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// Service errors.
var (
	ErrNotFound      = errors.New("membership not found")
	ErrInvalidRole   = errors.New("invalid role")
	ErrUserRequired  = errors.New("user_id is required")
	ErrLastOwner     = errors.New("cannot remove the last owner")
	ErrAlreadyExists = errors.New("membership already exists")
)

// --- Repository ---

// Repository handles database operations for memberships.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new membership repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// CreateOrgMembership adds a user to an org.
func (r *Repository) CreateOrgMembership(ctx context.Context, orgID, userID string, role models.Role) error {
	m := &models.OrgMembership{OrgID: orgID, UserID: userID, Role: role}
	return r.db.WithContext(ctx).Create(m).Error
}

// ListOrgMembers lists all members of an org.
func (r *Repository) ListOrgMembers(ctx context.Context, orgID string) ([]models.OrgMembership, error) {
	var members []models.OrgMembership
	return members, r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&members).Error
}

// GetOrgMembership retrieves a specific org membership.
func (r *Repository) GetOrgMembership(ctx context.Context, id string) (*models.OrgMembership, error) {
	var m models.OrgMembership
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateOrgMembership updates the role of an org membership.
func (r *Repository) UpdateOrgMembership(ctx context.Context, m *models.OrgMembership) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// DeleteOrgMembership soft-deletes an org membership.
func (r *Repository) DeleteOrgMembership(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.OrgMembership{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// CountOrgOwners counts the number of owners in an org.
func (r *Repository) CountOrgOwners(ctx context.Context, orgID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OrgMembership{}).
		Where("org_id = ? AND role = ?", orgID, models.RoleOwner).Count(&count).Error
	return count, err
}

// OrgMembershipExists checks if a membership already exists.
func (r *Repository) OrgMembershipExists(ctx context.Context, orgID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OrgMembership{}).
		Where("org_id = ? AND user_id = ?", orgID, userID).Count(&count).Error
	return count > 0, err
}

// CreateSpaceMembership adds a user to a space.
func (r *Repository) CreateSpaceMembership(ctx context.Context, spaceID, userID string, role models.Role) error {
	m := &models.SpaceMembership{SpaceID: spaceID, UserID: userID, Role: role}
	return r.db.WithContext(ctx).Create(m).Error
}

// ListSpaceMembers lists all members of a space.
func (r *Repository) ListSpaceMembers(ctx context.Context, spaceID string) ([]models.SpaceMembership, error) {
	var members []models.SpaceMembership
	return members, r.db.WithContext(ctx).Where("space_id = ?", spaceID).Find(&members).Error
}

// GetSpaceMembership retrieves a specific space membership.
func (r *Repository) GetSpaceMembership(ctx context.Context, id string) (*models.SpaceMembership, error) {
	var m models.SpaceMembership
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateSpaceMembership updates the role of a space membership.
func (r *Repository) UpdateSpaceMembership(ctx context.Context, m *models.SpaceMembership) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// DeleteSpaceMembership soft-deletes a space membership.
func (r *Repository) DeleteSpaceMembership(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.SpaceMembership{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// CreateBoardMembership adds a user to a board.
func (r *Repository) CreateBoardMembership(ctx context.Context, boardID, userID string, role models.Role) error {
	m := &models.BoardMembership{BoardID: boardID, UserID: userID, Role: role}
	return r.db.WithContext(ctx).Create(m).Error
}

// ListBoardMembers lists all members of a board.
func (r *Repository) ListBoardMembers(ctx context.Context, boardID string) ([]models.BoardMembership, error) {
	var members []models.BoardMembership
	return members, r.db.WithContext(ctx).Where("board_id = ?", boardID).Find(&members).Error
}

// GetBoardMembership retrieves a specific board membership.
func (r *Repository) GetBoardMembership(ctx context.Context, id string) (*models.BoardMembership, error) {
	var m models.BoardMembership
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateBoardMembership updates the role of a board membership.
func (r *Repository) UpdateBoardMembership(ctx context.Context, m *models.BoardMembership) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// DeleteBoardMembership soft-deletes a board membership.
func (r *Repository) DeleteBoardMembership(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.BoardMembership{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// --- Service ---

// MemberInput holds parameters for adding/updating a membership.
type MemberInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// Service provides business logic for membership operations.
type Service struct {
	repo *Repository
}

// NewService creates a new membership service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// AddOrgMember adds a user to an org.
func (s *Service) AddOrgMember(ctx context.Context, orgID string, input MemberInput) error {
	if input.UserID == "" {
		return ErrUserRequired
	}
	role := models.Role(input.Role)
	if input.Role == "" {
		role = models.RoleViewer
	} else if !role.IsValid() {
		return ErrInvalidRole
	}
	exists, err := s.repo.OrgMembershipExists(ctx, orgID, input.UserID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyExists
	}
	return s.repo.CreateOrgMembership(ctx, orgID, input.UserID, role)
}

// ListOrgMembers lists all members of an org.
func (s *Service) ListOrgMembers(ctx context.Context, orgID string) ([]models.OrgMembership, error) {
	return s.repo.ListOrgMembers(ctx, orgID)
}

// UpdateOrgMember updates a member's role.
func (s *Service) UpdateOrgMember(ctx context.Context, id string, input MemberInput) error {
	m, err := s.repo.GetOrgMembership(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	role := models.Role(input.Role)
	if !role.IsValid() {
		return ErrInvalidRole
	}
	// Check if downgrading the last owner.
	if m.Role == models.RoleOwner && role != models.RoleOwner {
		count, err := s.repo.CountOrgOwners(ctx, m.OrgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}
	m.Role = role
	return s.repo.UpdateOrgMembership(ctx, m)
}

// RemoveOrgMember removes a member from an org.
func (s *Service) RemoveOrgMember(ctx context.Context, id string) error {
	m, err := s.repo.GetOrgMembership(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if m.Role == models.RoleOwner {
		count, err := s.repo.CountOrgOwners(ctx, m.OrgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}
	return s.repo.DeleteOrgMembership(ctx, id)
}

// AddSpaceMember adds a user to a space.
func (s *Service) AddSpaceMember(ctx context.Context, spaceID string, input MemberInput) error {
	if input.UserID == "" {
		return ErrUserRequired
	}
	role := models.Role(input.Role)
	if input.Role == "" {
		role = models.RoleViewer
	} else if !role.IsValid() {
		return ErrInvalidRole
	}
	return s.repo.CreateSpaceMembership(ctx, spaceID, input.UserID, role)
}

// ListSpaceMembers lists space members.
func (s *Service) ListSpaceMembers(ctx context.Context, spaceID string) ([]models.SpaceMembership, error) {
	return s.repo.ListSpaceMembers(ctx, spaceID)
}

// UpdateSpaceMember updates a space member's role.
func (s *Service) UpdateSpaceMember(ctx context.Context, id string, input MemberInput) error {
	m, err := s.repo.GetSpaceMembership(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	role := models.Role(input.Role)
	if !role.IsValid() {
		return ErrInvalidRole
	}
	m.Role = role
	return s.repo.UpdateSpaceMembership(ctx, m)
}

// RemoveSpaceMember removes a space member.
func (s *Service) RemoveSpaceMember(ctx context.Context, id string) error {
	return s.repo.DeleteSpaceMembership(ctx, id)
}

// AddBoardMember adds a user to a board.
func (s *Service) AddBoardMember(ctx context.Context, boardID string, input MemberInput) error {
	if input.UserID == "" {
		return ErrUserRequired
	}
	role := models.Role(input.Role)
	if input.Role == "" {
		role = models.RoleViewer
	} else if !role.IsValid() {
		return ErrInvalidRole
	}
	return s.repo.CreateBoardMembership(ctx, boardID, input.UserID, role)
}

// ListBoardMembers lists board members.
func (s *Service) ListBoardMembers(ctx context.Context, boardID string) ([]models.BoardMembership, error) {
	return s.repo.ListBoardMembers(ctx, boardID)
}

// UpdateBoardMember updates a board member's role.
func (s *Service) UpdateBoardMember(ctx context.Context, id string, input MemberInput) error {
	m, err := s.repo.GetBoardMembership(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	role := models.Role(input.Role)
	if !role.IsValid() {
		return ErrInvalidRole
	}
	m.Role = role
	return s.repo.UpdateBoardMembership(ctx, m)
}

// RemoveBoardMember removes a board member.
func (s *Service) RemoveBoardMember(ctx context.Context, id string) error {
	return s.repo.DeleteBoardMembership(ctx, id)
}

// --- Handler ---

// Handler provides HTTP handlers for membership endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new membership handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// AddOrgMember handles POST /v1/orgs/{org}/members.
func (h *Handler) AddOrgMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.AddOrgMember(r.Context(), orgID, input); err != nil {
		writeMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// ListOrgMembers handles GET /v1/orgs/{org}/members.
func (h *Handler) ListOrgMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	members, err := h.service.ListOrgMembers(r.Context(), orgID)
	if err != nil {
		apierrors.InternalError(w, "failed to list members")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": members})
}

// UpdateOrgMember handles PATCH /v1/orgs/{org}/members/{id}.
func (h *Handler) UpdateOrgMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.UpdateOrgMember(r.Context(), id, input); err != nil {
		writeMemberError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// RemoveOrgMember handles DELETE /v1/orgs/{org}/members/{id}.
func (h *Handler) RemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.RemoveOrgMember(r.Context(), id); err != nil {
		writeMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddSpaceMember handles POST .../spaces/{space}/members.
func (h *Handler) AddSpaceMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.AddSpaceMember(r.Context(), spaceID, input); err != nil {
		writeMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// ListSpaceMembers handles GET .../spaces/{space}/members.
func (h *Handler) ListSpaceMembers(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	members, err := h.service.ListSpaceMembers(r.Context(), spaceID)
	if err != nil {
		apierrors.InternalError(w, "failed to list members")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": members})
}

// UpdateSpaceMember handles PATCH .../spaces/{space}/members/{id}.
func (h *Handler) UpdateSpaceMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.UpdateSpaceMember(r.Context(), id, input); err != nil {
		writeMemberError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// RemoveSpaceMember handles DELETE .../spaces/{space}/members/{id}.
func (h *Handler) RemoveSpaceMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.RemoveSpaceMember(r.Context(), id); err != nil {
		writeMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddBoardMember handles POST .../boards/{board}/members.
func (h *Handler) AddBoardMember(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.AddBoardMember(r.Context(), boardID, input); err != nil {
		writeMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// ListBoardMembers handles GET .../boards/{board}/members.
func (h *Handler) ListBoardMembers(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	members, err := h.service.ListBoardMembers(r.Context(), boardID)
	if err != nil {
		apierrors.InternalError(w, "failed to list members")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": members})
}

// UpdateBoardMember handles PATCH .../boards/{board}/members/{id}.
func (h *Handler) UpdateBoardMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input MemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.UpdateBoardMember(r.Context(), id, input); err != nil {
		writeMemberError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// RemoveBoardMember handles DELETE .../boards/{board}/members/{id}.
func (h *Handler) RemoveBoardMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.RemoveBoardMember(r.Context(), id); err != nil {
		writeMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeMemberError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierrors.NotFound(w, err.Error())
	case errors.Is(err, ErrInvalidRole):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "role", Message: err.Error()},
		})
	case errors.Is(err, ErrUserRequired):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "user_id", Message: err.Error()},
		})
	case errors.Is(err, ErrLastOwner):
		apierrors.Conflict(w, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		apierrors.Conflict(w, err.Error())
	default:
		apierrors.InternalError(w, "internal server error")
	}
}
