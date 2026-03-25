package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// CRMHandler provides HTTP handlers for CRM AI features:
// briefing, deal strategy, and pipeline strategy.
type CRMHandler struct {
	provider  LLMProvider
	db        *gorm.DB
	ceoUserID string
}

// NewCRMHandler creates a new CRM AI handler.
func NewCRMHandler(provider LLMProvider, db *gorm.DB, ceoUserID string) *CRMHandler {
	return &CRMHandler{
		provider:  provider,
		db:        db,
		ceoUserID: ceoUserID,
	}
}

// SSEResponse is the JSON envelope for SSE streaming responses.
type SSEResponse struct {
	Content string `json:"content"`
}

// Brief handles POST /v1/orgs/{org}/crm/ai/brief.
// Loads the requesting user's ACL-scoped open opportunities, tasks, and
// recent messages, then calls LLMProvider.Briefing and streams the
// result as a single SSE event.
func (h *CRMHandler) Brief(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}
	userID := uc.UserID

	opps, err := h.loadUserOpportunities(r.Context(), userID)
	if err != nil {
		apierrors.InternalError(w, fmt.Sprintf("loading opportunities: %s", err.Error()))
		return
	}

	tasks, err := h.loadUserTasks(r.Context(), userID)
	if err != nil {
		apierrors.InternalError(w, fmt.Sprintf("loading tasks: %s", err.Error()))
		return
	}

	msgs, err := h.loadRecentMessages(r.Context(), userID)
	if err != nil {
		apierrors.InternalError(w, fmt.Sprintf("loading messages: %s", err.Error()))
		return
	}

	result, err := h.provider.Briefing(r.Context(), userID, opps, tasks, msgs)
	if err != nil {
		apierrors.InternalError(w, fmt.Sprintf("briefing generation failed: %s", err.Error()))
		return
	}

	writeSSE(w, result)
}

// DealStrategy handles POST /v1/orgs/{org}/crm/opportunities/{id}/strategy.
// Owner or admin only. Loads full opportunity context and calls DealStrategy.
func (h *CRMHandler) DealStrategy(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	oppID := chi.URLParam(r, "id")
	if oppID == "" {
		apierrors.BadRequest(w, "opportunity id is required")
		return
	}

	// Load the opportunity thread.
	var opp models.Thread
	if err := h.db.WithContext(r.Context()).Where("id = ?", oppID).First(&opp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierrors.NotFound(w, "opportunity not found")
			return
		}
		apierrors.InternalError(w, "loading opportunity")
		return
	}

	// Check ownership or admin.
	if !h.isOwnerOrAdmin(r.Context(), opp, uc.UserID) {
		apierrors.Forbidden(w, "only the opportunity owner or an admin can request deal strategy")
		return
	}

	// Load last 20 messages on this thread.
	var messages []models.Message
	h.db.WithContext(r.Context()).
		Where("thread_id = ?", oppID).
		Order("created_at DESC").
		Limit(20).
		Find(&messages)

	// Load tasks for this opportunity (simplified: query by parent thread ID
	// from crm_tasks if table exists, else empty).
	tasks := h.loadTasksForThread(r.Context(), oppID)

	result, err := h.provider.DealStrategy(r.Context(), opp, messages, tasks)
	if err != nil {
		apierrors.InternalError(w, fmt.Sprintf("deal strategy generation failed: %s", err.Error()))
		return
	}

	writeSSE(w, result)
}

// PipelineStrategy handles POST /v1/orgs/{org}/crm/ai/pipeline-strategy.
// CEO (DEFT_CEO_USER_ID) or admin only. Loads all open opportunities unfiltered.
func (h *CRMHandler) PipelineStrategy(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	if !h.isCEOOrAdmin(r.Context(), uc.UserID) {
		apierrors.Forbidden(w, "only the CEO or an admin can access pipeline strategy")
		return
	}

	orgID := chi.URLParam(r, "org")

	// Load all open opportunities for this org (unfiltered by ACL).
	var opps []models.Thread
	h.db.WithContext(r.Context()).
		Joins("JOIN boards ON boards.id = threads.board_id").
		Joins("JOIN spaces ON spaces.id = boards.space_id").
		Where("spaces.org_id = ? AND spaces.type = ?", orgID, models.SpaceTypeCRM).
		Where("threads.metadata LIKE ?", `%"crm_type":"opportunity"%`).
		Where("threads.deleted_at IS NULL").
		Where("threads.metadata NOT LIKE ?", `%"stage":"closed_won"%`).
		Where("threads.metadata NOT LIKE ?", `%"stage":"closed_lost"%`).
		Find(&opps)

	result, err := h.provider.PipelineStrategy(r.Context(), opps)
	if err != nil {
		apierrors.InternalError(w, fmt.Sprintf("pipeline strategy generation failed: %s", err.Error()))
		return
	}

	writeSSE(w, result)
}

// writeSSE writes a single SSE event with the given content and flushes.
func writeSSE(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	payload, _ := json.Marshal(SSEResponse{Content: content})
	fmt.Fprintf(w, "data: %s\n\n", payload)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// loadUserOpportunities loads open opportunity threads visible to the user
// via thread_acl (ACL-scoped). Falls back to authored threads if thread_acl
// table does not exist (prior phases not yet deployed).
func (h *CRMHandler) loadUserOpportunities(ctx context.Context, userID string) ([]models.Thread, error) {
	var opps []models.Thread

	// Try ACL-scoped query first (thread_acl from Phase 2).
	err := h.db.WithContext(ctx).
		Joins("JOIN thread_acl ON thread_acl.thread_id = threads.id AND thread_acl.user_id = ?", userID).
		Where("threads.metadata LIKE ?", `%"crm_type":"opportunity"%`).
		Where("threads.deleted_at IS NULL").
		Where("threads.metadata NOT LIKE ?", `%"stage":"closed_won"%`).
		Where("threads.metadata NOT LIKE ?", `%"stage":"closed_lost"%`).
		Order("threads.created_at DESC").
		Find(&opps).Error

	if err != nil {
		// Fallback: author-based if thread_acl table doesn't exist.
		return h.loadUserOpportunitiesFallback(ctx, userID)
	}
	return opps, nil
}

// loadUserOpportunitiesFallback loads opportunities by author_id when ACL tables
// are not available.
func (h *CRMHandler) loadUserOpportunitiesFallback(ctx context.Context, userID string) ([]models.Thread, error) {
	var opps []models.Thread
	err := h.db.WithContext(ctx).
		Where("author_id = ?", userID).
		Where("metadata LIKE ?", `%"crm_type":"opportunity"%`).
		Where("deleted_at IS NULL").
		Where("metadata NOT LIKE ?", `%"stage":"closed_won"%`).
		Where("metadata NOT LIKE ?", `%"stage":"closed_lost"%`).
		Order("created_at DESC").
		Find(&opps).Error
	return opps, err
}

// loadUserTasks loads open tasks assigned to the user from crm_tasks table.
// Returns empty slice if the table does not exist.
func (h *CRMHandler) loadUserTasks(ctx context.Context, userID string) ([]CRMTask, error) {
	var tasks []CRMTask
	rows, err := h.db.WithContext(ctx).Raw(
		`SELECT id, title, description, assigned_to, due_date, priority, status, parent_id
		 FROM crm_tasks
		 WHERE assigned_to = ? AND status IN ('open', 'in-progress') AND deleted_at IS NULL
		 ORDER BY due_date ASC, priority DESC`,
		userID,
	).Rows()
	if err != nil {
		// Table may not exist; return empty.
		return []CRMTask{}, nil
	}
	defer rows.Close()

	for rows.Next() {
		var t CRMTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.DueDate, &t.Priority, &t.Status, &t.ParentID); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// loadRecentMessages loads recent messages on the user's entity threads (last 7 days).
func (h *CRMHandler) loadRecentMessages(ctx context.Context, userID string) ([]models.Message, error) {
	var msgs []models.Message

	err := h.db.WithContext(ctx).
		Joins("JOIN thread_acl ON thread_acl.thread_id = messages.thread_id AND thread_acl.user_id = ?", userID).
		Where("messages.created_at >= datetime('now', '-7 days')").
		Order("messages.created_at DESC").
		Limit(50).
		Find(&msgs).Error

	if err != nil {
		// Fallback: messages authored by user.
		h.db.WithContext(ctx).
			Where("author_id = ? AND created_at >= datetime('now', '-7 days')", userID).
			Order("created_at DESC").
			Limit(50).
			Find(&msgs)
	}
	return msgs, nil
}

// loadTasksForThread loads tasks linked to a specific thread (opportunity).
func (h *CRMHandler) loadTasksForThread(ctx context.Context, threadID string) []CRMTask {
	var tasks []CRMTask
	rows, err := h.db.WithContext(ctx).Raw(
		`SELECT id, title, description, assigned_to, due_date, priority, status, parent_id
		 FROM crm_tasks
		 WHERE parent_id = ? AND deleted_at IS NULL
		 ORDER BY due_date ASC`,
		threadID,
	).Rows()
	if err != nil {
		return []CRMTask{}
	}
	defer rows.Close()

	for rows.Next() {
		var t CRMTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.DueDate, &t.Priority, &t.Status, &t.ParentID); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks
}

// isOwnerOrAdmin checks if the user is the opportunity's author/owner or
// an org admin.
func (h *CRMHandler) isOwnerOrAdmin(ctx context.Context, opp models.Thread, userID string) bool {
	// Check if user is the author (owner).
	if opp.AuthorID == userID {
		return true
	}

	// Check thread_acl for owner grant.
	var count int64
	h.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM thread_acl WHERE thread_id = ? AND user_id = ? AND grant_type = 'owner'`,
		opp.ID, userID,
	).Scan(&count)
	if count > 0 {
		return true
	}

	// Check org admin role via org membership.
	if opp.OrgID != nil {
		var adminCount int64
		h.db.WithContext(ctx).Model(&models.OrgMembership{}).
			Where("org_id = ? AND user_id = ? AND role IN ('admin', 'owner')", *opp.OrgID, userID).
			Count(&adminCount)
		if adminCount > 0 {
			return true
		}
	}

	return false
}

// isCEOOrAdmin checks if the user is the CEO (by DEFT_CEO_USER_ID env) or
// a platform admin.
func (h *CRMHandler) isCEOOrAdmin(ctx context.Context, userID string) bool {
	if h.ceoUserID != "" && userID == h.ceoUserID {
		return true
	}

	// Check platform admin status.
	var count int64
	h.db.WithContext(ctx).Model(&models.PlatformAdmin{}).
		Where("user_id = ?", userID).
		Count(&count)
	return count > 0
}
