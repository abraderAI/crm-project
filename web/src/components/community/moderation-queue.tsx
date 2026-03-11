"use client";

import { Shield, Check, X } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Flag } from "@/lib/api-types";

export interface ModerationQueueProps {
  /** List of flags to display. */
  flags: Flag[];
  /** Whether data is loading. */
  loading?: boolean;
  /** Called when a flag is resolved. */
  onResolve: (flagId: string, note: string) => void;
  /** Called when a flag is dismissed. */
  onDismiss: (flagId: string) => void;
  /** Active filter status. */
  statusFilter?: Flag["status"];
  /** Called when filter changes. */
  onFilterChange?: (status: Flag["status"] | "all") => void;
}

/** Format a date string for display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  } catch {
    return dateStr;
  }
}

/** Status badge color mapping. */
function statusColor(status: Flag["status"]): string {
  switch (status) {
    case "pending":
      return "bg-yellow-100 text-yellow-800";
    case "resolved":
      return "bg-green-100 text-green-800";
    case "dismissed":
      return "bg-muted text-muted-foreground";
  }
}

const FILTER_OPTIONS: { value: Flag["status"] | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "resolved", label: "Resolved" },
  { value: "dismissed", label: "Dismissed" },
];

/** Moderation queue displaying flags with resolve/dismiss actions. */
export function ModerationQueue({
  flags,
  loading = false,
  onResolve,
  onDismiss,
  statusFilter,
  onFilterChange,
}: ModerationQueueProps): React.ReactNode {
  return (
    <div data-testid="moderation-queue" className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-muted-foreground" data-testid="moderation-icon" />
          <h2 className="text-lg font-semibold text-foreground">Moderation Queue</h2>
          <span
            className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
            data-testid="flag-count"
          >
            {flags.length}
          </span>
        </div>
        {onFilterChange && (
          <div className="flex rounded-md border border-border" data-testid="moderation-filters">
            {FILTER_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                onClick={() => onFilterChange(opt.value)}
                data-testid={`filter-${opt.value}`}
                className={cn(
                  "px-3 py-1 text-xs font-medium transition-colors first:rounded-l-md last:rounded-r-md",
                  statusFilter === opt.value || (!statusFilter && opt.value === "all")
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-accent",
                )}
              >
                {opt.label}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Loading */}
      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="moderation-loading"
        >
          Loading flags...
        </div>
      )}

      {/* Empty state */}
      {!loading && flags.length === 0 && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="moderation-empty"
        >
          No flags to review.
        </div>
      )}

      {/* Flag list */}
      {!loading && flags.length > 0 && (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="flag-list"
        >
          {flags.map((flag) => (
            <div
              key={flag.id}
              className="flex items-start gap-3 px-4 py-3"
              data-testid={`flag-item-${flag.id}`}
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span
                    className={cn(
                      "rounded-full px-2 py-0.5 text-xs font-medium",
                      statusColor(flag.status),
                    )}
                    data-testid={`flag-status-${flag.id}`}
                  >
                    {flag.status}
                  </span>
                  <span
                    className="text-xs text-muted-foreground"
                    data-testid={`flag-date-${flag.id}`}
                  >
                    {formatDate(flag.created_at)}
                  </span>
                </div>
                <p className="mt-1 text-sm text-foreground" data-testid={`flag-reason-${flag.id}`}>
                  {flag.reason}
                </p>
                <span
                  className="text-xs text-muted-foreground"
                  data-testid={`flag-reporter-${flag.id}`}
                >
                  Reporter: {flag.reporter_id}
                </span>
                {flag.resolution_note && (
                  <p
                    className="mt-1 text-xs text-muted-foreground"
                    data-testid={`flag-note-${flag.id}`}
                  >
                    Note: {flag.resolution_note}
                  </p>
                )}
              </div>
              {flag.status === "pending" && (
                <div className="flex shrink-0 gap-1" data-testid={`flag-actions-${flag.id}`}>
                  <button
                    onClick={() => onResolve(flag.id, "")}
                    data-testid={`flag-resolve-${flag.id}`}
                    title="Resolve"
                    className="rounded-md border border-border p-1.5 text-green-600 transition-colors hover:bg-green-50"
                  >
                    <Check className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => onDismiss(flag.id)}
                    data-testid={`flag-dismiss-${flag.id}`}
                    title="Dismiss"
                    className="rounded-md border border-border p-1.5 text-muted-foreground transition-colors hover:bg-muted"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
