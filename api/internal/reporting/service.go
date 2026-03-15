package reporting

import "context"

// ReportingService orchestrates reporting queries and formats results.
type ReportingService struct {
	repo ReportingRepository
}

// NewService creates a new ReportingService.
func NewService(repo ReportingRepository) *ReportingService {
	return &ReportingService{repo: repo}
}

// GetSupportMetrics gathers all 7 support metrics for an org.
func (s *ReportingService) GetSupportMetrics(ctx context.Context, orgID string, params ReportParams) (*SupportMetrics, error) {
	statusBreakdown, err := s.repo.GetStatusBreakdown(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	volume, err := s.repo.GetVolumeOverTime(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	avgRes, err := s.repo.GetAvgResolutionHours(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	byAssignee, err := s.repo.GetTicketsByAssignee(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	byPriority, err := s.repo.GetTicketsByPriority(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	avgFirst, err := s.repo.GetAvgFirstResponseHours(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	overdue, err := s.repo.GetOverdueCount(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	return &SupportMetrics{
		StatusBreakdown:       statusBreakdown,
		VolumeOverTime:        volume,
		AvgResolutionHours:    avgRes,
		TicketsByAssignee:     byAssignee,
		TicketsByPriority:     byPriority,
		AvgFirstResponseHours: avgFirst,
		OverdueCount:          overdue,
	}, nil
}

// GetSalesMetrics gathers all 8 sales metrics for an org.
func (s *ReportingService) GetSalesMetrics(ctx context.Context, orgID string, params ReportParams) (*SalesMetrics, error) {
	funnel, err := s.repo.GetPipelineFunnel(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	velocity, err := s.repo.GetLeadVelocity(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	won, lost, err := s.repo.GetWinLossCounts(ctx, orgID, params)
	if err != nil {
		return nil, err
	}
	winRate, lossRate := computeWinLossRates(won, lost)

	avgDeal, err := s.repo.GetAvgDealValue(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	byAssignee, err := s.repo.GetLeadsByAssignee(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	scoreDist, err := s.repo.GetScoreDistribution(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	transitions, err := s.repo.GetStageTransitions(ctx, orgID, params)
	if err != nil {
		return nil, err
	}
	convRates := computeConversionRates(transitions)

	avgTime, err := s.repo.GetAvgTimeInStage(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	return &SalesMetrics{
		PipelineFunnel:       funnel,
		LeadVelocity:         velocity,
		WinRate:              winRate,
		LossRate:             lossRate,
		AvgDealValue:         avgDeal,
		LeadsByAssignee:      byAssignee,
		ScoreDistribution:    scoreDist,
		StageConversionRates: convRates,
		AvgTimeInStage:       avgTime,
	}, nil
}

// GetSupportExportRows delegates to the repository for CSV export data.
func (s *ReportingService) GetSupportExportRows(ctx context.Context, orgID string, params ReportParams) ([]SupportExportRow, error) {
	return s.repo.GetSupportExportRows(ctx, orgID, params)
}

// GetSalesExportRows delegates to the repository for CSV export data.
func (s *ReportingService) GetSalesExportRows(ctx context.Context, orgID string, params ReportParams) ([]SalesExportRow, error) {
	return s.repo.GetSalesExportRows(ctx, orgID, params)
}

// computeWinLossRates calculates win and loss rates from counts.
// Returns 0.0 for both if the denominator is zero.
func computeWinLossRates(won, lost int64) (winRate, lossRate float64) {
	total := won + lost
	if total == 0 {
		return 0.0, 0.0
	}
	return float64(won) / float64(total), float64(lost) / float64(total)
}

// computeConversionRates turns raw transition counts into per-from_stage rates.
// rate = count(from→to) / SUM(count(from→*))
func computeConversionRates(transitions []stageTransitionRow) []StageConversion {
	if len(transitions) == 0 {
		return []StageConversion{}
	}

	// Sum total outgoing transitions per from_stage.
	totals := make(map[string]int64)
	for _, t := range transitions {
		totals[t.FromStage] += t.TransitionCount
	}

	rates := make([]StageConversion, 0, len(transitions))
	for _, t := range transitions {
		rate := float64(0)
		if totals[t.FromStage] > 0 {
			rate = float64(t.TransitionCount) / float64(totals[t.FromStage])
		}
		rates = append(rates, StageConversion{
			FromStage: t.FromStage,
			ToStage:   t.ToStage,
			Rate:      rate,
		})
	}
	return rates
}
