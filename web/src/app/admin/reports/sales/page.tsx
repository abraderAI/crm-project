"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertCircle } from "lucide-react";

import type { AdminSalesMetrics } from "@/lib/reporting-types";
import { buildHeaders, buildUrl } from "@/lib/api-client";
import { getAdminSalesExportUrl } from "@/lib/reporting-api";

import { MetricCard } from "@/components/reports/metric-card";
import { DateRangePicker } from "@/components/reports/date-range-picker";
import { ExportButton } from "@/components/reports/export-button";
import { PipelineFunnelChart } from "@/components/reports/sales/pipeline-funnel-chart";
import { LeadVelocityChart } from "@/components/reports/sales/lead-velocity-chart";
import { ScoreDistributionChart } from "@/components/reports/sales/score-distribution-chart";
import { OrgBreakdownTable } from "@/components/reports/org-breakdown-table";

/** Format a Date as YYYY-MM-DD. */
function toDateString(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

/** Parse a YYYY-MM-DD string into a Date, or return fallback. */
function parseDate(value: string | null, fallback: Date): Date {
  if (!value) return fallback;
  const d = new Date(value + "T00:00:00");
  return isNaN(d.getTime()) ? fallback : d;
}

/** Skeleton placeholder for a chart area. */
function ChartSkeleton(): React.ReactNode {
  return (
    <div data-testid="chart-skeleton" className="h-72 w-full animate-pulse rounded-lg bg-muted" />
  );
}

/** Platform admin sales dashboard page. */
export default function AdminSalesPage(): React.ReactNode {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { getToken } = useAuth();

  const now = new Date();
  const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

  const [from, setFrom] = useState<Date>(parseDate(searchParams.get("from"), thirtyDaysAgo));
  const [to, setTo] = useState<Date>(parseDate(searchParams.get("to"), now));

  const [data, setData] = useState<AdminSalesMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  /** Fetch admin sales metrics from the API. */
  const fetchMetrics = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const token = await getToken();
      const queryParams: Record<string, string> = {
        from: toDateString(from),
        to: toDateString(to),
      };

      const url = buildUrl("/admin/reports/sales", queryParams);
      const response = await fetch(url, {
        method: "GET",
        headers: buildHeaders(token),
      });

      if (!response.ok) {
        throw new Error(`Failed to load sales data: ${response.status}`);
      }

      const metrics = (await response.json()) as AdminSalesMetrics;
      setData(metrics);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load sales data";
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [getToken, from, to]);

  useEffect(() => {
    void fetchMetrics();
  }, [fetchMetrics]);

  /** Sync date filter to URL. */
  useEffect(() => {
    const params = new URLSearchParams();
    params.set("from", toDateString(from));
    params.set("to", toDateString(to));
    router.replace(`/admin/reports/sales?${params.toString()}`, { scroll: false });
  }, [from, to, router]);

  function handleDateChange(range: { from: Date; to: Date }): void {
    setFrom(range.from);
    setTo(range.to);
  }

  const exportUrl = useMemo(
    () =>
      getAdminSalesExportUrl({
        from: toDateString(from),
        to: toDateString(to),
      }),
    [from, to],
  );

  /** Compute total leads from pipeline funnel. */
  const totalLeads = useMemo(() => {
    if (!data) return 0;
    return data.pipeline_funnel.reduce((sum, stage) => sum + stage.count, 0);
  }, [data]);

  return (
    <div data-testid="admin-sales-dashboard" className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-foreground" data-testid="admin-sales-title">
          Platform Sales Overview
        </h2>
        <ExportButton url={exportUrl} filename="admin-sales-report.csv" />
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3" data-testid="admin-sales-filter-bar">
        <DateRangePicker from={from} to={to} onChange={handleDateChange} />
      </div>

      {/* Error alert */}
      {error && (
        <div
          data-testid="admin-sales-error"
          className="flex items-center gap-2 rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          <p>{error}</p>
        </div>
      )}

      {/* KPI row */}
      <div className="grid gap-4 sm:grid-cols-3" data-testid="admin-sales-kpi-row">
        <MetricCard label="Total Leads" value={loading ? "–" : totalLeads} loading={loading} />
        <MetricCard
          label="Platform Win Rate"
          value={data ? `${(data.win_rate * 100).toFixed(1)}%` : "–"}
          loading={loading}
        />
        <MetricCard
          label="Avg Deal Value"
          value={data?.avg_deal_value ? `$${data.avg_deal_value.toLocaleString()}` : "–"}
          loading={loading}
        />
      </div>

      {/* Charts grid */}
      <div className="grid gap-6 md:grid-cols-2" data-testid="admin-sales-charts-grid">
        <div
          className="rounded-lg border border-border p-4"
          data-testid="chart-section-pipeline-funnel"
        >
          <h3 className="mb-2 text-sm font-medium text-foreground">Pipeline Funnel</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <PipelineFunnelChart data={data.pipeline_funnel} />
          ) : null}
        </div>

        <div
          className="rounded-lg border border-border p-4"
          data-testid="chart-section-lead-velocity"
        >
          <h3 className="mb-2 text-sm font-medium text-foreground">Lead Velocity</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <LeadVelocityChart data={data.lead_velocity} />
          ) : null}
        </div>

        <div
          className="rounded-lg border border-border p-4"
          data-testid="chart-section-score-distribution"
        >
          <h3 className="mb-2 text-sm font-medium text-foreground">Score Distribution</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <ScoreDistributionChart data={data.score_distribution} />
          ) : null}
        </div>
      </div>

      {/* Per-org breakdown */}
      <div className="flex flex-col gap-3" data-testid="admin-sales-org-section">
        <h3 className="text-lg font-medium text-foreground">By Organization</h3>
        {loading ? (
          <div
            data-testid="org-breakdown-skeleton"
            className="h-48 w-full animate-pulse rounded-lg bg-muted"
          />
        ) : data ? (
          <OrgBreakdownTable variant="sales" data={data.org_breakdown} />
        ) : null}
      </div>
    </div>
  );
}
