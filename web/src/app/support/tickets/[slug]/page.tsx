"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { LifeBuoy } from "lucide-react";

import type { SupportEntry, ThreadWithAuthor } from "@/lib/api-types";
import { fetchSupportTicket } from "@/lib/global-api";
import { fetchTicketEntries } from "@/lib/support-api";
import { useTier } from "@/hooks/use-tier";
import { ContentEditorLayout } from "@/components/editor/content-editor-layout";
import { TicketTimeline } from "@/components/support/ticket-timeline";
import { TicketEntryComposer } from "@/components/support/ticket-entry-composer";
import { NotificationPrefs } from "@/components/support/notification-prefs";

/** Parse notification_detail_level from thread metadata JSON. */
function parseNotifLevel(metadata: string): "full" | "privacy" {
  try {
    const parsed = JSON.parse(metadata) as Record<string, unknown>;
    if (parsed["notification_detail_level"] === "privacy") return "privacy";
  } catch {
    /* ignore */
  }
  return "full";
}

/** Parse ticket status from thread metadata JSON. */
function parseStatus(metadata: string): string {
  try {
    const parsed = JSON.parse(metadata) as Record<string, unknown>;
    if (typeof parsed["status"] === "string") return parsed["status"];
  } catch {
    /* ignore */
  }
  return "open";
}

const STATUS_BADGE: Record<string, string> = {
  open: "bg-yellow-100 text-yellow-800",
  pending: "bg-blue-100 text-blue-800",
  resolved: "bg-green-100 text-green-800",
  closed: "bg-gray-100 text-gray-800",
};

/**
 * Ticket detail page — full conversation view for a single support ticket.
 * Accessible at /support/tickets/[slug].
 */
export default function TicketDetailPage(): ReactNode {
  const { slug } = useParams<{ slug: string }>();
  const { getToken } = useAuth();
  const { tier } = useTier();

  const [ticket, setTicket] = useState<ThreadWithAuthor | null>(null);
  const [entries, setEntries] = useState<SupportEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  // Tier 4+ (DEFT employees, platform admins) are considered DEFT members.
  const isDeftMember = tier >= 4;

  const loadData = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const token = await getToken();
      if (!token || !mountedRef.current) return;

      const [ticketData, entriesData] = await Promise.all([
        fetchSupportTicket(token, slug),
        fetchTicketEntries(token, slug),
      ]);
      if (!mountedRef.current) return;
      setTicket(ticketData);
      setEntries(entriesData);
    } catch (err) {
      if (mountedRef.current) {
        setError(err instanceof Error ? err.message : "Failed to load ticket");
      }
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, [getToken, slug]);

  useEffect(() => {
    mountedRef.current = true;
    void loadData();
    return () => {
      mountedRef.current = false;
    };
  }, [loadData]);

  if (loading) {
    return (
      <div data-testid="ticket-detail-loading" className="py-12 text-center text-sm text-muted-foreground">
        Loading ticket…
      </div>
    );
  }

  if (error || !ticket) {
    return (
      <div data-testid="ticket-detail-error" className="py-12 text-center text-sm text-red-600">
        {error ?? "Ticket not found."}
      </div>
    );
  }

  const status = parseStatus(ticket.metadata);
  const badgeClass = STATUS_BADGE[status] ?? STATUS_BADGE["open"];
  const notifLevel = parseNotifLevel(ticket.metadata);

  return (
    <div data-testid="ticket-detail-page" className="mx-auto max-w-5xl px-4 py-6">
      <ContentEditorLayout
        header={
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="flex min-w-0 flex-col gap-1">
              {/* Ticket # and title */}
              <div className="flex items-center gap-2">
                <LifeBuoy className="h-5 w-5 shrink-0 text-primary" />
                {ticket.ticket_number ? (
                  <span
                    data-testid="ticket-number"
                    className="shrink-0 rounded-md bg-primary/10 px-2 py-0.5 text-sm font-mono font-semibold text-primary"
                  >
                    #{ticket.ticket_number}
                  </span>
                ) : null}
                <h1
                  data-testid="ticket-title"
                  className="truncate text-lg font-semibold text-foreground"
                >
                  {ticket.title}
                </h1>
              </div>
              {/* Author + org */}
              <p className="text-xs text-muted-foreground">
                {ticket.author_name ?? ticket.author_email ?? ticket.author_id}
                {ticket.org_name ? ` · ${ticket.org_name}` : ""}
              </p>
            </div>
            {/* Status badge */}
            <span
              data-testid="ticket-status-badge"
              className={`shrink-0 rounded-full px-3 py-1 text-xs font-medium ${badgeClass}`}
            >
              {status}
            </span>
          </div>
        }
        sidebar={
          <div className="flex flex-col gap-4">
            {/* Notification preferences — visible to ticket creator / customer */}
            {!isDeftMember && (
              <NotificationPrefs ticketSlug={slug} currentLevel={notifLevel} />
            )}

            {/* Metadata card */}
            <div className="rounded-lg border border-border bg-background p-4">
              <h4 className="mb-2 text-sm font-semibold text-foreground">Ticket info</h4>
              <dl className="flex flex-col gap-1 text-xs">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Created</dt>
                  <dd>{new Date(ticket.created_at).toLocaleDateString()}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Updated</dt>
                  <dd>{new Date(ticket.updated_at).toLocaleDateString()}</dd>
                </div>
                {ticket.ticket_number ? (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">Ticket #</dt>
                    <dd className="font-mono font-semibold">#{ticket.ticket_number}</dd>
                  </div>
                ) : null}
              </dl>
            </div>
          </div>
        }
        composer={
          <TicketEntryComposer
            ticketSlug={slug}
            isDeftMember={isDeftMember}
            onCreated={() => void loadData()}
          />
        }
      >
        <TicketTimeline
          entries={entries}
          ticketSlug={slug}
          isDeftMember={isDeftMember}
          onMutated={() => void loadData()}
        />
      </ContentEditorLayout>
    </div>
  );
}
