"use client";

import { RotateCcw } from "lucide-react";
import { cn } from "@/lib/utils";
import type { WebhookDelivery } from "@/lib/api-types";

export interface WebhookDeliveryLogProps {
  /** Delivery records to display. */
  deliveries: WebhookDelivery[];
  /** Whether data is loading. */
  loading?: boolean;
  /** Called to replay a delivery. */
  onReplay: (deliveryId: string) => void;
  /** Whether there are more pages. */
  hasMore?: boolean;
  /** Called when the user requests the next page. */
  onLoadMore?: () => void;
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

/** Color for HTTP status codes. */
function statusColor(code: number): string {
  if (code >= 200 && code < 300) return "bg-green-100 text-green-800";
  if (code >= 400 && code < 500) return "bg-yellow-100 text-yellow-800";
  return "bg-red-100 text-red-800";
}

/** Webhook delivery log table with replay support. */
export function WebhookDeliveryLog({
  deliveries,
  loading = false,
  onReplay,
  hasMore = false,
  onLoadMore,
}: WebhookDeliveryLogProps): React.ReactNode {
  return (
    <div data-testid="delivery-log" className="flex flex-col gap-4">
      <h3 className="text-sm font-semibold text-foreground">Delivery Log</h3>

      {loading && (
        <div
          className="py-6 text-center text-sm text-muted-foreground"
          data-testid="delivery-loading"
        >
          Loading deliveries...
        </div>
      )}

      {!loading && deliveries.length === 0 && (
        <div
          className="py-6 text-center text-sm text-muted-foreground"
          data-testid="delivery-empty"
        >
          No deliveries recorded.
        </div>
      )}

      {!loading && deliveries.length > 0 && (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="delivery-list"
        >
          {deliveries.map((d) => (
            <div
              key={d.id}
              className="flex items-center gap-3 px-4 py-2.5"
              data-testid={`delivery-item-${d.id}`}
            >
              <span
                className={cn(
                  "rounded-full px-2 py-0.5 text-xs font-medium",
                  statusColor(d.status_code),
                )}
                data-testid={`delivery-status-${d.id}`}
              >
                {d.status_code}
              </span>
              <span className="text-sm text-foreground" data-testid={`delivery-event-${d.id}`}>
                {d.event_type}
              </span>
              <span
                className="text-xs text-muted-foreground"
                data-testid={`delivery-attempts-${d.id}`}
              >
                {d.attempts} attempt{d.attempts !== 1 ? "s" : ""}
              </span>
              <span
                className="ml-auto text-xs text-muted-foreground"
                data-testid={`delivery-date-${d.id}`}
              >
                {formatDate(d.created_at)}
              </span>
              <button
                onClick={() => onReplay(d.id)}
                data-testid={`delivery-replay-${d.id}`}
                title="Replay"
                className="rounded-md border border-border p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                <RotateCcw className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      {hasMore && !loading && (
        <div className="flex justify-center">
          <button
            onClick={onLoadMore}
            data-testid="delivery-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}
