"use client";

import { useCallback, useEffect, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { Activity } from "lucide-react";
import { cn } from "@/lib/utils";
import { buildHeaders, buildUrl } from "@/lib/api-client";
import type { ApiUsageEntry, ApiUsagePeriod, ApiUsageResponse } from "@/lib/api-types";

const PERIODS: ApiUsagePeriod[] = ["24h", "7d", "30d"];

const PERIOD_LABELS: Record<ApiUsagePeriod, string> = {
  "24h": "24 hours",
  "7d": "7 days",
  "30d": "30 days",
};

/** Method badge color based on HTTP method. */
function methodColor(method: string): string {
  switch (method) {
    case "GET":
      return "bg-blue-100 text-blue-800";
    case "POST":
      return "bg-green-100 text-green-800";
    case "PUT":
    case "PATCH":
      return "bg-yellow-100 text-yellow-800";
    case "DELETE":
      return "bg-red-100 text-red-800";
    default:
      return "bg-muted text-muted-foreground";
  }
}

/** API usage stats table with time-window toggle. */
export function ApiUsageTable(): React.ReactNode {
  const { getToken } = useAuth();
  const [period, setPeriod] = useState<ApiUsagePeriod>("24h");
  const [entries, setEntries] = useState<ApiUsageEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(
    async (p: ApiUsagePeriod) => {
      setLoading(true);
      setError(null);
      try {
        const token = await getToken();
        const url = buildUrl("/admin/api-usage", { period: p });
        const response = await fetch(url, {
          method: "GET",
          headers: buildHeaders(token),
        });
        if (!response.ok) {
          throw new Error(`Failed to fetch API usage: ${response.status}`);
        }
        const data = (await response.json()) as ApiUsageResponse;
        setEntries(data.data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    },
    [getToken],
  );

  useEffect(() => {
    void fetchData(period);
  }, [fetchData, period]);

  const handlePeriodChange = (newPeriod: ApiUsagePeriod): void => {
    if (newPeriod === period) return;
    setPeriod(newPeriod);
  };

  if (loading) {
    return (
      <div
        className="py-8 text-center text-sm text-muted-foreground"
        data-testid="api-usage-loading"
      >
        Loading API usage stats…
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-8 text-center text-sm text-destructive" data-testid="api-usage-error">
        {error}
      </div>
    );
  }

  return (
    <div data-testid="api-usage-table" className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="h-5 w-5 text-muted-foreground" />
          <h2 className="text-lg font-semibold text-foreground">API Usage</h2>
        </div>

        <div
          className="flex gap-1 rounded-lg border border-border p-1"
          role="group"
          aria-label="Time period"
        >
          {PERIODS.map((p) => (
            <button
              key={p}
              data-testid={`period-btn-${p}`}
              aria-pressed={period === p}
              onClick={() => handlePeriodChange(p)}
              className={cn(
                "rounded-md px-3 py-1 text-xs font-medium transition-colors",
                period === p
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground",
              )}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      <p className="text-sm text-muted-foreground">
        Showing request counts for the last {PERIOD_LABELS[period]}.
      </p>

      {entries.length === 0 ? (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="api-usage-empty"
        >
          No API usage data for this period.
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/40">
                <th className="px-4 py-2 text-left font-medium text-muted-foreground">Endpoint</th>
                <th className="px-4 py-2 text-left font-medium text-muted-foreground">Method</th>
                <th className="px-4 py-2 text-right font-medium text-muted-foreground">Requests</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {entries.map((entry, idx) => (
                <tr key={`${entry.endpoint}-${entry.method}`} data-testid={`api-usage-row-${idx}`}>
                  <td className="px-4 py-2 font-mono text-xs text-foreground">{entry.endpoint}</td>
                  <td className="px-4 py-2">
                    <span
                      className={cn(
                        "rounded-full px-2 py-0.5 text-xs font-medium",
                        methodColor(entry.method),
                      )}
                      data-testid={`api-usage-method-${idx}`}
                    >
                      {entry.method}
                    </span>
                  </td>
                  <td className="px-4 py-2 text-right font-medium text-foreground">
                    {entry.count.toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
