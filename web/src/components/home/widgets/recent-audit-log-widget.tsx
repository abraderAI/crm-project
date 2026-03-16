"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { fetchRecentAuditEvents, type AuditEvent } from "@/lib/widget-api";

const ACTION_COLORS: Record<string, string> = {
  create: "bg-green-100 text-green-800",
  update: "bg-blue-100 text-blue-800",
  delete: "bg-red-100 text-red-800",
  login: "bg-gray-100 text-gray-800",
};

export interface RecentAuditLogWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays the most recent audit log events with actor and action. */
export function RecentAuditLogWidget({ token }: RecentAuditLogWidgetProps): ReactNode {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchRecentAuditEvents(token, 10);
      if (mountedRef.current) {
        setEvents(result);
      }
    } catch {
      if (mountedRef.current) {
        setError("Failed to load audit events");
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
      <div data-testid="audit-log-loading" className="text-sm text-muted-foreground">
        Loading audit log…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="audit-log-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <div data-testid="audit-log-empty" className="text-sm text-muted-foreground">
        No recent audit events.
      </div>
    );
  }

  return (
    <div data-testid="audit-log-content" className="space-y-2">
      <ul className="divide-y divide-border">
        {events.map((event) => (
          <li
            key={event.id}
            data-testid={`audit-${event.id}`}
            className="flex items-center justify-between py-1.5"
          >
            <div className="min-w-0 flex-1">
              <span className="text-xs font-medium text-foreground">{event.actor}</span>
              <span
                className={`ml-2 rounded-full px-2 py-0.5 text-xs ${ACTION_COLORS[event.action] ?? "bg-gray-100 text-gray-800"}`}
                data-testid={`audit-action-${event.id}`}
              >
                {event.action}
              </span>
              <span className="ml-2 text-xs text-muted-foreground">
                {event.entity_type}/{event.entity_id}
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
