package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// validExportTypes lists the accepted export types.
var validExportTypes = map[string]bool{
	"users": true,
	"orgs":  true,
	"audit": true,
}

// validExportFormats lists the accepted export formats.
var validExportFormats = map[string]bool{
	"csv":  true,
	"json": true,
}

// CreateExport creates a new async export request and starts processing.
func (s *Service) CreateExport(ctx context.Context, exportType, format, filters, requestedBy string, storage upload.StorageProvider) (*models.AdminExport, error) {
	if !validExportTypes[exportType] {
		return nil, fmt.Errorf("invalid export type: %s (must be users, orgs, or audit)", exportType)
	}
	if !validExportFormats[format] {
		return nil, fmt.Errorf("invalid export format: %s (must be csv or json)", format)
	}
	if requestedBy == "" {
		return nil, fmt.Errorf("requested_by is required")
	}

	if filters == "" {
		filters = "{}"
	}
	if !json.Valid([]byte(filters)) {
		return nil, fmt.Errorf("invalid filters JSON")
	}

	export := &models.AdminExport{
		Type:        exportType,
		Filters:     filters,
		Format:      format,
		Status:      "pending",
		RequestedBy: requestedBy,
	}
	if err := s.db.WithContext(ctx).Create(export).Error; err != nil {
		return nil, fmt.Errorf("creating export: %w", err)
	}

	// Process in background.
	go s.processExport(export.ID, storage)

	return export, nil
}

// ListExports returns all exports for the requesting admin.
func (s *Service) ListExports(ctx context.Context, requestedBy string, params pagination.Params) ([]models.AdminExport, *pagination.PageInfo, error) {
	var exports []models.AdminExport
	query := s.db.WithContext(ctx).Where("requested_by = ?", requestedBy).Order("created_at DESC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&exports).Error; err != nil {
		return nil, nil, fmt.Errorf("listing exports: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(exports) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(exports[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		exports = exports[:params.Limit]
	}

	return exports, pageInfo, nil
}

// GetExport returns a single export by ID.
func (s *Service) GetExport(ctx context.Context, id string) (*models.AdminExport, error) {
	var export models.AdminExport
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&export).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting export: %w", err)
	}
	return &export, nil
}

// processExport runs the export job in the background.
func (s *Service) processExport(exportID string, storage upload.StorageProvider) {
	ctx := context.Background()

	// Mark as processing.
	s.db.Model(&models.AdminExport{}).Where("id = ?", exportID).
		Update("status", "processing")

	var export models.AdminExport
	if err := s.db.Where("id = ?", exportID).First(&export).Error; err != nil {
		s.failExport(exportID, "failed to load export record")
		return
	}

	var data []byte
	var err error

	switch export.Type {
	case "users":
		data, err = s.exportUsers(ctx, export.Format)
	case "orgs":
		data, err = s.exportOrgs(ctx, export.Format)
	case "audit":
		data, err = s.exportAudit(ctx, export.Format)
	default:
		err = fmt.Errorf("unknown export type: %s", export.Type)
	}

	if err != nil {
		s.failExport(exportID, err.Error())
		return
	}

	// Store the file.
	filename := fmt.Sprintf("export-%s-%s.%s", export.Type, exportID[:8], export.Format)
	var filePath string
	if storage != nil {
		filePath, err = storage.Store(filename, bytes.NewReader(data))
		if err != nil {
			s.failExport(exportID, "failed to store export file: "+err.Error())
			return
		}
	} else {
		filePath = filename // Fallback: just store the name.
	}

	now := time.Now()
	s.db.Model(&models.AdminExport{}).Where("id = ?", exportID).
		Updates(map[string]any{
			"status":       "completed",
			"file_path":    filePath,
			"completed_at": &now,
		})
}

// failExport marks an export as failed with an error message.
func (s *Service) failExport(exportID, errMsg string) {
	s.db.Model(&models.AdminExport{}).Where("id = ?", exportID).
		Updates(map[string]any{
			"status":    "failed",
			"error_msg": errMsg,
		})
}

// exportUsers generates user data in the requested format.
func (s *Service) exportUsers(ctx context.Context, format string) ([]byte, error) {
	var users []models.UserShadow
	if err := s.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("querying users: %w", err)
	}

	if format == "json" {
		return json.Marshal(users)
	}

	// CSV format.
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"clerk_user_id", "email", "display_name", "is_banned", "last_seen_at"})
	for _, u := range users {
		_ = w.Write([]string{
			u.ClerkUserID, u.Email, u.DisplayName,
			fmt.Sprintf("%v", u.IsBanned),
			u.LastSeenAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	return buf.Bytes(), nil
}

// exportOrgs generates org data in the requested format.
func (s *Service) exportOrgs(ctx context.Context, format string) ([]byte, error) {
	var orgs []models.Org
	if err := s.db.WithContext(ctx).Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("querying orgs: %w", err)
	}

	if format == "json" {
		return json.Marshal(orgs)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "name", "slug", "created_at"})
	for _, o := range orgs {
		_ = w.Write([]string{o.ID, o.Name, o.Slug, o.CreatedAt.Format(time.RFC3339)})
	}
	w.Flush()
	return buf.Bytes(), nil
}

// exportAudit generates audit log data in the requested format.
func (s *Service) exportAudit(ctx context.Context, format string) ([]byte, error) {
	var logs []models.AuditLog
	if err := s.db.WithContext(ctx).Limit(10000).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("querying audit logs: %w", err)
	}

	if format == "json" {
		return json.Marshal(logs)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "user_id", "action", "entity_type", "entity_id", "created_at"})
	for _, l := range logs {
		_ = w.Write([]string{
			l.ID, l.UserID, string(l.Action),
			l.EntityType, l.EntityID,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	return buf.Bytes(), nil
}

// --- Export Handlers ---

// CreateExportHandler handles POST /v1/admin/exports.
func (h *Handler) CreateExportHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type    string          `json:"type"`
		Filters json.RawMessage `json:"filters,omitempty"`
		Format  string          `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if body.Type == "" || body.Format == "" {
		apierrors.ValidationError(w, "type and format are required", nil)
		return
	}

	uc := auth.GetUserContext(r.Context())
	requestedBy := ""
	if uc != nil {
		requestedBy = uc.UserID
	}

	filters := "{}"
	if len(body.Filters) > 0 {
		filters = string(body.Filters)
	}

	export, err := h.service.CreateExport(r.Context(), body.Type, body.Format, filters, requestedBy, h.storage)
	if err != nil {
		if isExportValidationErr(err) {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		apierrors.InternalError(w, "failed to create export")
		return
	}

	response.JSON(w, http.StatusAccepted, map[string]any{
		"export_id": export.ID,
		"status":    export.Status,
	})
}

// ListExportsHandler handles GET /v1/admin/exports.
func (h *Handler) ListExportsHandler(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	requestedBy := ""
	if uc != nil {
		requestedBy = uc.UserID
	}

	params := pagination.Parse(r)
	exports, pageInfo, err := h.service.ListExports(r.Context(), requestedBy, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list exports")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     exports,
		PageInfo: pageInfo,
	})
}

// GetExportHandler handles GET /v1/admin/exports/{id}.
func (h *Handler) GetExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		apierrors.BadRequest(w, "export id is required")
		return
	}

	export, err := h.service.GetExport(r.Context(), id)
	if err != nil {
		apierrors.InternalError(w, "failed to get export")
		return
	}
	if export == nil {
		apierrors.NotFound(w, "export not found")
		return
	}

	response.JSON(w, http.StatusOK, export)
}

// isExportValidationErr checks if an error is a validation error.
func isExportValidationErr(err error) bool {
	msg := err.Error()
	return strings.HasPrefix(msg, "invalid export") ||
		msg == "requested_by is required" ||
		strings.HasPrefix(msg, "invalid filters")
}
