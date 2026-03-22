"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { type RecentLead } from "@/lib/widget-api";

const STATUS_BADGE: Record<string, string> = {
  new_lead: "bg-blue-100 text-blue-800",
  contacted: "bg-indigo-100 text-indigo-800",
  qualified: "bg-purple-100 text-purple-800",
};

export interface RecentLeadsWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays the most recent leads with source and status. */
export function RecentLeadsWidget({ token }: RecentLeadsWidgetProps): ReactNode {
  const [leads, setLeads] = useState<RecentLead[]>([]);
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
        setError("Failed to load recent leads");
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
      <div data-testid="recent-leads-loading" className="text-sm text-muted-foreground">
        Loading leads…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="recent-leads-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (leads.length === 0) {
    return (
      <div data-testid="recent-leads-empty" className="text-sm text-muted-foreground">
        No leads found.
      </div>
    );
  }

  return (
    <div data-testid="recent-leads-content" className="space-y-2">
      <ul className="divide-y divide-border">
        {leads.map((lead) => (
          <li
            key={lead.id}
            data-testid={`lead-${lead.id}`}
            className="flex items-center justify-between py-1.5"
          >
            <div className="min-w-0 flex-1">
              <span className="text-xs font-medium text-foreground">{lead.title}</span>
              <span className="ml-2 text-xs text-muted-foreground">{lead.source}</span>
            </div>
            <span
              className={`rounded-full px-2 py-0.5 text-xs ${STATUS_BADGE[lead.status] ?? "bg-gray-100 text-gray-800"}`}
              data-testid={`lead-status-${lead.id}`}
            >
              {lead.status}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
