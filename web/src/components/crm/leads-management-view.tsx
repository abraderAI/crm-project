"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle, TrendingUp } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import { fetchGlobalLeads } from "@/lib/global-api";
import { parseLeadData, STAGE_LABELS, STAGE_COLORS, type PipelineStage } from "@/lib/crm-types";
import { useTier } from "@/hooks/use-tier";

/** Filter values for the leads list. */
interface LeadsFilterValues {
  stage: string;
  assignee: string;
  search: string;
}

const DEFAULT_FILTERS: LeadsFilterValues = {
  stage: "all",
  assignee: "all",
  search: "",
};

/**
 * Leads management view for DEFT sales staff.
 * Tier 6 and Tier 5 see all leads; Tier 4 sales reps see only own/assigned leads.
 * All other tiers receive an access-denied message.
 */
export function LeadsManagementView(): ReactNode {
  const { tier, deftDepartment, isLoading: tierLoading } = useTier();
  const { getToken } = useAuth();

  const [threads, setThreads] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [filters, setFilters] = useState<LeadsFilterValues>(DEFAULT_FILTERS);
  const mountedRef = useRef(true);

  /** Tier 5 (org manager) and Tier 6 (platform admin) can see all leads. */
  const canSeeAll = tier >= 5;
  /** Tier 4 DEFT sales reps see only their own / assigned leads. */
  const isSalesRep = tier === 4 && deftDepartment === "sales";
  const hasAccess = canSeeAll || isSalesRep;

  const loadLeads = useCallback(
    async (cursor?: string): Promise<void> => {
      setIsLoading(true);
      setError(null);
      try {
        const token = await getToken();
        if (!token) return;

        const result = await fetchGlobalLeads(token, {
          mine: isSalesRep,
          limit: 50,
          cursor,
        });

        if (!mountedRef.current) return;

        if (cursor) {
          setThreads((prev) => [...prev, ...result.data]);
        } else {
          setThreads(result.data);
        }
        setHasMore(result.page_info.has_more);
        setNextCursor(result.page_info.next_cursor);
      } catch (err) {
        if (mountedRef.current) {
          setError(err instanceof Error ? err.message : "Failed to load leads");
        }
      } finally {
        if (mountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [getToken, isSalesRep],
  );

  useEffect(() => {
    mountedRef.current = true;
    if (!tierLoading && hasAccess) {
      void loadLeads();
    }
    return () => {
      mountedRef.current = false;
    };
  }, [tierLoading, hasAccess, loadLeads]);

  // ---------------------------------------------------------------------------
  // Client-side filtering
  // ---------------------------------------------------------------------------

  const filteredThreads = threads.filter((thread) => {
    const lead = parseLeadData(thread.metadata);
    if (filters.stage !== "all" && thread.stage !== filters.stage) return false;
    if (filters.assignee !== "all" && lead.assigned_to !== filters.assignee) return false;
    if (filters.search) {
      const q = filters.search.toLowerCase();
      const matchesTitle = thread.title.toLowerCase().includes(q);
      const matchesCompany = lead.company?.toLowerCase().includes(q) ?? false;
      if (!matchesTitle && !matchesCompany) return false;
    }
    return true;
  });

  // Collect unique assignees for the assignee filter dropdown (tier 5+ only).
  const uniqueAssignees = Array.from(
    new Set(
      threads
        .map((t) => parseLeadData(t.metadata).assigned_to)
        .filter((a): a is string => Boolean(a)),
    ),
  ).sort();

  // ---------------------------------------------------------------------------
  // Loading / access-denied states
  // ---------------------------------------------------------------------------

  if (tierLoading) {
    return (
      <div
        data-testid="leads-loading-tier"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        Loading...
      </div>
    );
  }

  if (!hasAccess) {
    return (
      <div
        data-testid="leads-access-denied"
        className="flex flex-col items-center gap-3 py-12 text-center"
      >
        <AlertTriangle className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm font-medium text-foreground">Access Denied</p>
        <p className="text-sm text-muted-foreground">
          This page is only available to DEFT sales staff.
        </p>
      </div>
    );
  }

  const pageTitle = canSeeAll ? "All Leads" : "My Leads";

  return (
    <div data-testid="leads-management-view" className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center gap-2">
        <TrendingUp className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold text-foreground">{pageTitle}</h2>
        {!isLoading && (
          <span
            className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
            data-testid="leads-count"
          >
            {filteredThreads.length}
          </span>
        )}
      </div>

      {/* Error banner */}
      {error && (
        <div
          data-testid="leads-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3" data-testid="leads-filters">
        <input
          type="search"
          value={filters.search}
          onChange={(e) => setFilters((f) => ({ ...f, search: e.target.value }))}
          placeholder="Search leads..."
          data-testid="leads-search-input"
          className="h-8 w-48 rounded-md border border-border bg-background px-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <select
          value={filters.stage}
          onChange={(e) => setFilters((f) => ({ ...f, stage: e.target.value }))}
          data-testid="leads-stage-filter"
          className="h-8 rounded-md border border-border bg-background px-2 text-sm"
        >
          <option value="all">All stages</option>
          {Object.entries(STAGE_LABELS).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
        {canSeeAll && (
          <select
            value={filters.assignee}
            onChange={(e) => setFilters((f) => ({ ...f, assignee: e.target.value }))}
            data-testid="leads-assignee-filter"
            className="h-8 rounded-md border border-border bg-background px-2 text-sm"
          >
            <option value="all">All assignees</option>
            {uniqueAssignees.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Lead list */}
      {isLoading && threads.length === 0 ? (
        <div
          data-testid="leads-list-loading"
          className="py-8 text-center text-sm text-muted-foreground"
        >
          Loading leads...
        </div>
      ) : filteredThreads.length === 0 ? (
        <div data-testid="leads-empty" className="py-8 text-center text-sm text-muted-foreground">
          No leads found.
        </div>
      ) : (
        <div
          data-testid="leads-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {filteredThreads.map((thread) => {
            const lead = parseLeadData(thread.metadata);
            const stage = (thread.stage ?? "new_lead") as PipelineStage;
            const stageLabel = STAGE_LABELS[stage] ?? stage;
            const stageColor = STAGE_COLORS[stage] ?? "";

            return (
              <Link
                key={thread.id}
                href={`/crm/leads/global/${thread.slug}`}
                data-testid={`lead-row-${thread.id}`}
                className="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-accent/50"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-foreground">{thread.title}</p>
                  {lead.company && (
                    <p
                      className="text-xs text-muted-foreground"
                      data-testid={`lead-company-${thread.id}`}
                    >
                      {lead.company}
                    </p>
                  )}
                </div>
                <span
                  className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${stageColor}`}
                  data-testid={`lead-stage-${thread.id}`}
                >
                  {stageLabel}
                </span>
                {canSeeAll && lead.assigned_to && (
                  <span
                    className="hidden shrink-0 text-xs text-muted-foreground sm:block"
                    data-testid={`lead-assignee-${thread.id}`}
                  >
                    {lead.assigned_to}
                  </span>
                )}
                {lead.score !== undefined && (
                  <span
                    className="shrink-0 text-xs font-medium tabular-nums text-foreground"
                    data-testid={`lead-score-${thread.id}`}
                  >
                    {lead.score}
                  </span>
                )}
              </Link>
            );
          })}
        </div>
      )}

      {/* Load more */}
      {hasMore && (
        <div className="flex justify-center">
          <button
            onClick={() => void loadLeads(nextCursor)}
            disabled={isLoading}
            data-testid="leads-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
          >
            {isLoading ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
