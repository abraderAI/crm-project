"use client";

import { MetricCard } from "@/components/reports/metric-card";
import type { PlatformStats as PlatformStatsType } from "@/lib/api-types";

export interface PlatformStatsProps {
  /** Platform-wide statistics from GET /v1/admin/stats. */
  stats: PlatformStatsType;
  /** When true, all metric cards show loading skeletons. */
  loading?: boolean;
}

/** Format bytes into a human-readable string (B, KB, MB, GB). */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / 1024 ** i).toFixed(1)} ${units[i]}`;
}

/** Format uptime percentage for display. */
export function formatUptime(pct: number): string {
  if (pct === 0) return "0%";
  // Drop trailing zeroes: 100.00 → "100", 99.98 → "99.98", 95.5 → "95.5"
  const formatted = parseFloat(pct.toFixed(2)).toString();
  return `${formatted}%`;
}

/** KPI row showing Total Orgs, Total Users, Total Threads, DB Size, and API Uptime. */
export function PlatformStats({ stats, loading = false }: PlatformStatsProps): React.ReactNode {
  return (
    <div data-testid="platform-stats" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
      <div data-testid="platform-stats-orgs">
        <MetricCard
          label="Total Orgs"
          value={stats.orgs.total.toLocaleString()}
          loading={loading}
        />
      </div>
      <div data-testid="platform-stats-users">
        <MetricCard
          label="Total Users"
          value={stats.users.total.toLocaleString()}
          loading={loading}
        />
      </div>
      <div data-testid="platform-stats-threads">
        <MetricCard
          label="Total Threads"
          value={stats.threads.total.toLocaleString()}
          loading={loading}
        />
      </div>
      <div data-testid="platform-stats-db-size">
        <MetricCard label="DB Size" value={formatBytes(stats.db_size_bytes)} loading={loading} />
      </div>
      <div data-testid="platform-stats-api-uptime">
        <MetricCard
          label="API Uptime"
          value={formatUptime(stats.api_uptime_pct)}
          loading={loading}
        />
      </div>
    </div>
  );
}
