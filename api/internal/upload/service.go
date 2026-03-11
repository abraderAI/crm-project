package upload

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// AllowedContentTypes defines permitted file MIME types.
var AllowedContentTypes = map[string]bool{
	"image/jpeg":               true,
	"image/png":                true,
	"image/gif":                true,
	"image/webp":               true,
	"image/svg+xml":            true,
	"application/pdf":          true,
	"text/plain":               true,
	"text/csv":                 true,
	"text/markdown":            true,
	"application/json":         true,
	"application/xml":          true,
	"application/zip":          true,
	"application/octet-stream": true,
}

// Service provides business logic for file uploads.
type Service struct {
	db      *gorm.DB
	storage StorageProvider
	maxSize int64
}

// NewService creates a new upload service.
func NewService(db *gorm.DB, storage StorageProvider, maxSize int64) *Service {
	return &Service{db: db, storage: storage, maxSize: maxSize}
}

// ValidateContentType checks if a content type is allowed.
func ValidateContentType(contentType string) bool {
	// Normalize: take only the mime type part (before params).
	ct := strings.Split(contentType, ";")[0]
	ct = strings.TrimSpace(ct)
	return AllowedContentTypes[ct]
}

// DetectContentType reads the first 512 bytes to detect the MIME type.
func DetectContentType(file multipart.File) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading file header: %w", err)
	}
	// Seek back to the beginning.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seeking file: %w", err)
	}
	return http.DetectContentType(buf[:n]), nil
}

// Create validates and stores an uploaded file.
func (s *Service) Create(ctx context.Context, orgID, entityType, entityID, uploaderID, filename string, size int64, file multipart.File) (*models.Upload, error) {
	if orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	if size > s.maxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", size, s.maxSize)
	}

	// Detect content type.
	contentType, err := DetectContentType(file)
	if err != nil {
		return nil, err
	}
	if !ValidateContentType(contentType) {
		return nil, fmt.Errorf("content type %q is not allowed", contentType)
	}

	// Store the file.
	storagePath, err := s.storage.Store(filename, file)
	if err != nil {
		return nil, fmt.Errorf("storing file: %w", err)
	}

	upload := &models.Upload{
		OrgID:       orgID,
		EntityType:  entityType,
		EntityID:    entityID,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		StoragePath: storagePath,
		UploaderID:  uploaderID,
	}
	if err := s.db.WithContext(ctx).Create(upload).Error; err != nil {
		// Clean up stored file on DB error.
		_ = s.storage.Delete(storagePath)
		return nil, fmt.Errorf("saving upload record: %w", err)
	}

	return upload, nil
}

// Get retrieves an upload record by ID.
func (s *Service) Get(ctx context.Context, id string) (*models.Upload, error) {
	var upload models.Upload
	if err := s.db.WithContext(ctx).First(&upload, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("finding upload: %w", err)
	}
	return &upload, nil
}

// GetFile returns a reader for the uploaded file content.
func (s *Service) GetFile(storagePath string) (io.ReadCloser, error) {
	return s.storage.Get(storagePath)
}

// Delete soft-deletes an upload record and removes the stored file.
func (s *Service) Delete(ctx context.Context, id string) error {
	var upload models.Upload
	if err := s.db.WithContext(ctx).First(&upload, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("not found")
		}
		return fmt.Errorf("finding upload: %w", err)
	}

	// Delete from storage.
	_ = s.storage.Delete(upload.StoragePath)

	// Soft delete record.
	if err := s.db.WithContext(ctx).Delete(&upload).Error; err != nil {
		return fmt.Errorf("deleting upload record: %w", err)
	}

	return nil
}
