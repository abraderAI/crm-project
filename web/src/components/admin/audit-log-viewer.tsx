"use client";

import { useState } from "react";
import { History, Eye } from "lucide-react";
import { cn } from "@/lib/utils";
import type { AuditEntry } from "@/lib/api-types";

export interface AuditLogViewerProps {
  /** Audit log entries. */
  entries: AuditEntry[];
  /** Whether data is loading. */
  loading?: boolean;
  /** Whether there are more pages. */
  hasMore?: boolean;
  /** Called when the user requests the next page. */
  onLoadMore?: () => void;
  /** Resolve a user ID to a display label (e.g. "Alice (DEFT)"). */
  formatUser?: (userId: string) => string;
}

/** Format a date string for display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

/** Pretty-print JSON safely. */
function prettyJson(raw?: string): string {
  if (!raw) return "—";
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

/** Action type badge color. */
function actionColor(action: string): string {
  if (action.startsWith("create")) return "bg-green-100 text-green-800";
  if (action.startsWith("update")) return "bg-blue-100 text-blue-800";
  if (action.startsWith("delete")) return "bg-red-100 text-red-800";
  return "bg-muted text-muted-foreground";
}

/** Audit log viewer displaying entries with expandable before/after JSON diff. */
export function AuditLogViewer({
  entries,
  loading = false,
  hasMore = false,
  onLoadMore,
  formatUser,
}: AuditLogViewerProps): React.ReactNode {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const toggleExpand = (id: string): void => {
    setExpandedId(expandedId === id ? null : id);
  };

  return (
    <div data-testid="audit-log-viewer" className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <History className="h-5 w-5 text-muted-foreground" data-testid="audit-icon" />
        <h2 className="text-lg font-semibold text-foreground">Audit Log</h2>
      </div>

      {loading && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="audit-loading">
          Loading audit log...
        </div>
      )}

      {!loading && entries.length === 0 && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="audit-empty">
          No audit log entries.
        </div>
      )}

      {!loading && entries.length > 0 && (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="audit-list"
        >
          {entries.map((entry) => (
            <div key={entry.id} data-testid={`audit-item-${entry.id}`}>
              <div className="flex items-center gap-3 px-4 py-2.5">
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-xs font-medium",
                    actionColor(entry.action),
                  )}
                  data-testid={`audit-action-${entry.id}`}
                >
                  {entry.action}
                </span>
                <span className="text-sm text-foreground" data-testid={`audit-entity-${entry.id}`}>
                  {entry.entity_type}:{entry.entity_id.slice(0, 8)}
                </span>
                <span
                  className="text-xs text-muted-foreground"
                  data-testid={`audit-user-${entry.id}`}
                >
                  {formatUser ? formatUser(entry.user_id) : entry.user_id}
                </span>
                <span
                  className="ml-auto text-xs text-muted-foreground"
                  data-testid={`audit-date-${entry.id}`}
                >
                  {formatDate(entry.created_at)}
                </span>
                {(entry.before_state || entry.after_state) && (
                  <button
                    onClick={() => toggleExpand(entry.id)}
                    data-testid={`audit-expand-${entry.id}`}
                    className="rounded-md border border-border p-1 text-muted-foreground transition-colors hover:bg-accent"
                  >
                    <Eye className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
              {expandedId === entry.id && (
                <div
                  className="border-t border-border bg-muted/20 px-4 py-3"
                  data-testid={`audit-diff-${entry.id}`}
                >
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div>
                      <p className="mb-1 text-xs font-medium text-muted-foreground">Before</p>
                      <pre
                        className="max-h-48 overflow-auto rounded-md bg-muted p-2 text-xs text-foreground"
                        data-testid={`audit-before-${entry.id}`}
                      >
                        {prettyJson(entry.before_state)}
                      </pre>
                    </div>
                    <div>
                      <p className="mb-1 text-xs font-medium text-muted-foreground">After</p>
                      <pre
                        className="max-h-48 overflow-auto rounded-md bg-muted p-2 text-xs text-foreground"
                        data-testid={`audit-after-${entry.id}`}
                      >
                        {prettyJson(entry.after_state)}
                      </pre>
                    </div>
                  </div>
                  {entry.ip_address && (
                    <p
                      className="mt-2 text-xs text-muted-foreground"
                      data-testid={`audit-ip-${entry.id}`}
                    >
                      IP: {entry.ip_address}
                    </p>
                  )}
                  {entry.request_id && (
                    <p
                      className="text-xs text-muted-foreground"
                      data-testid={`audit-request-${entry.id}`}
                    >
                      Request: {entry.request_id}
                    </p>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {hasMore && !loading && (
        <div className="flex justify-center">
          <button
            onClick={onLoadMore}
            data-testid="audit-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}
