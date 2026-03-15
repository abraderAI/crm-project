"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertCircle } from "lucide-react";

import type { SalesMetrics } from "@/lib/reporting-types";
import { buildHeaders, buildUrl } from "@/lib/api-client";
import { getSalesExportUrl } from "@/lib/reporting-api";
import { MetricCard } from "@/components/reports/metric-card";
import { DateRangePicker } from "@/components/reports/date-range-picker";
import { AssigneeFilter } from "@/components/reports/assignee-filter";
import { ExportButton } from "@/components/reports/export-button";
import { PipelineFunnelChart } from "@/components/reports/sales/pipeline-funnel-chart";
import { LeadVelocityChart } from "@/components/reports/sales/lead-velocity-chart";
import { LeadsByAssigneeChart } from "@/components/reports/sales/leads-by-assignee-chart";
import { ScoreDistributionChart } from "@/components/reports/sales/score-distribution-chart";
import { StageConversionChart } from "@/components/reports/sales/stage-conversion-chart";
import { TimeInStageChart } from "@/components/reports/sales/time-in-stage-chart";

/** Default org ID placeholder. */
const DEFAULT_ORG_ID = "default";

/** Format a Date to YYYY-MM-DD. */
function toDateString(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

/** Default "from" date: 30 days ago. */
function defaultFrom(): Date {
  const d = new Date();
  d.setDate(d.getDate() - 30);
  return d;
}

/** Parse a YYYY-MM-DD string to a Date. Returns fallback if invalid. */
function parseDate(value: string | null, fallback: Date): Date {
  if (!value) return fallback;
  const d = new Date(value + "T00:00:00");
  return isNaN(d.getTime()) ? fallback : d;
}

/** Skeleton block for chart sections. */
function ChartSkeleton(): React.ReactNode {
  return (
    <div data-testid="chart-skeleton" className="h-72 w-full animate-pulse rounded-lg bg-muted" />
  );
}

/** Sales Pipeline dashboard page. */
export default function SalesPage(): React.ReactNode {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { getToken, orgId: clerkOrgId } = useAuth();

  const orgId = clerkOrgId ?? DEFAULT_ORG_ID;

  // Parse filters from URL search params.
  const [from, setFrom] = useState<Date>(() => parseDate(searchParams.get("from"), defaultFrom()));
  const [to, setTo] = useState<Date>(() => parseDate(searchParams.get("to"), new Date()));
  const [assignee, setAssignee] = useState<string | null>(searchParams.get("assignee") || null);

  // Data state.
  const [data, setData] = useState<SalesMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [token, setToken] = useState<string | null>(null);

  // Acquire Clerk token.
  useEffect(() => {
    let active = true;
    getToken().then((t) => {
      if (active) setToken(t);
    });
    return () => {
      active = false;
    };
  }, [getToken]);

  // Sync filter state to URL search params.
  useEffect(() => {
    const params = new URLSearchParams();
    params.set("from", toDateString(from));
    params.set("to", toDateString(to));
    if (assignee) params.set("assignee", assignee);
    const qs = params.toString();
    router.replace(`/reports/sales?${qs}`, { scroll: false });
  }, [from, to, assignee, router]);

  // Fetch sales metrics.
  useEffect(() => {
    let active = true;

    async function fetchData(): Promise<void> {
      setLoading(true);
      setError(null);

      try {
        const queryParams: Record<string, string> = {
          from: toDateString(from),
          to: toDateString(to),
        };
        if (assignee) queryParams["assignee"] = assignee;

        const url = buildUrl(`/orgs/${orgId}/reports/sales`, queryParams);
        const response = await fetch(url, {
          method: "GET",
          headers: buildHeaders(token),
        });

        if (!response.ok) {
          throw new Error(`Failed to load sales data: ${response.status}`);
        }

        const result = (await response.json()) as SalesMetrics;
        if (active) setData(result);
      } catch (err) {
        if (active) {
          setError(err instanceof Error ? err.message : "Failed to load sales data");
        }
      } finally {
        if (active) setLoading(false);
      }
    }

    void fetchData();
    return () => {
      active = false;
    };
  }, [from, to, assignee, orgId, token]);

  // Build export URL.
  const exportUrl = useMemo(
    () =>
      getSalesExportUrl(orgId, {
        from: toDateString(from),
        to: toDateString(to),
        assignee: assignee ?? undefined,
      }),
    [orgId, from, to, assignee],
  );

  // Date range change handler.
  const handleDateChange = useCallback((range: { from: Date; to: Date }) => {
    setFrom(range.from);
    setTo(range.to);
  }, []);

  // Assignee change handler.
  const handleAssigneeChange = useCallback((userId: string | null) => {
    setAssignee(userId);
  }, []);

  // Pipeline bar click → navigate to CRM list filtered by stage.
  const handlePipelineClick = useCallback(
    (stage: string) => {
      router.push(`/crm?stage=${encodeURIComponent(stage)}`);
    },
    [router],
  );

  // Assignee bar click → navigate to CRM list filtered by assignee.
  const handleAssigneeBarClick = useCallback(
    (userId: string) => {
      router.push(`/crm?assignee=${encodeURIComponent(userId)}`);
    },
    [router],
  );

  return (
    <div data-testid="sales-page" className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-foreground" data-testid="sales-page-title">
          Sales Pipeline
        </h2>
        <ExportButton url={exportUrl} filename="sales-report.csv" />
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3" data-testid="sales-filter-bar">
        <DateRangePicker from={from} to={to} onChange={handleDateChange} />
        <AssigneeFilter
          orgId={orgId}
          value={assignee}
          onChange={handleAssigneeChange}
          token={token}
        />
      </div>

      {/* Error state */}
      {error && (
        <div
          data-testid="sales-error"
          className="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* KPI row */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3" data-testid="sales-kpi-row">
        <MetricCard
          label="Win Rate"
          value={data ? `${(data.win_rate * 100).toFixed(1)}%` : "–"}
          loading={loading}
        />
        <MetricCard
          label="Loss Rate"
          value={data ? `${(data.loss_rate * 100).toFixed(1)}%` : "–"}
          loading={loading}
        />
        <MetricCard
          label="Avg Deal Value"
          value={data?.avg_deal_value ? `$${data.avg_deal_value.toLocaleString()}` : "–"}
          loading={loading}
        />
      </div>

      {/* Charts grid */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2" data-testid="sales-charts-grid">
        {/* Pipeline Funnel */}
        <div
          className="rounded-lg border border-border bg-background p-4"
          data-testid="chart-section-pipeline-funnel"
        >
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Pipeline Funnel</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <PipelineFunnelChart data={data.pipeline_funnel} onBarClick={handlePipelineClick} />
          ) : null}
        </div>

        {/* Lead Velocity */}
        <div
          className="rounded-lg border border-border bg-background p-4"
          data-testid="chart-section-lead-velocity"
        >
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Lead Velocity</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <LeadVelocityChart data={data.lead_velocity} />
          ) : null}
        </div>

        {/* Leads by Assignee */}
        <div
          className="rounded-lg border border-border bg-background p-4"
          data-testid="chart-section-leads-by-assignee"
        >
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Leads by Assignee</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <LeadsByAssigneeChart
              data={data.leads_by_assignee}
              onBarClick={handleAssigneeBarClick}
            />
          ) : null}
        </div>

        {/* Score Distribution */}
        <div
          className="rounded-lg border border-border bg-background p-4"
          data-testid="chart-section-score-distribution"
        >
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Score Distribution</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <ScoreDistributionChart data={data.score_distribution} />
          ) : null}
        </div>

        {/* Stage Conversion (full width) */}
        <div
          className="rounded-lg border border-border bg-background p-4 md:col-span-2"
          data-testid="chart-section-stage-conversion"
        >
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Stage Conversion</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <StageConversionChart data={data.stage_conversion_rates} />
          ) : null}
        </div>

        {/* Time in Stage (full width) */}
        <div
          className="rounded-lg border border-border bg-background p-4 md:col-span-2"
          data-testid="chart-section-time-in-stage"
        >
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Time in Stage</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <TimeInStageChart data={data.avg_time_in_stage} />
          ) : null}
        </div>
      </div>
    </div>
  );
}
