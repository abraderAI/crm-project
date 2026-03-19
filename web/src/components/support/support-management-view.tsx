"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import {
  AlertTriangle,
  ExternalLink,
  LifeBuoy,
  Paperclip,
  Plus,
  Save,
  User,
  X,
} from "lucide-react";

import type { ThreadWithAuthor, Upload } from "@/lib/api-types";
import {
  fetchGlobalSupportTickets,
  createSupportTicket,
  updateSupportTicket,
  fetchThreadAttachments,
  uploadThreadAttachment,
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

/** Status transitions available in the work-view. */
const WORK_STATUS_OPTIONS = [
  { value: "open", label: "Open" },
  { value: "pending", label: "Pending" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

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
  const { tier, subType, orgId, isLoading: tierLoading } = useTier();
  const { getToken } = useAuth();

  const [threads, setThreads] = useState<ThreadWithAuthor[]>([]);
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

  // Work-view modal state.
  const [workTicket, setWorkTicket] = useState<ThreadWithAuthor | null>(null);
  const [workBody, setWorkBody] = useState("");
  const [workStatus, setWorkStatus] = useState("");
  const [workSaving, setWorkSaving] = useState(false);
  const [workError, setWorkError] = useState("");

  // Attachments state.
  const [attachments, setAttachments] = useState<Upload[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [uploadingFile, setUploadingFile] = useState(false);
  const [attachmentError, setAttachmentError] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

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

  // Open a ticket in the work-view modal and load its attachments.
  const handleOpenTicket = (ticket: ThreadWithAuthor): void => {
    setWorkTicket(ticket);
    setWorkBody(ticket.body ?? "");
    setWorkStatus(ticket.status ?? "open");
    setWorkError("");
    setAttachments([]);
    setAttachmentError("");
    // Fetch attachments asynchronously.
    setAttachmentsLoading(true);
    void getToken().then((token) => {
      if (!token) {
        setAttachmentsLoading(false);
        return;
      }
      return fetchThreadAttachments(token, ticket.slug)
        .then((uploads) => {
          setAttachments(uploads);
        })
        .catch((err: unknown) => {
          setAttachmentError(err instanceof Error ? err.message : "Failed to load attachments");
        })
        .finally(() => {
          setAttachmentsLoading(false);
        });
    });
  };

  // Handle file upload in the work-view.
  const handleAttachFile = async (e: React.ChangeEvent<HTMLInputElement>): Promise<void> => {
    const file = e.target.files?.[0];
    if (!file || !workTicket) return;
    // Reset input so the same file can be re-selected.
    if (fileInputRef.current) fileInputRef.current.value = "";
    setUploadingFile(true);
    setAttachmentError("");
    try {
      const token = await getToken();
      if (!token) return;
      const uploaded = await uploadThreadAttachment(token, workTicket.slug, file);
      setAttachments((prev) => [...prev, uploaded]);
    } catch (err) {
      setAttachmentError(err instanceof Error ? err.message : "Failed to upload file");
    } finally {
      setUploadingFile(false);
    }
  };

  // Close the work-view modal without saving.
  const handleCloseWork = (): void => {
    setWorkTicket(null);
    setWorkError("");
    setAttachments([]);
    setAttachmentError("");
  };

  // Save changes from the work-view modal.
  const handleSaveWork = async (): Promise<void> => {
    if (!workTicket) return;
    setWorkSaving(true);
    setWorkError("");
    try {
      const token = await getToken();
      if (!token) return;
      const updated = await updateSupportTicket(token, workTicket.slug, {
        body: workBody,
        status: workStatus,
      });
      setThreads((prev) => prev.map((t) => (t.id === updated.id ? updated : t)));
      setWorkTicket(updated);
    } catch (err) {
      setWorkError(err instanceof Error ? err.message : "Failed to save changes");
    } finally {
      setWorkSaving(false);
    }
  };

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
            const creatorLabel = ticket.author_name ?? ticket.author_email ?? ticket.author_id;
            const orgLabel = ticket.org_name ?? null;
            return (
              <div
                key={ticket.id}
                data-testid={`ticket-row-${ticket.id}`}
                className="flex items-center justify-between px-4 py-3"
              >
                <div className="flex min-w-0 flex-1 items-start gap-2">
                  <LifeBuoy className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                  <div className="min-w-0">
                    <span className="block truncate text-sm font-medium text-foreground">
                      {ticket.title}
                    </span>
                    <div
                      data-testid={`ticket-creator-${ticket.id}`}
                      className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground"
                    >
                      <User className="h-3 w-3 shrink-0" />
                      <span className="truncate">{creatorLabel}</span>
                      {orgLabel && <span className="truncate">&middot; {orgLabel}</span>}
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
                  <button
                    data-testid={`ticket-open-btn-${ticket.id}`}
                    onClick={() => handleOpenTicket(ticket)}
                    className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs font-medium text-foreground transition-colors hover:bg-accent"
                  >
                    <ExternalLink className="h-3 w-3" />
                    Open
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Work-view modal */}
      {workTicket && (
        <div
          data-testid="work-view-modal"
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          role="dialog"
          aria-modal="true"
          aria-label="Support ticket work view"
        >
          <div className="relative flex w-full max-w-lg flex-col gap-4 rounded-lg border border-border bg-background p-6 shadow-lg">
            {/* Modal header */}
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <h3
                  data-testid="work-view-title"
                  className="text-base font-semibold text-foreground"
                >
                  {workTicket.title}
                </h3>
                <div
                  data-testid="work-view-creator"
                  className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground"
                >
                  <User className="h-3 w-3 shrink-0" />
                  <span>
                    {workTicket.author_name ?? workTicket.author_email ?? workTicket.author_id}
                  </span>
                  {workTicket.org_name && <span>&middot; {workTicket.org_name}</span>}
                </div>
              </div>
              <button
                data-testid="work-view-close-btn"
                onClick={handleCloseWork}
                className="shrink-0 rounded-md p-1 text-muted-foreground hover:bg-accent"
                aria-label="Close"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Error banner */}
            {workError && (
              <div
                data-testid="work-view-error"
                className="flex items-center gap-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700"
              >
                <AlertTriangle className="h-4 w-4 shrink-0" />
                {workError}
              </div>
            )}

            {/* Status selector — only operators/admins may change status */}
            {scopesAll && (
              <div>
                <label htmlFor="work-view-status" className="text-xs font-medium text-foreground">
                  Status
                </label>
                <select
                  id="work-view-status"
                  data-testid="work-view-status-select"
                  value={workStatus}
                  onChange={(e) => setWorkStatus(e.target.value)}
                  className="mt-1 w-full rounded-md border border-border bg-background px-2 py-2 text-sm"
                >
                  {WORK_STATUS_OPTIONS.map(({ value, label }) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* Issue body */}
            <div>
              <label htmlFor="work-view-body" className="text-xs font-medium text-foreground">
                Description
              </label>
              <textarea
                id="work-view-body"
                data-testid="work-view-body-input"
                value={workBody}
                onChange={(e) => setWorkBody(e.target.value)}
                rows={5}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>

            {/* Last updated timestamp */}
            {workTicket.updated_at && (
              <p data-testid="work-view-updated-at" className="text-xs text-muted-foreground">
                Last updated: {new Date(workTicket.updated_at).toLocaleString()}
              </p>
            )}

            {/* Attachments section */}
            <div>
              <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-foreground">Attachments</label>
                <label
                  htmlFor="work-view-file-input"
                  data-testid="work-view-attach-btn"
                  className="inline-flex cursor-pointer items-center gap-1 rounded-md border border-border px-2 py-1 text-xs font-medium text-foreground hover:bg-accent"
                >
                  <Paperclip className="h-3 w-3" />
                  {uploadingFile ? "Uploading..." : "Attach file"}
                </label>
                <input
                  id="work-view-file-input"
                  data-testid="work-view-file-input"
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  onChange={(e) => void handleAttachFile(e)}
                  disabled={uploadingFile}
                />
              </div>
              {attachmentError && (
                <p data-testid="work-view-attachment-error" className="mt-1 text-xs text-red-600">
                  {attachmentError}
                </p>
              )}
              {attachmentsLoading ? (
                <p
                  data-testid="work-view-attachments-loading"
                  className="mt-1 text-xs text-muted-foreground"
                >
                  Loading attachments...
                </p>
              ) : attachments.length > 0 ? (
                <ul data-testid="work-view-attachments-list" className="mt-2 space-y-1">
                  {attachments.map((a) => (
                    <li key={a.id} className="flex items-center gap-1 text-xs">
                      <Paperclip className="h-3 w-3 shrink-0 text-muted-foreground" />
                      <span className="truncate text-foreground">{a.filename}</span>
                      <span className="shrink-0 text-muted-foreground">
                        ({(a.size / 1024).toFixed(1)} KB)
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-1 text-xs text-muted-foreground">No attachments.</p>
              )}
            </div>

            {/* Actions */}
            <div className="flex gap-2">
              <button
                data-testid="work-view-save-btn"
                onClick={() => void handleSaveWork()}
                disabled={workSaving}
                className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                <Save className="h-4 w-4" />
                {workSaving ? "Saving..." : "Save changes"}
              </button>
              <button
                data-testid="work-view-cancel-btn"
                onClick={handleCloseWork}
                className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
              >
                Close
              </button>
            </div>
          </div>
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
