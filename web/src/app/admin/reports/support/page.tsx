"use client";
export const dynamic = "force-dynamic";

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertCircle } from "lucide-react";

import type { AdminSupportMetrics } from "@/lib/reporting-types";
import { buildHeaders, buildUrl, parseResponse } from "@/lib/api-client";
import { getAdminSupportExportUrl } from "@/lib/reporting-api";

import { MetricCard } from "@/components/reports/metric-card";
import { DateRangePicker } from "@/components/reports/date-range-picker";
import { ExportButton } from "@/components/reports/export-button";
import { StatusBreakdownChart } from "@/components/reports/support/status-breakdown-chart";
import { VolumeOverTimeChart } from "@/components/reports/support/volume-over-time-chart";
import { TicketsByPriorityChart } from "@/components/reports/support/tickets-by-priority-chart";
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

/** Platform admin support dashboard page inner component. */
function AdminSupportPageInner(): React.ReactNode {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { getToken } = useAuth();

  const now = new Date();
  const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

  const [from, setFrom] = useState<Date>(parseDate(searchParams.get("from"), thirtyDaysAgo));
  const [to, setTo] = useState<Date>(parseDate(searchParams.get("to"), now));

  const [data, setData] = useState<AdminSupportMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  /** Fetch admin support metrics from the API. */
  const fetchMetrics = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const token = await getToken();
      const queryParams: Record<string, string> = {
        from: toDateString(from),
        to: toDateString(to),
      };

      const url = buildUrl("/admin/reports/support", queryParams);
      const response = await fetch(url, {
        method: "GET",
        headers: buildHeaders(token),
        cache: "no-store",
      });
      const metrics = await parseResponse<AdminSupportMetrics>(response);
      setData(metrics);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load metrics";
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
    router.replace(`/admin/reports/support?${params.toString()}`, { scroll: false });
  }, [from, to, router]);

  function handleDateChange(range: { from: Date; to: Date }): void {
    setFrom(range.from);
    setTo(range.to);
  }

  const exportUrl = useMemo(
    () =>
      getAdminSupportExportUrl({
        from: toDateString(from),
        to: toDateString(to),
      }),
    [from, to],
  );

  return (
    <div data-testid="admin-support-dashboard" className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-foreground" data-testid="admin-support-title">
          Platform Support Overview
        </h2>
        <ExportButton url={exportUrl} filename="admin-support-report.csv" />
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3" data-testid="admin-support-filter-bar">
        <DateRangePicker from={from} to={to} onChange={handleDateChange} />
      </div>

      {/* Error alert */}
      {error && (
        <div
          data-testid="admin-support-error"
          className="flex items-center gap-2 rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          <p>{error}</p>
        </div>
      )}

      {/* KPI row */}
      <div className="grid gap-4 sm:grid-cols-3" data-testid="admin-support-kpi-row">
        <MetricCard
          label="Total Open Tickets"
          value={loading ? "–" : (data?.status_breakdown.open ?? 0)}
          loading={loading}
        />
        <MetricCard
          label="Platform Overdue"
          value={loading ? "–" : (data?.overdue_count ?? 0)}
          loading={loading}
        />
        <MetricCard
          label="Avg Resolution"
          value={loading ? "–" : `${data?.avg_resolution_hours?.toFixed(1) ?? "–"} hrs`}
          loading={loading}
        />
      </div>

      {/* Charts grid */}
      <div className="grid gap-6 md:grid-cols-2" data-testid="admin-support-charts-grid">
        <div className="rounded-lg border border-border p-4" data-testid="chart-section-status">
          <h3 className="mb-2 text-sm font-medium text-foreground">Status Breakdown</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <StatusBreakdownChart data={data.status_breakdown} />
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

        <div className="rounded-lg border border-border p-4" data-testid="chart-section-priority">
          <h3 className="mb-2 text-sm font-medium text-foreground">Tickets by Priority</h3>
          {loading ? (
            <ChartSkeleton />
          ) : data ? (
            <TicketsByPriorityChart data={data.tickets_by_priority} />
          ) : null}
        </div>
      </div>

      {/* Per-org breakdown */}
      <div className="flex flex-col gap-3" data-testid="admin-support-org-section">
        <h3 className="text-lg font-medium text-foreground">By Organization</h3>
        {loading ? (
          <div
            data-testid="org-breakdown-skeleton"
            className="h-48 w-full animate-pulse rounded-lg bg-muted"
          />
        ) : data ? (
          <OrgBreakdownTable variant="support" data={data.org_breakdown} />
        ) : null}
      </div>
    </div>
  );
}

export default function AdminSupportPage(): React.ReactNode {
  return (
    <Suspense>
      <AdminSupportPageInner />
    </Suspense>
  );
}
