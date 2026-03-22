"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { type LeadsByStatus } from "@/lib/widget-api";

const STAGE_LABELS: Record<keyof LeadsByStatus, string> = {
  new_lead: "New Lead",
  contacted: "Contacted",
  qualified: "Qualified",
  proposal: "Proposal",
  negotiation: "Negotiation",
  closed_won: "Closed Won",
  closed_lost: "Closed Lost",
  nurturing: "Nurturing",
};

const STAGE_COLORS: Record<keyof LeadsByStatus, string> = {
  new_lead: "bg-blue-500",
  contacted: "bg-indigo-500",
  qualified: "bg-purple-500",
  proposal: "bg-amber-500",
  negotiation: "bg-orange-500",
  closed_won: "bg-green-500",
  closed_lost: "bg-red-500",
  nurturing: "bg-teal-500",
};

export interface LeadPipelineWidgetProps {
  /** Auth token for API calls. */
  token: string;
}

/** Displays leads from global-leads grouped by pipeline status. Visible to DEFT sales only. */
export function LeadPipelineWidget({ token }: LeadPipelineWidgetProps): ReactNode {
  const [data, setData] = useState<LeadsByStatus | null>(null);
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
        setError("Failed to load lead pipeline data");
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
      <div data-testid="lead-pipeline-loading" className="text-sm text-muted-foreground">
        Loading pipeline…
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="lead-pipeline-error" className="text-sm text-destructive">
        {error}
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const total = Object.values(data).reduce((sum, n) => sum + n, 0);

  return (
    <div data-testid="lead-pipeline-content" className="space-y-2">
      <div className="text-xs text-muted-foreground">
        Total leads: <span data-testid="lead-pipeline-total">{total}</span>
      </div>
      <div className="space-y-1.5">
        {(Object.entries(data) as [keyof LeadsByStatus, number][]).map(([stage, count]) => (
          <div key={stage} className="flex items-center gap-2" data-testid={`stage-${stage}`}>
            <div className={`h-2 w-2 rounded-full ${STAGE_COLORS[stage]}`} />
            <span className="flex-1 text-xs text-foreground">{STAGE_LABELS[stage]}</span>
            <span className="text-xs font-medium text-foreground">{count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
