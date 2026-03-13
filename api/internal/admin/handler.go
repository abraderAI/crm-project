package admin

import (
	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/upload"
)

// Handler provides HTTP handlers for admin endpoints.
type Handler struct {
	service      *Service
	auditService *audit.Service
	gdprService  *gdpr.Service
	rbacPolicy   *config.RBACPolicy
	storage      upload.StorageProvider
}

// NewHandler creates a new admin handler.
func NewHandler(service *Service, auditService *audit.Service, gdprService *gdpr.Service, rbacPolicy *config.RBACPolicy) *Handler {
	return &Handler{
		service:      service,
		auditService: auditService,
		gdprService:  gdprService,
		rbacPolicy:   rbacPolicy,
	}
}

// SetStorage sets the storage provider for exports.
func (h *Handler) SetStorage(storage upload.StorageProvider) {
	h.storage = storage
}
