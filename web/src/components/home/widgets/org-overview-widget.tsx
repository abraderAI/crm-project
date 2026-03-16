"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Building2, Users, CreditCard } from "lucide-react";
import { fetchOrgOverview, type OrgOverview } from "@/lib/org-api";

interface OrgOverviewWidgetProps {
  /** Auth token for API calls. */
  token: string;
  /** Org ID to fetch overview data for. */
  orgId: string;
  /** Whether the user is an org owner (shows billing status). */
  isOwner?: boolean;
}

/** Displays org name, member count, and plan status. Org owners see billing stub. */
export function OrgOverviewWidget({ token, orgId, isOwner }: OrgOverviewWidgetProps): ReactNode {
  const [data, setData] = useState<OrgOverview | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const overview = await fetchOrgOverview(token, orgId);
      if (!mountedRef.current) return;
      setData(overview);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load organization data.");
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
      <div data-testid="org-overview-loading" className="animate-pulse space-y-2">
        <div className="h-4 w-3/4 rounded bg-muted" />
        <div className="h-4 w-1/2 rounded bg-muted" />
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="org-overview-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (!data) {
    return (
      <p data-testid="org-overview-empty" className="text-sm text-muted-foreground">
        Organization data unavailable.
      </p>
    );
  }

  return (
    <div data-testid="org-overview-widget" className="space-y-3">
      <div className="flex items-center gap-2">
        <Building2 className="h-5 w-5 text-primary" />
        <span className="text-sm font-medium text-foreground" data-testid="org-overview-name">
          {data.name}
        </span>
      </div>

      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <Users className="h-3 w-3" />
          <span data-testid="org-overview-member-count">{data.member_count} members</span>
        </div>

        <div className="flex items-center gap-1">
          <CreditCard className="h-3 w-3" />
          <span data-testid="org-overview-plan">{data.billing_tier}</span>
        </div>
      </div>

      {isOwner && (
        <div
          data-testid="org-overview-billing-status"
          className="mt-2 rounded bg-accent/50 p-2 text-xs text-muted-foreground"
        >
          Payment status: <span className="font-medium">{data.plan_status}</span>
        </div>
      )}
    </div>
  );
}
