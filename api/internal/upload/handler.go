package upload

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for upload operations.
type Handler struct {
	service *Service
	maxSize int64
}

// NewHandler creates a new upload handler.
func NewHandler(service *Service, maxSize int64) *Handler {
	return &Handler{service: service, maxSize: maxSize}
}

// Create handles POST /v1/uploads (multipart form upload).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Limit request body size.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxSize+1024) // extra for form overhead

	if err := r.ParseMultipartForm(h.maxSize); err != nil {
		apierrors.BadRequest(w, "failed to parse multipart form or file too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apierrors.BadRequest(w, "file field is required")
		return
	}
	defer func() { _ = file.Close() }()

	orgID := r.FormValue("org_id")
	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")

	uc := auth.GetUserContext(r.Context())
	uploaderID := ""
	if uc != nil {
		uploaderID = uc.UserID
	}

	upload, err := h.service.Create(
		r.Context(),
		orgID, entityType, entityID, uploaderID,
		header.Filename, header.Size,
		file,
	)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, upload)
}

// Get handles GET /v1/uploads/{id} — returns upload metadata.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	upload, err := h.service.Get(r.Context(), id)
	if err != nil {
		apierrors.InternalError(w, "failed to get upload")
		return
	}
	if upload == nil {
		apierrors.NotFound(w, "upload not found")
		return
	}
	response.JSON(w, http.StatusOK, upload)
}

// Download handles GET /v1/uploads/{id}/download — returns file content.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	upload, err := h.service.Get(r.Context(), id)
	if err != nil {
		apierrors.InternalError(w, "failed to get upload")
		return
	}
	if upload == nil {
		apierrors.NotFound(w, "upload not found")
		return
	}

	reader, err := h.service.GetFile(upload.StoragePath)
	if err != nil {
		apierrors.InternalError(w, "failed to read file")
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", upload.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+upload.Filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// Delete handles DELETE /v1/uploads/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "upload not found")
			return
		}
		apierrors.InternalError(w, "failed to delete upload")
		return
	}
	response.NoContent(w)
}
