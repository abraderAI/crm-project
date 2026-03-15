package reporting

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// GetAdminSupportMetrics gathers platform-wide support metrics and per-org
// breakdown concurrently using errgroup.
func (s *ReportingService) GetAdminSupportMetrics(ctx context.Context, params ReportParams) (*AdminSupportMetrics, error) {
	repo := s.repo.(*repository)

	var (
		platformMetrics *SupportMetrics
		breakdown       []OrgSupportSummary
		firstResp       map[string]*float64
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		platformMetrics, err = repo.GetPlatformSupportMetrics(gctx, params)
		return err
	})

	g.Go(func() error {
		var err error
		breakdown, err = repo.GetOrgSupportBreakdown(gctx, params)
		return err
	})

	g.Go(func() error {
		var err error
		firstResp, err = repo.GetOrgFirstResponseBreakdown(gctx, params)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Join first response hours into the breakdown.
	for i := range breakdown {
		if avg, ok := firstResp[breakdown[i].OrgID]; ok {
			breakdown[i].AvgFirstResponseHours = avg
		}
	}

	return &AdminSupportMetrics{
		SupportMetrics: *platformMetrics,
		OrgBreakdown:   breakdown,
	}, nil
}

// GetAdminSalesMetrics gathers platform-wide sales metrics and per-org
// breakdown concurrently using errgroup.
func (s *ReportingService) GetAdminSalesMetrics(ctx context.Context, params ReportParams) (*AdminSalesMetrics, error) {
	repo := s.repo.(*repository)

	var (
		platformMetrics *SalesMetrics
		breakdown       []OrgSalesSummary
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		platformMetrics, err = repo.GetPlatformSalesMetrics(gctx, params)
		return err
	})

	g.Go(func() error {
		var err error
		breakdown, err = repo.GetOrgSalesBreakdown(gctx, params)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &AdminSalesMetrics{
		SalesMetrics: *platformMetrics,
		OrgBreakdown: breakdown,
	}, nil
}

// GetAdminSupportExportRows returns cross-org support export data.
func (s *ReportingService) GetAdminSupportExportRows(ctx context.Context, params ReportParams) ([]AdminSupportExportRow, error) {
	repo := s.repo.(*repository)
	return repo.GetAdminSupportExportRows(ctx, params)
}

// GetAdminSalesExportRows returns cross-org sales export data.
func (s *ReportingService) GetAdminSalesExportRows(ctx context.Context, params ReportParams) ([]AdminSalesExportRow, error) {
	repo := s.repo.(*repository)
	return repo.GetAdminSalesExportRows(ctx, params)
}
