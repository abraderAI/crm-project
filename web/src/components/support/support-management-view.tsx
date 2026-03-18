"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle, LifeBuoy, Plus, X } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import {
  fetchGlobalSupportTickets,
  createSupportTicket,
  type GlobalSupportParams,
} from "@/lib/global-api";
import { useTier } from "@/hooks/use-tier";

/** Badge styles keyed by ticket status. */
const STATUS_STYLES: Record<string, string> = {
  open: "bg-yellow-100 text-yellow-800",
  pending: "bg-blue-100 text-blue-800",
  resolved: "bg-green-100 text-green-800",
  closed: "bg-gray-100 text-gray-800",
};

/** Status options for the filter dropdown. */
const STATUS_OPTIONS = [
  { value: "all", label: "All statuses" },
  { value: "open", label: "Open" },
  { value: "pending", label: "Pending" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

/** Filter values for the ticket list. */
interface TicketFilterValues {
  status: string;
  search: string;
}

const DEFAULT_FILTERS: TicketFilterValues = {
  status: "all",
  search: "",
};

/**
 * Support ticket management view with full tier-based RBAC.
 *
 * Tier 1 (anonymous): sign-in prompt, no tickets shown.
 * Tier 2 (registered): own tickets only (mine=true).
 * Tier 3 (customer, no org): own tickets only (mine=true).
 * Tier 3 (customer, with org): org-scoped tickets (org_id).
 * Tier 4 (DEFT employee): all tickets.
 * Tier 5 (customer org admin, subType=owner): org-scoped tickets + stats.
 * Tier 5 (DEFT support admin, deftDepartment=support): all tickets + stats.
 * Tier 6 (platform admin): all tickets + stats.
 */
export function SupportManagementView(): ReactNode {
  const { tier, subType, deftDepartment, orgId, isLoading: tierLoading } = useTier();
  const { getToken } = useAuth();

  const [threads, setThreads] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [filters, setFilters] = useState<TicketFilterValues>(DEFAULT_FILTERS);

  // Create form state.
  const [showCreate, setShowCreate] = useState(false);
  const [createTitle, setCreateTitle] = useState("");
  const [createBody, setCreateBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [createError, setCreateError] = useState("");

  const mountedRef = useRef(true);

  // ---------------------------------------------------------------------------
  // Scope / RBAC derivation
  // ---------------------------------------------------------------------------

  /** Any tier 2+ user has access to at least their own tickets. */
  const hasAccess = tier >= 2;

  /**
   * Sees all tickets globally (tier 4 any dept, tier 5 DEFT dept, tier 6).
   * Tier 5 with subType==="owner" is a customer admin, NOT a DEFT admin.
   */
  const scopesAll = tier === 4 || (tier === 5 && subType !== "owner") || tier >= 6;

  /**
   * Sees org-scoped tickets (tier 3 paying customer with an org,
   * or tier 5 customer org admin).
   */
  const scopesOrg = (tier === 3 && !!orgId) || (tier === 5 && subType === "owner");

  /** Sees only own tickets (tier 2, or tier 3 without an org). */
  const scopesMine = !scopesAll && !scopesOrg;

  /** Show stats strip for org and global views (not for personal-only views). */
  const showStats = scopesAll || scopesOrg;

  // Page heading based on scope.
  let pageTitle = "My Tickets";
  if (scopesAll) {
    const isDeftSupportAdmin = tier === 5 && deftDepartment === "support";
    pageTitle = isDeftSupportAdmin || tier >= 6 ? "Support Dashboard" : "All Support Tickets";
  } else if (scopesOrg) {
    pageTitle = tier === 5 ? "Organization Support" : "Organization Tickets";
  }

  // ---------------------------------------------------------------------------
  // Data fetching
  // ---------------------------------------------------------------------------

  const loadTickets = useCallback(
    async (cursor?: string): Promise<void> => {
      setIsLoading(true);
      setError(null);
      try {
        const token = await getToken();
        if (!token) return;

        const params: GlobalSupportParams = { limit: 50 };
        if (cursor) params.cursor = cursor;
        if (scopesMine) params.mine = true;
        if (scopesOrg && orgId) params.org_id = orgId;

        const result = await fetchGlobalSupportTickets(token, params);

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
          setError(err instanceof Error ? err.message : "Failed to load tickets");
        }
      } finally {
        if (mountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [getToken, scopesMine, scopesOrg, orgId],
  );

  useEffect(() => {
    mountedRef.current = true;
    if (!tierLoading && hasAccess) {
      void loadTickets();
    }
    return () => {
      mountedRef.current = false;
    };
  }, [tierLoading, hasAccess, loadTickets]);

  // ---------------------------------------------------------------------------
  // Client-side filtering
  // ---------------------------------------------------------------------------

  const filteredThreads = threads.filter((thread) => {
    const status = thread.status ?? "open";
    if (filters.status !== "all" && status !== filters.status) return false;
    if (filters.search) {
      const q = filters.search.toLowerCase();
      if (!thread.title.toLowerCase().includes(q)) return false;
    }
    return true;
  });

  // Compute stats from all loaded tickets (for the stats strip).
  let statsOpen = 0;
  let statsPending = 0;
  let statsResolved = 0;
  for (const thread of threads) {
    const status = thread.status ?? "open";
    if (status === "open") statsOpen++;
    else if (status === "pending") statsPending++;
    else if (status === "resolved" || status === "closed") statsResolved++;
    else statsOpen++;
  }

  // ---------------------------------------------------------------------------
  // Create ticket handler
  // ---------------------------------------------------------------------------

  const handleCreate = async (): Promise<void> => {
    if (!createTitle.trim()) return;
    setCreateError("");
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      const newTicket = await createSupportTicket(token, {
        title: createTitle.trim(),
        body: createBody.trim() || undefined,
        org_id: orgId ?? undefined,
      });
      setThreads((prev) => [newTicket, ...prev]);
      setCreateTitle("");
      setCreateBody("");
      setShowCreate(false);
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to create ticket");
    } finally {
      setSubmitting(false);
    }
  };

  // ---------------------------------------------------------------------------
  // Loading / access guard
  // ---------------------------------------------------------------------------

  if (tierLoading) {
    return (
      <div
        data-testid="support-loading-tier"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        Loading...
      </div>
    );
  }

  if (!hasAccess) {
    return (
      <div
        data-testid="support-access-denied"
        className="flex flex-col items-center gap-3 py-12 text-center"
      >
        <LifeBuoy className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm font-medium text-foreground">Sign in to view support tickets</p>
        <p className="text-sm text-muted-foreground">
          Please sign in or register to create and manage support tickets.
        </p>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Main UI
  // ---------------------------------------------------------------------------

  return (
    <div data-testid="support-management-view" className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <LifeBuoy className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold text-foreground">{pageTitle}</h2>
          {!isLoading && (
            <span
              className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
              data-testid="tickets-count"
            >
              {filteredThreads.length}
            </span>
          )}
        </div>
        <button
          data-testid="new-ticket-btn"
          onClick={() => setShowCreate((v) => !v)}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          {showCreate ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          {showCreate ? "Cancel" : "New Ticket"}
        </button>
      </div>

      {/* Fetch error banner */}
      {error && (
        <div
          data-testid="support-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Stats strip — org/global scopes only */}
      {showStats && !isLoading && (
        <div data-testid="support-stats" className="grid grid-cols-3 gap-3">
          <div
            data-testid="stats-open"
            className="rounded-lg border border-border bg-yellow-50 p-3 text-center"
          >
            <span className="block text-xl font-bold text-yellow-700">{statsOpen}</span>
            <span className="text-xs text-yellow-600">Open</span>
          </div>
          <div
            data-testid="stats-pending"
            className="rounded-lg border border-border bg-blue-50 p-3 text-center"
          >
            <span className="block text-xl font-bold text-blue-700">{statsPending}</span>
            <span className="text-xs text-blue-600">Pending</span>
          </div>
          <div
            data-testid="stats-resolved"
            className="rounded-lg border border-border bg-green-50 p-3 text-center"
          >
            <span className="block text-xl font-bold text-green-700">{statsResolved}</span>
            <span className="text-xs text-green-600">Resolved</span>
          </div>
        </div>
      )}

      {/* Inline create form */}
      {showCreate && (
        <div
          data-testid="create-ticket-form"
          className="rounded-lg border border-border bg-background p-4"
        >
          <h3 className="mb-3 text-sm font-semibold text-foreground">New Support Ticket</h3>
          {createError && (
            <div
              data-testid="create-error"
              className="mb-3 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700"
            >
              {createError}
            </div>
          )}
          <div className="flex flex-col gap-3">
            <div>
              <label htmlFor="smv-ticket-title" className="text-xs font-medium text-foreground">
                Title <span className="text-red-500">*</span>
              </label>
              <input
                id="smv-ticket-title"
                data-testid="ticket-title-input"
                type="text"
                value={createTitle}
                onChange={(e) => setCreateTitle(e.target.value)}
                placeholder="Describe your issue"
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label htmlFor="smv-ticket-body" className="text-xs font-medium text-foreground">
                Details
              </label>
              <textarea
                id="smv-ticket-body"
                data-testid="ticket-body-input"
                value={createBody}
                onChange={(e) => setCreateBody(e.target.value)}
                placeholder="Additional details (optional)"
                rows={3}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div className="flex gap-2">
              <button
                data-testid="ticket-submit-btn"
                onClick={() => void handleCreate()}
                disabled={submitting || !createTitle.trim()}
                className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {submitting ? "Submitting..." : "Submit Ticket"}
              </button>
              <button
                data-testid="ticket-cancel-btn"
                onClick={() => {
                  setShowCreate(false);
                  setCreateTitle("");
                  setCreateBody("");
                }}
                className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3" data-testid="tickets-filters">
        <input
          type="search"
          value={filters.search}
          onChange={(e) => setFilters((f) => ({ ...f, search: e.target.value }))}
          placeholder="Search tickets..."
          data-testid="tickets-search-input"
          className="h-8 w-48 rounded-md border border-border bg-background px-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <select
          value={filters.status}
          onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
          data-testid="tickets-status-filter"
          className="h-8 rounded-md border border-border bg-background px-2 text-sm"
        >
          {STATUS_OPTIONS.map(({ value, label }) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>

      {/* Ticket list */}
      {isLoading && threads.length === 0 ? (
        <div
          data-testid="tickets-list-loading"
          className="py-8 text-center text-sm text-muted-foreground"
        >
          Loading tickets...
        </div>
      ) : filteredThreads.length === 0 ? (
        <div data-testid="tickets-empty" className="py-8 text-center text-sm text-muted-foreground">
          No tickets found.
        </div>
      ) : (
        <div
          data-testid="tickets-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {filteredThreads.map((ticket) => {
            const status = ticket.status ?? "open";
            const badgeClass = STATUS_STYLES[status] ?? STATUS_STYLES["open"];
            return (
              <div
                key={ticket.id}
                data-testid={`ticket-row-${ticket.id}`}
                className="flex items-center justify-between px-4 py-3"
              >
                <div className="flex min-w-0 items-start gap-2">
                  <LifeBuoy className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                  <span className="truncate text-sm text-foreground">{ticket.title}</span>
                </div>
                <span
                  data-testid={`ticket-status-${ticket.id}`}
                  className={`ml-2 shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}
                >
                  {status}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {/* Load more */}
      {hasMore && (
        <div className="flex justify-center">
          <button
            onClick={() => void loadTickets(nextCursor)}
            disabled={isLoading}
            data-testid="tickets-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
          >
            {isLoading ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
