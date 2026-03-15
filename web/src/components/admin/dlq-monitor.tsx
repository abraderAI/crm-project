"use client";

import { useEffect, useRef, useState } from "react";
import { RefreshCw, RotateCcw, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ChannelType, DeadLetterEvent, DLQStatus } from "@/lib/api-types";

const STATUS_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "all", label: "All" },
  { value: "failed", label: "Failed" },
  { value: "retrying", label: "Retrying" },
  { value: "resolved", label: "Resolved" },
  { value: "dismissed", label: "Dismissed" },
];

export interface DLQMonitorProps {
  /** Org identifier. */
  org: string;
  /** Channel type this monitor is for. */
  channelType: ChannelType;
  /** Dead-letter queue events. */
  events: DeadLetterEvent[];
  /** Whether events are loading. */
  loading?: boolean;
  /** Called when retrying an event. */
  onRetry: (eventId: string) => void;
  /** Called when dismissing an event. */
  onDismiss: (eventId: string) => void;
  /** Called to refresh the event list. */
  onRefresh: () => void;
}

/** Format a date string for display in the table. */
function formatTime(dateStr: string | undefined | null): string {
  if (!dateStr) return "—";
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return "—";
    return d.toLocaleString("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "—";
  }
}

/** Status badge colour. */
function statusColor(status: DLQStatus): string {
  switch (status) {
    case "failed":
      return "bg-red-100 text-red-800";
    case "retrying":
      return "bg-yellow-100 text-yellow-800";
    case "resolved":
      return "bg-green-100 text-green-800";
    case "dismissed":
      return "bg-gray-100 text-gray-600";
    default:
      return "bg-muted text-muted-foreground";
  }
}

/** Truncate a string to a max length. */
function truncate(str: string, max: number): string {
  if (str.length <= max) return str;
  return str.slice(0, max) + "…";
}

/** Dead-letter queue monitor with filtering, retry/dismiss actions, and auto-refresh. */
export function DLQMonitor({
  events,
  loading = false,
  onRetry,
  onDismiss,
  onRefresh,
}: DLQMonitorProps): React.ReactNode {
  const [statusFilter, setStatusFilter] = useState("all");
  const [confirmDismiss, setConfirmDismiss] = useState<string | null>(null);
  const [lastRefreshed, setLastRefreshed] = useState<Date>(new Date());
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Auto-refresh every 30s.
  useEffect(() => {
    intervalRef.current = setInterval(() => {
      onRefresh();
      setLastRefreshed(new Date());
    }, 30_000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [onRefresh]);

  const filteredEvents =
    statusFilter === "all" ? events : events.filter((e) => e.status === statusFilter);

  const handleRefreshClick = (): void => {
    onRefresh();
    setLastRefreshed(new Date());
  };

  const handleDismissConfirm = (eventId: string): void => {
    onDismiss(eventId);
    setConfirmDismiss(null);
  };

  return (
    <div data-testid="dlq-monitor" className="flex flex-col gap-4" id="dlq">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">Dead Letter Queue</h2>
        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground" data-testid="dlq-last-refreshed">
            Last refreshed: {formatTime(lastRefreshed.toISOString())}
          </span>
          <button
            onClick={handleRefreshClick}
            data-testid="dlq-refresh-btn"
            className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent"
          >
            <RefreshCw className="h-3 w-3" />
            Refresh
          </button>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <label htmlFor="dlq-status-filter" className="text-xs text-muted-foreground">
          Status:
        </label>
        <select
          id="dlq-status-filter"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          data-testid="dlq-status-filter"
          className="rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground"
        >
          {STATUS_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      {loading && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="dlq-loading">
          Loading events...
        </div>
      )}

      {!loading && filteredEvents.length === 0 && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="dlq-empty">
          No dead-letter events.
        </div>
      )}

      {!loading && filteredEvents.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-border" data-testid="dlq-table">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/50">
                <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                  Created
                </th>
                <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                  Error Message
                </th>
                <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                  Attempts
                </th>
                <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                  Last Attempt
                </th>
                <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                  Status
                </th>
                <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filteredEvents.map((event) => {
                const isTerminal = event.status === "resolved" || event.status === "dismissed";
                return (
                  <tr key={event.id} data-testid={`dlq-row-${event.id}`}>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-foreground">
                      {formatTime(event.created_at)}
                    </td>
                    <td
                      className="max-w-[200px] truncate px-3 py-2 text-xs text-foreground"
                      title={event.error_message}
                      data-testid={`dlq-error-${event.id}`}
                    >
                      {truncate(event.error_message, 80)}
                    </td>
                    <td className="px-3 py-2 text-xs text-foreground">{event.attempts}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-foreground">
                      {formatTime(event.last_attempt_at)}
                    </td>
                    <td className="px-3 py-2">
                      <span
                        className={cn(
                          "inline-block rounded-full px-2 py-0.5 text-xs font-medium",
                          statusColor(event.status),
                        )}
                        data-testid={`dlq-status-${event.id}`}
                      >
                        {event.status}
                      </span>
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => onRetry(event.id)}
                          disabled={isTerminal}
                          data-testid={`dlq-retry-${event.id}`}
                          className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-foreground hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                          <RotateCcw className="h-3 w-3" />
                          Retry
                        </button>
                        {confirmDismiss === event.id ? (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleDismissConfirm(event.id)}
                              data-testid={`dlq-dismiss-confirm-${event.id}`}
                              className="rounded bg-destructive px-2 py-1 text-xs font-medium text-destructive-foreground"
                            >
                              Confirm
                            </button>
                            <button
                              onClick={() => setConfirmDismiss(null)}
                              data-testid={`dlq-dismiss-cancel-${event.id}`}
                              className="rounded border border-border px-2 py-1 text-xs font-medium text-foreground"
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <button
                            onClick={() => setConfirmDismiss(event.id)}
                            disabled={isTerminal}
                            data-testid={`dlq-dismiss-${event.id}`}
                            className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-foreground hover:bg-destructive/10 hover:text-destructive disabled:opacity-40 disabled:cursor-not-allowed"
                          >
                            <XCircle className="h-3 w-3" />
                            Dismiss
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
