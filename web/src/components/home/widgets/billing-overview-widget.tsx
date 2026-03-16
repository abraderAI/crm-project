"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { fetchBillingOverview, type BillingOverview } from "@/lib/widget-api";

export interface BillingOverviewWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays paying org count, MRR stub, and recent payments. Visible to DEFT finance only. */
export function BillingOverviewWidget({ token }: BillingOverviewWidgetProps): ReactNode {
  const [data, setData] = useState<BillingOverview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchBillingOverview(token);
      if (mountedRef.current) {
        setData(result);
      }
    } catch {
      if (mountedRef.current) {
        setError("Failed to load billing overview");
      }
    } finally {
      if (mountedRef.current) {
        setIsLoading(false);
      }
    }
  }, [token]);

  useEffect(() => {
    mountedRef.current = true;
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, [load]);

  if (isLoading) {
    return (
      <div data-testid="billing-overview-loading" className="text-sm text-muted-foreground">
        Loading billing…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="billing-overview-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const formatCurrency = (value: number): string =>
    new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 0,
    }).format(value);

  return (
    <div data-testid="billing-overview-content" className="space-y-3">
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <div className="text-lg font-bold text-foreground" data-testid="billing-orgs">
            {data.paying_org_count}
          </div>
          <div className="text-xs text-muted-foreground">Paying Orgs</div>
        </div>
        <div>
          <div className="text-lg font-bold text-foreground" data-testid="billing-mrr">
            {formatCurrency(data.mrr)}
          </div>
          <div className="text-xs text-muted-foreground">MRR</div>
        </div>
        <div>
          <div className="text-lg font-bold text-foreground" data-testid="billing-payments">
            {data.recent_payments}
          </div>
          <div className="text-xs text-muted-foreground">Recent Payments</div>
        </div>
      </div>
      {data.mrr === 0 && (
        <div className="text-center text-xs text-muted-foreground" data-testid="billing-stub-note">
          Revenue data is a placeholder — billing integration pending.
        </div>
      )}
    </div>
  );
}
