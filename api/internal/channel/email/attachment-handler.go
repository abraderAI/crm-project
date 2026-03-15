package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
)

const (
	// InlineImageSizeThreshold is the maximum size of inline images to skip (10KB).
	InlineImageSizeThreshold = 10 * 1024
)

// AttachmentHandler processes email attachments by uploading them via
// the StorageProvider and creating Upload records linked to a message.
type AttachmentHandler struct {
	db      *gorm.DB
	storage upload.StorageProvider
}

// NewAttachmentHandler creates a new attachment handler.
func NewAttachmentHandler(db *gorm.DB, storage upload.StorageProvider) *AttachmentHandler {
	return &AttachmentHandler{db: db, storage: storage}
}

// ProcessAttachments uploads each attachment from a parsed email and creates
// Upload records linked to the given message. Inline images under 10KB are skipped.
// Returns the list of created Upload records.
func (h *AttachmentHandler) ProcessAttachments(
	ctx context.Context,
	orgID string,
	messageID string,
	attachments []ParsedAttachment,
) ([]*models.Upload, error) {
	if len(attachments) == 0 {
		return nil, nil
	}

	var uploads []*models.Upload
	for _, att := range attachments {
		// Skip inline images under 10KB.
		if att.IsInline && isImageContentType(att.ContentType) && int64(len(att.Data)) < InlineImageSizeThreshold {
			continue
		}

		// Upload to storage.
		storagePath, err := h.storage.Store(att.Filename, bytes.NewReader(att.Data))
		if err != nil {
			return uploads, fmt.Errorf("storing attachment %s: %w", att.Filename, err)
		}

		// Create Upload record.
		rec := &models.Upload{
			OrgID:       orgID,
			EntityType:  "message",
			EntityID:    messageID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        int64(len(att.Data)),
			StoragePath: storagePath,
			UploaderID:  "system",
		}
		if err := h.db.WithContext(ctx).Create(rec).Error; err != nil {
			// Clean up stored file on DB error.
			_ = h.storage.Delete(storagePath)
			return uploads, fmt.Errorf("creating upload record for %s: %w", att.Filename, err)
		}
		uploads = append(uploads, rec)
	}

	return uploads, nil
}

// isImageContentType returns true for image/* MIME types.
func isImageContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "image/")
}

// MockStorageProvider is a test double for upload.StorageProvider.
type MockStorageProvider struct {
	Files      map[string][]byte
	StoreCalls int
	StoreErr   error
	DeleteErr  error
}

// NewMockStorageProvider creates a new mock storage provider.
func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		Files: make(map[string][]byte),
	}
}

// Store saves the file content in memory and returns the filename as path.
func (m *MockStorageProvider) Store(filename string, content io.Reader) (string, error) {
	m.StoreCalls++
	if m.StoreErr != nil {
		return "", m.StoreErr
	}
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	path := "stored-" + filename
	m.Files[path] = data
	return path, nil
}

// Get returns a reader for the stored content.
func (m *MockStorageProvider) Get(storagePath string) (io.ReadCloser, error) {
	data, ok := m.Files[storagePath]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", storagePath)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Delete removes the file from memory.
func (m *MockStorageProvider) Delete(storagePath string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	delete(m.Files, storagePath)
	return nil
}
