// Package gdpr provides GDPR compliance endpoints for user data export,
// user PII purge with audit anonymization, and org cascade purge.
package gdpr

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Service provides GDPR compliance operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new GDPR service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// UserExport contains all data belonging to a user.
type UserExport struct {
	UserID        string                          `json:"user_id"`
	Memberships   UserMemberships                 `json:"memberships"`
	Threads       []models.Thread                 `json:"threads"`
	Messages      []models.Message                `json:"messages"`
	Votes         []models.Vote                   `json:"votes"`
	Notifications []models.Notification           `json:"notifications"`
	CallLogs      []models.CallLog                `json:"call_logs"`
	Uploads       []models.Upload                 `json:"uploads"`
	AuditLogs     []models.AuditLog               `json:"audit_logs"`
	Preferences   []models.NotificationPreference `json:"preferences"`
	Digests       []models.DigestSchedule         `json:"digests"`
}

// UserMemberships groups membership data by level.
type UserMemberships struct {
	Orgs   []models.OrgMembership   `json:"orgs"`
	Spaces []models.SpaceMembership `json:"spaces"`
	Boards []models.BoardMembership `json:"boards"`
}

// ExportUserData collects all data associated with a user into a JSON archive.
func (s *Service) ExportUserData(ctx context.Context, userID string) (*UserExport, error) {
	export := &UserExport{UserID: userID}

	// Memberships.
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Memberships.Orgs).Error; err != nil {
		return nil, fmt.Errorf("exporting org memberships: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Memberships.Spaces).Error; err != nil {
		return nil, fmt.Errorf("exporting space memberships: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Memberships.Boards).Error; err != nil {
		return nil, fmt.Errorf("exporting board memberships: %w", err)
	}

	// Content.
	if err := s.db.WithContext(ctx).Unscoped().Where("author_id = ?", userID).Find(&export.Threads).Error; err != nil {
		return nil, fmt.Errorf("exporting threads: %w", err)
	}
	if err := s.db.WithContext(ctx).Unscoped().Where("author_id = ?", userID).Find(&export.Messages).Error; err != nil {
		return nil, fmt.Errorf("exporting messages: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Votes).Error; err != nil {
		return nil, fmt.Errorf("exporting votes: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Notifications).Error; err != nil {
		return nil, fmt.Errorf("exporting notifications: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("caller_id = ?", userID).Find(&export.CallLogs).Error; err != nil {
		return nil, fmt.Errorf("exporting call logs: %w", err)
	}
	if err := s.db.WithContext(ctx).Unscoped().Where("uploader_id = ?", userID).Find(&export.Uploads).Error; err != nil {
		return nil, fmt.Errorf("exporting uploads: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.AuditLogs).Error; err != nil {
		return nil, fmt.Errorf("exporting audit logs: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Preferences).Error; err != nil {
		return nil, fmt.Errorf("exporting preferences: %w", err)
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&export.Digests).Error; err != nil {
		return nil, fmt.Errorf("exporting digests: %w", err)
	}

	return export, nil
}

// ExportUserDataJSON returns the user export as a JSON byte slice.
func (s *Service) ExportUserDataJSON(ctx context.Context, userID string) ([]byte, error) {
	export, err := s.ExportUserData(ctx, userID)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(export, "", "  ")
}

// PurgeUser removes all PII for a user and anonymizes their audit log entries.
// This is a hard delete — data is unrecoverable.
func (s *Service) PurgeUser(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Hard-delete memberships.
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.OrgMembership{}).Error; err != nil {
			return fmt.Errorf("purging org memberships: %w", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.SpaceMembership{}).Error; err != nil {
			return fmt.Errorf("purging space memberships: %w", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.BoardMembership{}).Error; err != nil {
			return fmt.Errorf("purging board memberships: %w", err)
		}

		// Hard-delete user content.
		if err := tx.Unscoped().Where("author_id = ?", userID).Delete(&models.Message{}).Error; err != nil {
			return fmt.Errorf("purging messages: %w", err)
		}
		if err := tx.Unscoped().Where("author_id = ?", userID).Delete(&models.Thread{}).Error; err != nil {
			return fmt.Errorf("purging threads: %w", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Vote{}).Error; err != nil {
			return fmt.Errorf("purging votes: %w", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Notification{}).Error; err != nil {
			return fmt.Errorf("purging notifications: %w", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.NotificationPreference{}).Error; err != nil {
			return fmt.Errorf("purging notification preferences: %w", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&models.DigestSchedule{}).Error; err != nil {
			return fmt.Errorf("purging digest schedules: %w", err)
		}
		if err := tx.Unscoped().Where("uploader_id = ?", userID).Delete(&models.Upload{}).Error; err != nil {
			return fmt.Errorf("purging uploads: %w", err)
		}
		if err := tx.Unscoped().Where("caller_id = ?", userID).Delete(&models.CallLog{}).Error; err != nil {
			return fmt.Errorf("purging call logs: %w", err)
		}

		// Anonymize audit log entries — do not delete, just replace PII.
		anonymized := "anonymized"
		if err := tx.Model(&models.AuditLog{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"user_id":    anonymized,
				"ip_address": "",
			}).Error; err != nil {
			return fmt.Errorf("anonymizing audit logs: %w", err)
		}

		return nil
	})
}

// PurgeOrg performs a cascade hard-delete of an org and all its children.
// This is irreversible.
func (s *Service) PurgeOrg(ctx context.Context, orgID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all spaces for the org.
		var spaces []models.Space
		if err := tx.Unscoped().Where("org_id = ?", orgID).Find(&spaces).Error; err != nil {
			return fmt.Errorf("finding spaces: %w", err)
		}

		for _, sp := range spaces {
			// Get all boards for this space.
			var boards []models.Board
			if err := tx.Unscoped().Where("space_id = ?", sp.ID).Find(&boards).Error; err != nil {
				return fmt.Errorf("finding boards: %w", err)
			}

			for _, bd := range boards {
				// Get all threads for this board.
				var threads []models.Thread
				if err := tx.Unscoped().Where("board_id = ?", bd.ID).Find(&threads).Error; err != nil {
					return fmt.Errorf("finding threads: %w", err)
				}

				for _, th := range threads {
					// Delete messages for thread.
					if err := tx.Unscoped().Where("thread_id = ?", th.ID).Delete(&models.Message{}).Error; err != nil {
						return fmt.Errorf("purging messages: %w", err)
					}
					// Delete votes for thread.
					if err := tx.Unscoped().Where("thread_id = ?", th.ID).Delete(&models.Vote{}).Error; err != nil {
						return fmt.Errorf("purging votes: %w", err)
					}
					// Delete revisions for thread.
					if err := tx.Unscoped().Where("entity_type = ? AND entity_id = ?", "thread", th.ID).Delete(&models.Revision{}).Error; err != nil {
						return fmt.Errorf("purging thread revisions: %w", err)
					}
				}

				// Delete threads for board.
				if err := tx.Unscoped().Where("board_id = ?", bd.ID).Delete(&models.Thread{}).Error; err != nil {
					return fmt.Errorf("purging threads: %w", err)
				}
				// Delete board memberships.
				if err := tx.Unscoped().Where("board_id = ?", bd.ID).Delete(&models.BoardMembership{}).Error; err != nil {
					return fmt.Errorf("purging board memberships: %w", err)
				}
			}

			// Delete boards for space.
			if err := tx.Unscoped().Where("space_id = ?", sp.ID).Delete(&models.Board{}).Error; err != nil {
				return fmt.Errorf("purging boards: %w", err)
			}
			// Delete space memberships.
			if err := tx.Unscoped().Where("space_id = ?", sp.ID).Delete(&models.SpaceMembership{}).Error; err != nil {
				return fmt.Errorf("purging space memberships: %w", err)
			}
		}

		// Delete spaces.
		if err := tx.Unscoped().Where("org_id = ?", orgID).Delete(&models.Space{}).Error; err != nil {
			return fmt.Errorf("purging spaces: %w", err)
		}

		// Delete org-level resources.
		if err := tx.Unscoped().Where("org_id = ?", orgID).Delete(&models.OrgMembership{}).Error; err != nil {
			return fmt.Errorf("purging org memberships: %w", err)
		}
		if err := tx.Unscoped().Where("org_id = ?", orgID).Delete(&models.APIKey{}).Error; err != nil {
			return fmt.Errorf("purging api keys: %w", err)
		}
		if err := tx.Unscoped().Where("org_id = ?", orgID).Delete(&models.WebhookSubscription{}).Error; err != nil {
			return fmt.Errorf("purging webhook subscriptions: %w", err)
		}
		if err := tx.Unscoped().Where("org_id = ?", orgID).Delete(&models.Upload{}).Error; err != nil {
			return fmt.Errorf("purging uploads: %w", err)
		}
		if err := tx.Unscoped().Where("org_id = ?", orgID).Delete(&models.CallLog{}).Error; err != nil {
			return fmt.Errorf("purging call logs: %w", err)
		}

		// Delete the org itself.
		if err := tx.Unscoped().Where("id = ?", orgID).Delete(&models.Org{}).Error; err != nil {
			return fmt.Errorf("purging org: %w", err)
		}

		return nil
	})
}
