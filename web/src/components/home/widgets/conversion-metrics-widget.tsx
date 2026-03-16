"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { fetchConversionMetrics, type ConversionMetrics } from "@/lib/widget-api";

export interface ConversionMetricsWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays Tier 1→2→3 conversion funnel counts. */
export function ConversionMetricsWidget({ token }: ConversionMetricsWidgetProps): ReactNode {
  const [data, setData] = useState<ConversionMetrics | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchConversionMetrics(token);
      if (mountedRef.current) {
        setData(result);
      }
    } catch {
      if (mountedRef.current) {
        setError("Failed to load conversion metrics");
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
      <div data-testid="conversion-metrics-loading" className="text-sm text-muted-foreground">
        Loading metrics…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="conversion-metrics-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const regRate =
    data.anonymous_sessions > 0
      ? ((data.registrations / data.anonymous_sessions) * 100).toFixed(1)
      : "0.0";
  const convRate =
    data.registrations > 0 ? ((data.conversions / data.registrations) * 100).toFixed(1) : "0.0";

  return (
    <div data-testid="conversion-metrics-content" className="space-y-3">
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <div className="text-lg font-bold text-foreground" data-testid="metric-anonymous">
            {data.anonymous_sessions}
          </div>
          <div className="text-xs text-muted-foreground">Anonymous</div>
        </div>
        <div>
          <div className="text-lg font-bold text-foreground" data-testid="metric-registrations">
            {data.registrations}
          </div>
          <div className="text-xs text-muted-foreground">Registered</div>
        </div>
        <div>
          <div className="text-lg font-bold text-foreground" data-testid="metric-conversions">
            {data.conversions}
          </div>
          <div className="text-xs text-muted-foreground">Converted</div>
        </div>
      </div>
      <div className="flex justify-between text-xs text-muted-foreground">
        <span data-testid="rate-registration">Registration: {regRate}%</span>
        <span data-testid="rate-conversion">Conversion: {convRate}%</span>
      </div>
    </div>
  );
}
