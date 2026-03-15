package reporting

import (
	"context"
	"fmt"
)

// Service provides business logic for reporting operations.
type Service struct {
	repo *Repository
}

// NewService creates a new reporting Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetSupportMetrics gathers all 7 support metric queries and returns the
// aggregated result.
func (s *Service) GetSupportMetrics(ctx context.Context, orgID string, params ReportParams) (*SupportMetrics, error) {
	statusBreakdown, err := s.repo.GetStatusBreakdown(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("status breakdown: %w", err)
	}

	volumeOverTime, err := s.repo.GetVolumeOverTime(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("volume over time: %w", err)
	}

	avgResolution, err := s.repo.GetAvgResolutionHours(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("avg resolution: %w", err)
	}

	ticketsByAssignee, err := s.repo.GetTicketsByAssignee(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("tickets by assignee: %w", err)
	}

	ticketsByPriority, err := s.repo.GetTicketsByPriority(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("tickets by priority: %w", err)
	}

	avgFirstResponse, err := s.repo.GetAvgFirstResponseHours(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("avg first response: %w", err)
	}

	overdueCount, err := s.repo.GetOverdueCount(ctx, orgID, params)
	if err != nil {
		return nil, fmt.Errorf("overdue count: %w", err)
	}

	return &SupportMetrics{
		StatusBreakdown:       statusBreakdown,
		VolumeOverTime:        volumeOverTime,
		AvgResolutionHours:    avgResolution,
		TicketsByAssignee:     ticketsByAssignee,
		TicketsByPriority:     ticketsByPriority,
		AvgFirstResponseHours: avgFirstResponse,
		OverdueCount:          overdueCount,
	}, nil
}

// IsOrgAdmin returns true when the user has admin or owner role in the org.
func (s *Service) IsOrgAdmin(ctx context.Context, orgID, userID string) (bool, error) {
	return s.repo.IsOrgAdmin(ctx, orgID, userID)
}

// ScanExportRows delegates CSV row streaming to the repository.
func (s *Service) ScanExportRows(ctx context.Context, orgID string, params ReportParams, fn func(ExportRow) error) error {
	return s.repo.ScanExportRows(ctx, orgID, params, fn)
}
