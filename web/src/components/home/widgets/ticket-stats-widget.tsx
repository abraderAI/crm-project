"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { fetchTicketStats, type TicketStats } from "@/lib/widget-api";

export interface TicketStatsWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays open/pending/resolved ticket counts and average response time. */
export function TicketStatsWidget({ token }: TicketStatsWidgetProps): ReactNode {
  const [data, setData] = useState<TicketStats | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchTicketStats(token);
      if (mountedRef.current) {
        setData(result);
      }
    } catch {
      if (mountedRef.current) {
        setError("Failed to load ticket stats");
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
      <div data-testid="ticket-stats-loading" className="text-sm text-muted-foreground">
        Loading stats…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="ticket-stats-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (!data) {
    return null;
  }

  return (
    <div data-testid="ticket-stats-content" className="space-y-3">
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <div className="text-lg font-bold text-red-600" data-testid="stat-open">
            {data.open}
          </div>
          <div className="text-xs text-muted-foreground">Open</div>
        </div>
        <div>
          <div className="text-lg font-bold text-amber-600" data-testid="stat-pending">
            {data.pending}
          </div>
          <div className="text-xs text-muted-foreground">Pending</div>
        </div>
        <div>
          <div className="text-lg font-bold text-green-600" data-testid="stat-resolved">
            {data.resolved}
          </div>
          <div className="text-xs text-muted-foreground">Resolved</div>
        </div>
      </div>
      <div className="text-center text-xs text-muted-foreground">
        Avg response time:{" "}
        <span data-testid="stat-avg-response" className="font-medium text-foreground">
          {data.avg_response_time}
        </span>
      </div>
    </div>
  );
}
