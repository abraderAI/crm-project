"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { type TicketSummary } from "@/lib/widget-api";

const STATUS_COLORS: Record<string, string> = {
  open: "bg-red-100 text-red-800",
  pending: "bg-amber-100 text-amber-800",
  resolved: "bg-green-100 text-green-800",
};

export interface TicketQueueWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays all open support tickets from global-support. Visible to DEFT support only. */
export function TicketQueueWidget({ token }: TicketQueueWidgetProps): ReactNode {
  const [tickets, setTickets] = useState<TicketSummary[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      throw new Error("Widget not yet wired to real API");
    } catch {
      if (mountedRef.current) {
        setError("Failed to load ticket queue");
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
      <div data-testid="ticket-queue-loading" className="text-sm text-muted-foreground">
        Loading tickets…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="ticket-queue-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (tickets.length === 0) {
    return (
      <div data-testid="ticket-queue-empty" className="text-sm text-muted-foreground">
        No open tickets.
      </div>
    );
  }

  return (
    <div data-testid="ticket-queue-content" className="space-y-2">
      <ul className="divide-y divide-border">
        {tickets.map((ticket) => (
          <li
            key={ticket.id}
            data-testid={`ticket-${ticket.id}`}
            className="flex items-center justify-between py-1.5"
          >
            <div className="min-w-0 flex-1">
              <span className="text-xs font-medium text-foreground">{ticket.title}</span>
              <span className="ml-2 text-xs text-muted-foreground">{ticket.org_name}</span>
            </div>
            <span
              className={`rounded-full px-2 py-0.5 text-xs ${STATUS_COLORS[ticket.status] ?? "bg-gray-100 text-gray-800"}`}
              data-testid={`ticket-status-${ticket.id}`}
            >
              {ticket.status}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
