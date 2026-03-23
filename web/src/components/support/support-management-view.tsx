"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import Link from "next/link";
import { AlertTriangle, CheckCircle, ExternalLink, LifeBuoy, Plus, User, X } from "lucide-react";

import type { Thread, ThreadWithAuthor } from "@/lib/api-types";
import { fetchGlobalSupportTickets, type GlobalSupportParams } from "@/lib/global-api";
import { fetchUnclaimedTickets, claimTickets } from "@/lib/support-api";
import { useTier } from "@/hooks/use-tier";
import { useUserDirectory } from "@/lib/use-user-directory";

/** Badge styles keyed by ticket status. */
const STATUS_STYLES: Record<string, string> = {
  open: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  assigned: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  resolved: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  closed: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200",
};

/** Status options for the filter dropdown. */
const STATUS_OPTIONS = [
  { value: "all", label: "All statuses" },
  { value: "open", label: "Open" },
  { value: "assigned", label: "Assigned" },
  { value: "pending", label: "Pending" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

/** Filter values for the ticket list. */
interface TicketFilterValues {
  status: string;
  search: string;
  sort: "newest" | "oldest" | "updated";
}

const DEFAULT_FILTERS: TicketFilterValues = {
  status: "all",
  search: "",
  sort: "newest",
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
  const { tier, subType, orgId, isLoading: tierLoading } = useTier();
  const { getToken } = useAuth();
  const userDir = useUserDirectory();

  const [threads, setThreads] = useState<ThreadWithAuthor[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [filters, setFilters] = useState<TicketFilterValues>(DEFAULT_FILTERS);

  // Unclaimed ticket state.
  const [unclaimedTickets, setUnclaimedTickets] = useState<Thread[]>([]);
  const [selectedUnclaimed, setSelectedUnclaimed] = useState<Set<string>>(new Set());
  const [claimingInProgress, setClaimingInProgress] = useState(false);
  const [unclaimedDismissed, setUnclaimedDismissed] = useState(() => {
    if (typeof window !== "undefined") {
      return localStorage.getItem("deft-unclaimed-dismissed") === "true";
    }
    return false;
  });

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

  // Page heading is always the product name regardless of tier.
  const pageTitle = "DEFT.support";

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

  // Load unclaimed tickets for non-DEFT users.
  const loadUnclaimed = useCallback(async (): Promise<void> => {
    try {
      const token = await getToken();
      if (!token) return;
      const tickets = await fetchUnclaimedTickets(token);
      if (mountedRef.current) setUnclaimedTickets(tickets);
    } catch {
      // best-effort; don't block the page
    }
  }, [getToken]);

  useEffect(() => {
    mountedRef.current = true;
    if (!tierLoading && hasAccess) {
      void loadTickets();
      // Check for unclaimed tickets for non-DEFT users.
      if (!scopesAll && !unclaimedDismissed) {
        void loadUnclaimed();
      }
    }
    return () => {
      mountedRef.current = false;
    };
  }, [tierLoading, hasAccess, loadTickets, scopesAll, unclaimedDismissed, loadUnclaimed]);

  // ---------------------------------------------------------------------------
  // Client-side filtering
  // ---------------------------------------------------------------------------

  const filteredThreads = threads.filter((thread) => {
    const status = thread.status ?? "open";
    if (filters.status !== "all" && status !== filters.status) return false;
    if (filters.search) {
      const q = filters.search.toLowerCase();
      const matchesTitle = thread.title.toLowerCase().includes(q);
      const matchesContactEmail = thread.contact_email?.toLowerCase().includes(q) ?? false;
      const matchesAuthorEmail = thread.author_email?.toLowerCase().includes(q) ?? false;
      if (!matchesTitle && !matchesContactEmail && !matchesAuthorEmail) return false;
    }
    return true;
  });
  const sortedThreads = [...filteredThreads].sort((a, b) => {
    if (filters.sort === "oldest") {
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
    }
    if (filters.sort === "updated") {
      return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
    }
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
  });

  // Compute stats from all loaded tickets (for the stats strip).
  let statsOpen = 0;
  let statsAssigned = 0;
  let statsPending = 0;
  let statsResolved = 0;
  for (const thread of threads) {
    const status = thread.status ?? "open";
    if (status === "open") statsOpen++;
    else if (status === "assigned") statsAssigned++;
    else if (status === "pending") statsPending++;
    else if (status === "resolved" || status === "closed") statsResolved++;
    else statsOpen++;
  }

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
        <LifeBuoy className="h-8 w-8 text-primary/80" />
        <p className="text-sm font-semibold text-foreground">DEFT.support</p>
        <p className="text-xs text-muted-foreground">Loading support workspace…</p>
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
              {sortedThreads.length}
            </span>
          )}
        </div>
        <Link
          data-testid="new-ticket-btn"
          href="/support/tickets/new"
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          New Ticket
        </Link>
      </div>

      {/* Unclaimed ticket claim banner */}
      {unclaimedTickets.length > 0 && !unclaimedDismissed && (
        <div
          data-testid="unclaimed-ticket-banner"
          className="rounded-lg border border-primary/30 bg-primary/5 p-4"
        >
          <div className="mb-2 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CheckCircle className="h-4 w-4 text-primary" />
              <span className="text-sm font-semibold text-foreground">
                We found {unclaimedTickets.length} ticket(s) created before you registered
              </span>
            </div>
            <button
              data-testid="unclaimed-dismiss"
              onClick={() => {
                setUnclaimedDismissed(true);
                localStorage.setItem("deft-unclaimed-dismissed", "true");
              }}
              className="text-muted-foreground hover:text-foreground"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="mb-3 text-xs text-muted-foreground">
            Would you like to add them to your ticket history?
          </p>
          <ul className="mb-3 flex flex-col gap-1">
            {unclaimedTickets.map((t) => (
              <li key={t.id} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  data-testid={`unclaimed-check-${t.id}`}
                  checked={selectedUnclaimed.has(t.id)}
                  onChange={(e) => {
                    setSelectedUnclaimed((prev) => {
                      const next = new Set(prev);
                      if (e.target.checked) next.add(t.id);
                      else next.delete(t.id);
                      return next;
                    });
                  }}
                  className="rounded border-border"
                />
                <span className="text-foreground">{t.title}</span>
              </li>
            ))}
          </ul>
          <button
            data-testid="unclaimed-claim-btn"
            disabled={selectedUnclaimed.size === 0 || claimingInProgress}
            onClick={async () => {
              setClaimingInProgress(true);
              try {
                const token = await getToken();
                if (!token) return;
                await claimTickets(token, Array.from(selectedUnclaimed));
                setUnclaimedTickets([]);
                setSelectedUnclaimed(new Set());
                void loadTickets();
              } catch {
                // best-effort
              } finally {
                setClaimingInProgress(false);
              }
            }}
            className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {claimingInProgress ? "Claiming..." : "Claim Selected"}
          </button>
        </div>
      )}

      {/* Fetch error banner */}
      {error && (
        <div
          data-testid="support-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/50 dark:text-red-300"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Stats strip — org/global scopes only */}
      {showStats && !isLoading && (
        <div data-testid="support-stats" className="grid grid-cols-4 gap-3">
          <div
            data-testid="stats-open"
            className="rounded-lg border border-border bg-yellow-50 p-3 text-center dark:bg-yellow-950 dark:border-yellow-800"
          >
            <span className="block text-xl font-bold text-yellow-700 dark:text-yellow-200">
              {statsOpen}
            </span>
            <span className="text-xs text-yellow-600 dark:text-yellow-300">Open</span>
          </div>
          <div
            data-testid="stats-assigned"
            className="rounded-lg border border-border bg-purple-50 p-3 text-center dark:bg-purple-950 dark:border-purple-800"
          >
            <span className="block text-xl font-bold text-purple-700 dark:text-purple-200">
              {statsAssigned}
            </span>
            <span className="text-xs text-purple-600 dark:text-purple-300">Assigned</span>
          </div>
          <div
            data-testid="stats-pending"
            className="rounded-lg border border-border bg-blue-50 p-3 text-center dark:bg-blue-950 dark:border-blue-800"
          >
            <span className="block text-xl font-bold text-blue-700 dark:text-blue-200">
              {statsPending}
            </span>
            <span className="text-xs text-blue-600 dark:text-blue-300">Pending</span>
          </div>
          <div
            data-testid="stats-resolved"
            className="rounded-lg border border-border bg-green-50 p-3 text-center dark:bg-green-950 dark:border-green-800"
          >
            <span className="block text-xl font-bold text-green-700 dark:text-green-200">
              {statsResolved}
            </span>
            <span className="text-xs text-green-600 dark:text-green-300">Resolved</span>
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
        <select
          value={filters.sort}
          onChange={(e) =>
            setFilters((f) => ({
              ...f,
              sort: e.target.value as TicketFilterValues["sort"],
            }))
          }
          data-testid="tickets-sort-filter"
          className="h-8 rounded-md border border-border bg-background px-2 text-sm"
        >
          <option value="newest">Newest first</option>
          <option value="oldest">Oldest first</option>
          <option value="updated">Recently updated</option>
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
      ) : sortedThreads.length === 0 ? (
        <div data-testid="tickets-empty" className="py-8 text-center text-sm text-muted-foreground">
          No tickets found.
        </div>
      ) : (
        <div
          data-testid="tickets-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {sortedThreads.map((ticket) => {
            const status = ticket.status ?? "open";
            const badgeClass = STATUS_STYLES[status] ?? STATUS_STYLES["open"];
            const resolved = userDir.resolve(ticket.author_id);
            // Prioritise contact_email when the ticket was created on behalf
            // of another user, then fall back through normal author fields.
            const creatorLabel =
              ticket.contact_email ??
              ticket.author_name ??
              resolved?.display_name ??
              ticket.author_email ??
              ticket.author_id;
            const orgLabel = resolved?.org_name || ticket.org_name || null;
            return (
              <div
                key={ticket.id}
                data-testid={`ticket-row-${ticket.id}`}
                className="flex items-center justify-between px-4 py-3"
              >
                <div className="flex min-w-0 flex-1 items-start gap-2">
                  <LifeBuoy className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      {ticket.ticket_number ? (
                        <span
                          data-testid={`ticket-number-${ticket.id}`}
                          className="shrink-0 rounded bg-primary/10 px-1.5 py-0.5 font-mono text-xs font-semibold text-primary"
                        >
                          #{ticket.ticket_number}
                        </span>
                      ) : null}
                      <span className="block truncate text-sm font-medium text-foreground">
                        {ticket.title}
                      </span>
                    </div>
                    <div
                      data-testid={`ticket-creator-${ticket.id}`}
                      className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground"
                    >
                      <User className="h-3 w-3 shrink-0" />
                      <span className="truncate">{creatorLabel}</span>
                      {orgLabel ? (
                        <span
                          className="rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-700 dark:bg-blue-900 dark:text-blue-200"
                          data-testid={`ticket-org-badge-${ticket.id}`}
                        >
                          {orgLabel}
                        </span>
                      ) : ticket.registration_status === "unregistered" ? (
                        <span
                          className="rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300"
                          data-testid={`ticket-reg-badge-${ticket.id}`}
                        >
                          Unregistered
                        </span>
                      ) : ticket.registration_status === "registered" ? (
                        <span
                          className="rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-700 dark:bg-blue-900 dark:text-blue-200"
                          data-testid={`ticket-reg-badge-${ticket.id}`}
                        >
                          Registered
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>
                <div className="ml-3 flex shrink-0 items-center gap-2">
                  <span
                    data-testid={`ticket-status-${ticket.id}`}
                    className={`rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}
                  >
                    {status}
                  </span>
                  <Link
                    data-testid={`ticket-open-btn-${ticket.id}`}
                    href={`/support/tickets/${ticket.slug}`}
                    className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs font-medium text-foreground transition-colors hover:bg-accent"
                  >
                    <ExternalLink className="h-3 w-3" />
                    Open
                  </Link>
                </div>
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
