"use client";

import { useCallback, useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { Eye, EyeOff, Lock, Send } from "lucide-react";

import type { SupportEntry, SupportEntryType } from "@/lib/api-types";
import {
  publishTicketEntry,
  setEntryDeftVisibility,
} from "@/lib/support-api";

/** Visual config for each entry type. */
const ENTRY_TYPE_CONFIG: Record<
  SupportEntryType,
  { label: string; badgeClass: string; bgClass: string }
> = {
  customer: {
    label: "Customer",
    badgeClass: "bg-blue-100 text-blue-800",
    bgClass: "bg-blue-50 border-blue-200",
  },
  agent_reply: {
    label: "Agent Reply",
    badgeClass: "bg-green-100 text-green-800",
    bgClass: "bg-green-50 border-green-200",
  },
  draft: {
    label: "Draft",
    badgeClass: "bg-yellow-100 text-yellow-800",
    bgClass: "bg-yellow-50 border-yellow-200",
  },
  context: {
    label: "Internal",
    badgeClass: "bg-purple-100 text-purple-800",
    bgClass: "bg-purple-50 border-purple-200",
  },
  system_event: {
    label: "System",
    badgeClass: "bg-gray-100 text-gray-600",
    bgClass: "bg-gray-50 border-gray-200",
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
  /** Called after an entry is mutated so the parent can reload. */
  onMutated?: () => void;
}

/**
 * TicketTimeline renders the chronological conversation history of a support
 * ticket. Entries are immutable once posted except for draft bodies; DEFT
 * members may publish drafts and toggle DEFT-only visibility on any entry.
 */
export function TicketTimeline({
  entries,
  ticketSlug,
  isDeftMember,
  onMutated,
}: TicketTimelineProps): ReactNode {
  const { getToken } = useAuth();
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

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
                  className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700"
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
              <span className="ml-auto text-xs text-muted-foreground">
                {new Date(entry.created_at).toLocaleString()}
              </span>
            </div>

            {/* Entry body (rendered HTML from Tiptap) */}
            <div
              data-testid={`entry-body-${entry.id}`}
              className="prose prose-sm max-w-none"
              dangerouslySetInnerHTML={{ __html: entry.body }}
            />

            {/* DEFT-member actions */}
            {isDeftMember && (
              <div className="mt-3 flex gap-2">
                {/* Publish draft */}
                {entry.type === "draft" && (
                  <button
                    data-testid={`publish-btn-${entry.id}`}
                    onClick={() => void handlePublish(entry.id)}
                    disabled={isBusy}
                    className="inline-flex items-center gap-1 rounded-md bg-green-600 px-3 py-1 text-xs font-medium text-white hover:bg-green-700 disabled:opacity-50"
                  >
                    <Send className="h-3 w-3" />
                    {isBusy ? "Publishing…" : "Publish"}
                  </button>
                )}

                {/* Toggle DEFT-only */}
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
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
