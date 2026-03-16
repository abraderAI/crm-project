"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { BarChart3 } from "lucide-react";
import { fetchOrgSupportStats, type OrgSupportStats } from "@/lib/org-api";

interface OrgSupportDashboardWidgetProps {
  /** Auth token for API calls. */
  token: string;
  /** Org ID to fetch stats for. */
  orgId: string;
}

/** Displays ticket volume and open/closed counts for the org owner. */
export function OrgSupportDashboardWidget({
  token,
  orgId,
}: OrgSupportDashboardWidgetProps): ReactNode {
  const [stats, setStats] = useState<OrgSupportStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchOrgSupportStats(token, orgId);
      if (!mountedRef.current) return;
      setStats(result);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load support statistics.");
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [token, orgId]);

  useEffect(() => {
    mountedRef.current = true;
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, [load]);

  if (isLoading) {
    return (
      <div data-testid="org-support-dashboard-loading" className="animate-pulse space-y-2">
        <div className="h-4 w-3/4 rounded bg-muted" />
        <div className="h-8 rounded bg-muted" />
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="org-support-dashboard-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (!stats) {
    return (
      <p data-testid="org-support-dashboard-empty" className="text-sm text-muted-foreground">
        No support data available.
      </p>
    );
  }

  return (
    <div data-testid="org-support-dashboard-widget" className="space-y-3">
      <div className="flex items-center gap-2">
        <BarChart3 className="h-4 w-4 text-primary" />
        <span className="text-sm font-medium text-foreground">{stats.total} total tickets</span>
      </div>

      <div className="grid grid-cols-3 gap-2">
        <div data-testid="org-dashboard-open" className="rounded bg-yellow-50 p-2 text-center">
          <span className="block text-lg font-bold text-yellow-700">{stats.open}</span>
          <span className="text-xs text-yellow-600">Open</span>
        </div>
        <div data-testid="org-dashboard-pending" className="rounded bg-blue-50 p-2 text-center">
          <span className="block text-lg font-bold text-blue-700">{stats.pending}</span>
          <span className="text-xs text-blue-600">Pending</span>
        </div>
        <div data-testid="org-dashboard-resolved" className="rounded bg-green-50 p-2 text-center">
          <span className="block text-lg font-bold text-green-700">{stats.resolved}</span>
          <span className="text-xs text-green-600">Resolved</span>
        </div>
      </div>
    </div>
  );
}
