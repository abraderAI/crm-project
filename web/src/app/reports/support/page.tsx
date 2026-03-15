"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertCircle } from "lucide-react";

import type { SupportMetrics } from "@/lib/reporting-types";
import { buildHeaders, buildUrl, parseResponse } from "@/lib/api-client";
import { getSupportExportUrl } from "@/lib/reporting-api";

import { MetricCard } from "@/components/reports/metric-card";
import { DateRangePicker } from "@/components/reports/date-range-picker";
import { AssigneeFilter } from "@/components/reports/assignee-filter";
import { ExportButton } from "@/components/reports/export-button";
import { StatusBreakdownChart } from "@/components/reports/support/status-breakdown-chart";
import { VolumeOverTimeChart } from "@/components/reports/support/volume-over-time-chart";
import { TicketsByAssigneeChart } from "@/components/reports/support/tickets-by-assignee-chart";
import { TicketsByPriorityChart } from "@/components/reports/support/tickets-by-priority-chart";

/** Default org ID used when no org is in context. */
const DEFAULT_ORG = "default";

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

/** Support dashboard page with filter state synced to URL search params. */
export default function SupportDashboardPage(): React.ReactNode {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { getToken } = useAuth();

  // Derive initial filter state from URL search params.
  const now = new Date();
  const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

  const [from, setFrom] = useState<Date>(parseDate(searchParams.get("from"), thirtyDaysAgo));
  const [to, setTo] = useState<Date>(parseDate(searchParams.get("to"), now));
  const [assignee, setAssignee] = useState<string | null>(searchParams.get("assignee") || null);

  const [data, setData] = useState<SupportMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  /** Sync current filter state to URL search params. */
  const syncParams = useCallback(
    (f: Date, t: Date, a: string | null) => {
      const params = new URLSearchParams();
      params.set("from", toDateString(f));
      params.set("to", toDateString(t));
      if (a) params.set("assignee", a);
      router.push(`/reports/support?${params.toString()}`);
    },
    [router],
  );

  /** Fetch support metrics from the API. */
  const fetchMetrics = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const token = await getToken();
      const queryParams: Record<string, string> = {
        from: toDateString(from),
        to: toDateString(to),
      };
      if (assignee) queryParams["assignee"] = assignee;

      const url = buildUrl(`/orgs/${DEFAULT_ORG}/reports/support`, queryParams);
      const response = await fetch(url, {
        method: "GET",
        headers: buildHeaders(token),
        cache: "no-store",
      });
      const metrics = await parseResponse<SupportMetrics>(response);
      setData(metrics);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load metrics";
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [getToken, from, to, assignee]);

  // Fetch on mount and when filters change.
  useEffect(() => {
    void fetchMetrics();
  }, [fetchMetrics]);

  function handleDateChange(range: { from: Date; to: Date }): void {
    setFrom(range.from);
    setTo(range.to);
    syncParams(range.from, range.to, assignee);
  }

  function handleAssigneeChange(userId: string | null): void {
    setAssignee(userId);
    syncParams(from, to, userId);
  }

  function handleStatusClick(status: string): void {
    router.push(`/crm?status=${encodeURIComponent(status)}`);
  }

  function handleAssigneeBarClick(userId: string): void {
    router.push(`/crm?assigned_to=${encodeURIComponent(userId)}`);
  }

  const exportUrl = getSupportExportUrl(DEFAULT_ORG, {
    from: toDateString(from),
    to: toDateString(to),
    assignee: assignee ?? undefined,
  });

  return (
    <div data-testid="support-dashboard" className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-foreground" data-testid="support-title">
          Support Tickets
        </h2>
        <ExportButton url={exportUrl} filename="support-report.csv" />
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3" data-testid="filter-bar">
        <DateRangePicker from={from} to={to} onChange={handleDateChange} />
        <AssigneeFilter orgId={DEFAULT_ORG} value={assignee} onChange={handleAssigneeChange} />
      </div>

      {/* Error alert */}
      {error && (
        <div
          data-testid="error-alert"
          className="flex items-center gap-2 rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          <p>{error}</p>
        </div>
      )}

      {/* KPI row */}
      <div className="grid gap-4 sm:grid-cols-3" data-testid="kpi-row">
        <MetricCard
          label="Avg Resolution Time"
          value={loading ? "–" : `${data?.avg_resolution_hours?.toFixed(1) ?? "–"} hrs`}
          loading={loading}
        />
        <MetricCard
          label="Avg First Response"
          value={loading ? "–" : `${data?.avg_first_response_hours?.toFixed(1) ?? "–"} hrs`}
          loading={loading}
        />
        <MetricCard
          label="Overdue Tickets"
          value={loading ? "–" : (data?.overdue_count ?? 0)}
          loading={loading}
          href="/crm?status=open&overdue=true"
        />
      </div>

      {/* Charts grid */}
      <div className="grid gap-6 md:grid-cols-2" data-testid="charts-grid">
        <div className="rounded-lg border border-border p-4" data-testid="chart-section-status">
          <h3 className="mb-2 text-sm font-medium text-foreground">Status Breakdown</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <StatusBreakdownChart data={data.status_breakdown} onSegmentClick={handleStatusClick} />
          ) : null}
        </div>

        <div className="rounded-lg border border-border p-4" data-testid="chart-section-volume">
          <h3 className="mb-2 text-sm font-medium text-foreground">Volume Over Time</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <VolumeOverTimeChart data={data.volume_over_time} />
          ) : null}
        </div>

        <div className="rounded-lg border border-border p-4" data-testid="chart-section-assignee">
          <h3 className="mb-2 text-sm font-medium text-foreground">Tickets by Assignee</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <TicketsByAssigneeChart
              data={data.tickets_by_assignee}
              onBarClick={handleAssigneeBarClick}
            />
          ) : null}
        </div>

        <div className="rounded-lg border border-border p-4" data-testid="chart-section-priority">
          <h3 className="mb-2 text-sm font-medium text-foreground">Tickets by Priority</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <TicketsByPriorityChart data={data.tickets_by_priority} />
          ) : null}
        </div>
      </div>
    </div>
  );
}
