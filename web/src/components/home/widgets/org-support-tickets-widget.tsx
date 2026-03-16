"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { LifeBuoy } from "lucide-react";
import type { Thread } from "@/lib/api-types";
import { fetchOrgSupportTickets } from "@/lib/org-api";

/** Maximum number of tickets to display. */
const MAX_ITEMS = 5;

/** Map of ticket status to badge styling. */
const STATUS_STYLES: Record<string, string> = {
  open: "bg-yellow-100 text-yellow-800",
  pending: "bg-blue-100 text-blue-800",
  resolved: "bg-green-100 text-green-800",
  closed: "bg-gray-100 text-gray-800",
};

interface OrgSupportTicketsWidgetProps {
  /** Auth token for API calls. */
  token: string;
  /** Org ID to filter tickets by. */
  orgId: string;
}

/** Displays support tickets filtered to the user's org. No cross-org leakage. */
export function OrgSupportTicketsWidget({ token, orgId }: OrgSupportTicketsWidgetProps): ReactNode {
  const [tickets, setTickets] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchOrgSupportTickets(token, orgId, { limit: MAX_ITEMS });
      if (!mountedRef.current) return;
      setTickets(result.data);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load support tickets.");
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [token, orgId]);

  useEffect(() => {
    mountedRef.current = true;
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, [load]);

  if (isLoading) {
    return (
      <div data-testid="org-support-tickets-loading" className="animate-pulse space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-4 rounded bg-muted" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="org-support-tickets-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (tickets.length === 0) {
    return (
      <p data-testid="org-support-tickets-empty" className="text-sm text-muted-foreground">
        No support tickets for your organization.
      </p>
    );
  }

  return (
    <ul data-testid="org-support-tickets-list" className="space-y-2">
      {tickets.map((ticket) => {
        const status = ticket.status ?? "open";
        const badgeClass = STATUS_STYLES[status] ?? STATUS_STYLES["open"];
        return (
          <li
            key={ticket.id}
            className="flex items-center justify-between rounded p-1.5 text-sm hover:bg-accent/50"
          >
            <div className="flex items-start gap-2">
              <LifeBuoy className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
              <span className="text-foreground">{ticket.title}</span>
            </div>
            <span
              data-testid={`org-ticket-status-${ticket.id}`}
              className={`rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}
            >
              {status}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
