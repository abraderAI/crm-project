"use client";

import Link from "next/link";
import type { SecurityLogEntry } from "@/lib/api-types";

export interface SecurityLogProps {
  /** Security log entries to display. */
  entries: SecurityLogEntry[];
  /** Whether data is loading. */
  loading?: boolean;
  /** Whether more pages are available. */
  hasMore?: boolean;
  /** Called when the user requests the next page. */
  onLoadMore?: () => void;
}

/** Format an ISO timestamp for display. */
function formatTimestamp(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

/** Reusable paginated security log table. */
export function SecurityLog({
  entries,
  loading = false,
  hasMore = false,
  onLoadMore,
}: SecurityLogProps): React.ReactNode {
  return (
    <div data-testid="security-log" className="flex flex-col gap-4">
      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="security-log-loading"
        >
          Loading security events…
        </div>
      )}

      {!loading && entries.length === 0 && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="security-log-empty"
        >
          No entries found.
        </div>
      )}

      {!loading && entries.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-left text-sm" data-testid="security-log-table">
            <thead className="border-b border-border bg-muted/40">
              <tr>
                <th className="px-4 py-2 font-medium text-muted-foreground">User ID</th>
                <th className="px-4 py-2 font-medium text-muted-foreground">IP Address</th>
                <th className="px-4 py-2 font-medium text-muted-foreground">User Agent</th>
                <th className="px-4 py-2 font-medium text-muted-foreground">Timestamp</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {entries.map((entry) => (
                <tr key={entry.id} data-testid={`security-row-${entry.id}`}>
                  <td className="px-4 py-2">
                    <Link
                      href={`/admin/users/${entry.user_id}`}
                      className="text-primary underline-offset-4 hover:underline"
                    >
                      <span data-testid={`security-user-${entry.id}`}>{entry.user_id}</span>
                    </Link>
                  </td>
                  <td
                    className="px-4 py-2 font-mono text-xs"
                    data-testid={`security-ip-${entry.id}`}
                  >
                    {entry.ip_address}
                  </td>
                  <td
                    className="max-w-xs truncate px-4 py-2 text-xs text-muted-foreground"
                    data-testid={`security-ua-${entry.id}`}
                    title={entry.user_agent}
                  >
                    {entry.user_agent}
                  </td>
                  <td
                    className="whitespace-nowrap px-4 py-2 text-xs text-muted-foreground"
                    data-testid={`security-time-${entry.id}`}
                  >
                    {formatTimestamp(entry.timestamp)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {hasMore && !loading && (
        <div className="flex justify-center">
          <button
            onClick={onLoadMore}
            data-testid="security-log-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}
