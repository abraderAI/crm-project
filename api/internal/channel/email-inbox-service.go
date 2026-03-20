package channel

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
)

const passwordRedacted = "[REDACTED]"

// EmailInboxService provides business logic for managing EmailInbox records.
type EmailInboxService struct {
	repo *EmailInboxRepository
}

// NewEmailInboxService creates a new EmailInboxService.
func NewEmailInboxService(repo *EmailInboxRepository) *EmailInboxService {
	return &EmailInboxService{repo: repo}
}

// CreateInboxInput holds validated fields for creating an EmailInbox.
type CreateInboxInput struct {
	Name          string               `json:"name"`
	EmailAddress  string               `json:"email_address"`
	IMAPHost      string               `json:"imap_host"`
	IMAPPort      int                  `json:"imap_port"`
	Username      string               `json:"username"`
	Password      string               `json:"password"`
	Mailbox       string               `json:"mailbox"`
	RoutingAction models.RoutingAction `json:"routing_action"`
	Enabled       bool                 `json:"enabled"`
}

// UpdateInboxInput holds fields that may be changed on an existing EmailInbox.
// A blank Password means "keep the existing password".
type UpdateInboxInput struct {
	Name          string               `json:"name"`
	EmailAddress  string               `json:"email_address"`
	IMAPHost      string               `json:"imap_host"`
	IMAPPort      int                  `json:"imap_port"`
	Username      string               `json:"username"`
	Password      string               `json:"password"`
	Mailbox       string               `json:"mailbox"`
	RoutingAction models.RoutingAction `json:"routing_action"`
	Enabled       bool                 `json:"enabled"`
}

// validateCreate checks required fields in a create request.
func validateCreate(in CreateInboxInput) error {
	if in.Name == "" {
		return fmt.Errorf("name is required")
	}
	if in.IMAPHost == "" {
		return fmt.Errorf("imap_host is required")
	}
	if in.IMAPPort <= 0 {
		return fmt.Errorf("imap_port must be a positive integer")
	}
	if in.Username == "" {
		return fmt.Errorf("username is required")
	}
	if in.Password == "" {
		return fmt.Errorf("password is required")
	}
	if in.RoutingAction != "" && !in.RoutingAction.IsValid() {
		return fmt.Errorf("routing_action %q is not valid; allowed: support_ticket, sales_lead, general", in.RoutingAction)
	}
	return nil
}

// Create validates the input and creates a new EmailInbox.
// The returned inbox has its password masked.
func (s *EmailInboxService) Create(ctx context.Context, orgID string, in CreateInboxInput) (*models.EmailInbox, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}

	action := in.RoutingAction
	if action == "" {
		action = models.RoutingActionSupportTicket
	}
	mailbox := in.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	inbox := &models.EmailInbox{
		OrgID:         orgID,
		Name:          in.Name,
		EmailAddress:  in.EmailAddress,
		IMAPHost:      in.IMAPHost,
		IMAPPort:      in.IMAPPort,
		Username:      in.Username,
		Password:      in.Password,
		Mailbox:       mailbox,
		RoutingAction: action,
		Enabled:       in.Enabled,
	}
	if err := s.repo.Create(ctx, inbox); err != nil {
		return nil, err
	}

	inbox.Password = passwordRedacted
	return inbox, nil
}

// List returns all inboxes for the org, with passwords masked.
func (s *EmailInboxService) List(ctx context.Context, orgID string) ([]models.EmailInbox, error) {
	inboxes, err := s.repo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for i := range inboxes {
		if inboxes[i].Password != "" {
			inboxes[i].Password = passwordRedacted
		}
	}
	return inboxes, nil
}

// Get retrieves a single inbox by ID, scoped to the org.
// Returns nil, nil when not found or the inbox belongs to a different org.
func (s *EmailInboxService) Get(ctx context.Context, orgID, id string) (*models.EmailInbox, error) {
	inbox, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inbox == nil || inbox.OrgID != orgID {
		return nil, nil
	}
	if inbox.Password != "" {
		inbox.Password = passwordRedacted
	}
	return inbox, nil
}

// Update applies changes to an existing inbox. A blank Password in the input
// leaves the stored password unchanged.
// Returns nil, nil when the inbox is not found or belongs to another org.
func (s *EmailInboxService) Update(ctx context.Context, orgID, id string, in UpdateInboxInput) (*models.EmailInbox, error) {
	inbox, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inbox == nil || inbox.OrgID != orgID {
		return nil, nil
	}

	if in.Name != "" {
		inbox.Name = in.Name
	}
	inbox.EmailAddress = in.EmailAddress
	if in.IMAPHost != "" {
		inbox.IMAPHost = in.IMAPHost
	}
	if in.IMAPPort > 0 {
		inbox.IMAPPort = in.IMAPPort
	}
	if in.Username != "" {
		inbox.Username = in.Username
	}
	if in.Password != "" && in.Password != passwordRedacted {
		inbox.Password = in.Password
	}
	if in.Mailbox != "" {
		inbox.Mailbox = in.Mailbox
	}
	if in.RoutingAction != "" {
		if !in.RoutingAction.IsValid() {
			return nil, fmt.Errorf("routing_action %q is not valid", in.RoutingAction)
		}
		inbox.RoutingAction = in.RoutingAction
	}
	inbox.Enabled = in.Enabled

	if err := s.repo.Save(ctx, inbox); err != nil {
		return nil, err
	}
	inbox.Password = passwordRedacted
	return inbox, nil
}

// Delete soft-deletes an inbox scoped to the org.
// Returns false, nil when not found or the inbox belongs to another org.
func (s *EmailInboxService) Delete(ctx context.Context, orgID, id string) (bool, error) {
	inbox, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	if inbox == nil || inbox.OrgID != orgID {
		return false, nil
	}
	if err := s.repo.SoftDelete(ctx, inbox); err != nil {
		return false, err
	}
	return true, nil
}
