"use client";

import { useCallback, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { Eye, EyeOff, Lock, Pencil, Send, X } from "lucide-react";

import type { SupportEntry, SupportEntryType } from "@/lib/api-types";
import { publishTicketEntry, setEntryDeftVisibility, updateTicketEntry } from "@/lib/support-api";

/**
 * Visual config for each entry type.
 * Colors:
 *   customer + agent_reply (published conversation) — light green
 *   draft                                            — light gray
 *   context (internal / DEFT-only)                  — light red
 *   system_event (status changes etc.)              — light orange
 */
const ENTRY_TYPE_CONFIG: Record<
  SupportEntryType,
  { label: string; badgeClass: string; bgClass: string }
> = {
  customer: {
    label: "Customer",
    badgeClass: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    bgClass: "bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-800",
  },
  agent_reply: {
    label: "Agent Reply",
    badgeClass: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    bgClass: "bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-800",
  },
  draft: {
    label: "Draft",
    badgeClass: "bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-200",
    bgClass: "bg-gray-100 border-gray-300 dark:bg-gray-800 dark:border-gray-600",
  },
  context: {
    label: "Internal",
    badgeClass: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
    bgClass: "bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-800",
  },
  system_event: {
    label: "System",
    badgeClass: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-200",
    bgClass: "bg-orange-50 border-orange-200 dark:bg-orange-950 dark:border-orange-800",
  },
};

/** Props for TicketTimeline. */
export interface TicketTimelineProps {
  /** Ordered list of ticket entries to display. */
  entries: SupportEntry[];
  /** Slug of the parent ticket (used for API calls). */
  ticketSlug: string;
  /** Whether the current viewer is a DEFT member. */
  isDeftMember: boolean;
  /** The authenticated user's ID — used to gate draft editing to the author. */
  currentUserId?: string;
  /** Called after an entry is mutated so the parent can reload. */
  onMutated?: () => void;
  /** Resolve a user ID to a display label (e.g. "Alice (DEFT)"). */
  formatUser?: (userId: string) => string;
}

/**
 * TicketTimeline renders the chronological conversation history of a support
 * ticket.
 * — Drafts are editable by their author (not yet immutable).
 * — Drafts can be published by DEFT members OR by the customer who created them.
 * — DEFT-only visibility toggle is restricted to DEFT members.
 */
export function TicketTimeline({
  entries,
  ticketSlug,
  isDeftMember,
  currentUserId,
  onMutated,
  formatUser,
}: TicketTimelineProps): ReactNode {
  const { getToken } = useAuth();
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  // Track which entry is open for inline editing.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editBody, setEditBody] = useState("");

  const handlePublish = useCallback(
    async (entryId: string): Promise<void> => {
      setError(null);
      setBusyId(entryId);
      try {
        const token = await getToken();
        if (!token) return;
        await publishTicketEntry(token, ticketSlug, entryId);
        onMutated?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to publish");
      } finally {
        setBusyId(null);
      }
    },
    [getToken, ticketSlug, onMutated],
  );

  const handleToggleDeftOnly = useCallback(
    async (entryId: string, current: boolean): Promise<void> => {
      setError(null);
      setBusyId(entryId);
      try {
        const token = await getToken();
        if (!token) return;
        await setEntryDeftVisibility(token, ticketSlug, entryId, !current);
        onMutated?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to update visibility");
      } finally {
        setBusyId(null);
      }
    },
    [getToken, ticketSlug, onMutated],
  );

  const handleSaveEdit = useCallback(
    async (entryId: string): Promise<void> => {
      if (!editBody.trim()) return;
      setError(null);
      setBusyId(entryId);
      try {
        const token = await getToken();
        if (!token) return;
        await updateTicketEntry(token, ticketSlug, entryId, editBody);
        setEditingId(null);
        onMutated?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to save draft");
      } finally {
        setBusyId(null);
      }
    },
    [getToken, ticketSlug, editBody, onMutated],
  );

  if (entries.length === 0) {
    return (
      <p
        data-testid="ticket-timeline-empty"
        className="py-6 text-center text-sm text-muted-foreground"
      >
        No entries yet.
      </p>
    );
  }

  return (
    <div data-testid="ticket-timeline" className="flex flex-col gap-3">
      {error && (
        <p data-testid="timeline-error" className="text-sm text-red-600">
          {error}
        </p>
      )}
      {entries.map((entry) => {
        const cfg = ENTRY_TYPE_CONFIG[entry.type] ?? ENTRY_TYPE_CONFIG.customer;
        const isBusy = busyId === entry.id;
        const isMyDraft =
          entry.type === "draft" && !entry.is_immutable && entry.author_id === currentUserId;
        const isEditing = editingId === entry.id;
        // Drafts can be published by DEFT or by the draft's own author.
        const canPublish = entry.type === "draft" && (isDeftMember || isMyDraft);
        const isInternalOnlyType = entry.type === "context" || entry.type === "system_event";
        const canToggleVisibility = isDeftMember && (!entry.is_deft_only || !isInternalOnlyType);

        return (
          <div
            key={entry.id}
            data-testid={`entry-${entry.id}`}
            className={`rounded-lg border p-4 ${cfg.bgClass}`}
          >
            {/* Entry header */}
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <span
                data-testid={`entry-type-badge-${entry.id}`}
                className={`rounded-full px-2 py-0.5 text-xs font-medium ${cfg.badgeClass}`}
              >
                {cfg.label}
              </span>
              {entry.is_deft_only && (
                <span
                  data-testid={`entry-deft-only-badge-${entry.id}`}
                  className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900 dark:text-red-200"
                >
                  DEFT Only
                </span>
              )}
              {entry.is_immutable && (
                <Lock
                  data-testid={`entry-immutable-icon-${entry.id}`}
                  className="h-3 w-3 text-muted-foreground"
                  aria-label="Immutable"
                />
              )}
              {formatUser && (
                <span
                  className="text-xs font-medium text-foreground"
                  data-testid={`entry-author-${entry.id}`}
                >
                  {formatUser(entry.author_id)}
                </span>
              )}
              <span className="ml-auto text-xs text-muted-foreground">
                {new Date(entry.created_at).toLocaleString()}
              </span>
            </div>

            {/* Entry body — either inline editor or rendered HTML */}
            {isEditing ? (
              <div className="flex flex-col gap-2">
                <textarea
                  data-testid={`draft-edit-textarea-${entry.id}`}
                  value={editBody}
                  onChange={(e) => setEditBody(e.target.value)}
                  rows={6}
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none"
                />
                <div className="flex gap-2">
                  <button
                    data-testid={`draft-save-btn-${entry.id}`}
                    onClick={() => void handleSaveEdit(entry.id)}
                    disabled={isBusy || !editBody.trim()}
                    className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                  >
                    {isBusy ? "Saving…" : "Save draft"}
                  </button>
                  <button
                    data-testid={`draft-cancel-btn-${entry.id}`}
                    onClick={() => setEditingId(null)}
                    className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1 text-xs text-muted-foreground hover:bg-accent"
                  >
                    <X className="h-3 w-3" /> Cancel
                  </button>
                </div>
              </div>
            ) : (
              <div
                data-testid={`entry-body-${entry.id}`}
                className="support-entry-body prose prose-sm dark:prose-invert max-w-none text-foreground"
                dangerouslySetInnerHTML={{ __html: entry.body }}
              />
            )}

            {/* Per-entry action bar */}
            <div className="mt-3 flex flex-wrap gap-2">
              {/* Edit own draft (any user can edit their own drafts) */}
              {isMyDraft && !isEditing && (
                <button
                  data-testid={`draft-edit-btn-${entry.id}`}
                  onClick={() => {
                    setEditingId(entry.id);
                    setEditBody(entry.body);
                  }}
                  className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-accent"
                >
                  <Pencil className="h-3 w-3" /> Edit draft
                </button>
              )}

              {/* Publish draft — visible to DEFT or to the draft’s own author */}
              {canPublish && !isEditing && (
                <button
                  data-testid={`publish-btn-${entry.id}`}
                  onClick={() => void handlePublish(entry.id)}
                  disabled={isBusy}
                  className="inline-flex items-center gap-1 rounded-md bg-green-600 px-3 py-1 text-xs font-medium text-white hover:bg-green-700 disabled:opacity-50"
                >
                  <Send className="h-3 w-3" />
                  {isBusy ? "Publishing…" : "Send"}
                </button>
              )}

              {/* Toggle DEFT-only — DEFT members only */}
              {canToggleVisibility && (
                <button
                  data-testid={`deft-only-btn-${entry.id}`}
                  onClick={() => void handleToggleDeftOnly(entry.id, entry.is_deft_only)}
                  disabled={isBusy}
                  className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-accent disabled:opacity-50"
                  title={entry.is_deft_only ? "Make visible to customer" : "Hide from customer"}
                >
                  {entry.is_deft_only ? (
                    <Eye className="h-3 w-3" />
                  ) : (
                    <EyeOff className="h-3 w-3" />
                  )}
                  {entry.is_deft_only ? "Unhide" : "Hide"}
                </button>
              )}
              {isDeftMember && !canToggleVisibility && entry.is_deft_only && (
                <span
                  data-testid={`deft-only-locked-${entry.id}`}
                  className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground"
                  title="Internal DEFT-only entries cannot be unhidden"
                >
                  <Lock className="h-3 w-3" />
                  Internal only
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
