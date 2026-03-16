"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { fetchSystemHealth, type SystemHealth } from "@/lib/widget-api";

const STATUS_ICON: Record<string, string> = {
  healthy: "text-green-500",
  degraded: "text-amber-500",
  down: "text-red-500",
  unknown: "text-gray-400",
};

export interface SystemHealthWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays system health: API uptime, DB status, and channel health. */
export function SystemHealthWidget({ token }: SystemHealthWidgetProps): ReactNode {
  const [data, setData] = useState<SystemHealth | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchSystemHealth(token);
      if (mountedRef.current) {
        setData(result);
      }
    } catch {
      if (mountedRef.current) {
        setError("Failed to load system health");
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
      <div data-testid="system-health-loading" className="text-sm text-muted-foreground">
        Loading health…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="system-health-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const statusDot = (status: string): string =>
    STATUS_ICON[status] ?? STATUS_ICON["unknown"] ?? "text-gray-400";

  return (
    <div data-testid="system-health-content" className="space-y-3">
      <div className="space-y-1.5">
        <div className="flex items-center gap-2" data-testid="health-api">
          <span className={`inline-block h-2 w-2 rounded-full ${statusDot(data.api_status)}`} />
          <span className="text-xs text-foreground">API</span>
          <span className="ml-auto text-xs text-muted-foreground">{data.api_status}</span>
        </div>
        <div className="flex items-center gap-2" data-testid="health-db">
          <span className={`inline-block h-2 w-2 rounded-full ${statusDot(data.db_status)}`} />
          <span className="text-xs text-foreground">Database</span>
          <span className="ml-auto text-xs text-muted-foreground">{data.db_status}</span>
        </div>
        {Object.entries(data.channel_health).map(([channel, status]) => (
          <div key={channel} className="flex items-center gap-2" data-testid={`health-${channel}`}>
            <span className={`inline-block h-2 w-2 rounded-full ${statusDot(status)}`} />
            <span className="text-xs text-foreground capitalize">{channel}</span>
            <span className="ml-auto text-xs text-muted-foreground">{status}</span>
          </div>
        ))}
      </div>
      <div className="text-center text-xs text-muted-foreground">
        Uptime:{" "}
        <span data-testid="health-uptime" className="font-medium">
          {data.uptime}
        </span>
      </div>
    </div>
  );
}
