"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { ArrowLeft, LifeBuoy, Paperclip } from "lucide-react";

import type { SupportEntry, ThreadWithAuthor, Upload } from "@/lib/api-types";
import {
  fetchSupportTicket,
  updateSupportTicket,
  fetchThreadAttachments,
  uploadThreadAttachment,
  downloadUpload,
} from "@/lib/global-api";
import { fetchTicketEntries, createTicketEntry } from "@/lib/support-api";
import { useTier } from "@/hooks/use-tier";
import { useUser } from "@clerk/nextjs";
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

const STATUS_OPTIONS = [
  { value: "open", label: "Open" },
  { value: "pending", label: "Pending" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

/**
 * Ticket detail page — full conversation view for a single support ticket.
 * Accessible at /support/tickets/[slug].
 */
export default function TicketDetailPage(): ReactNode {
  const { slug } = useParams<{ slug: string }>();
  const { getToken } = useAuth();
  const { user } = useUser();
  const { tier } = useTier();

  const [ticket, setTicket] = useState<ThreadWithAuthor | null>(null);
  const [entries, setEntries] = useState<SupportEntry[]>([]);
  const [attachments, setAttachments] = useState<Upload[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Status picker state (DEFT-only).
  const [statusSaving, setStatusSaving] = useState(false);
  const [statusError, setStatusError] = useState("");

  // Attachment upload state.
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const mountedRef = useRef(true);

  // Tier 4+ (DEFT employees, platform admins) are considered DEFT members.
  const isDeftMember = tier >= 4;
  const currentUserId = user?.id ?? "";

  const loadData = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const token = await getToken();
      if (!token || !mountedRef.current) return;

      const [ticketData, entriesData, attachData] = await Promise.all([
        fetchSupportTicket(token, slug),
        fetchTicketEntries(token, slug),
        fetchThreadAttachments(token, slug),
      ]);
      if (!mountedRef.current) return;
      setTicket(ticketData);
      setEntries(entriesData);
      setAttachments(attachData);
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

  /** DEFT-only: change status and auto-insert a system_event entry. */
  const handleStatusChange = async (newStatus: string): Promise<void> => {
    if (!ticket || newStatus === parseStatus(ticket.metadata)) return;
    setStatusError("");
    setStatusSaving(true);
    try {
      const token = await getToken();
      if (!token) return;
      await updateSupportTicket(token, slug, { status: newStatus });
      // Auto-create a system_event entry recording the change.
      const now = new Date().toLocaleString();
      await createTicketEntry(token, slug, {
        type: "system_event",
        body: `<p>Status changed to <strong>${newStatus}</strong> — ${now}</p>`,
        is_deft_only: false,
      });
      await loadData();
    } catch (err) {
      setStatusError(err instanceof Error ? err.message : "Failed to update status");
    } finally {
      setStatusSaving(false);
    }
  };

  /** Sidebar attachment upload. */
  const handleAttachFile = async (e: React.ChangeEvent<HTMLInputElement>): Promise<void> => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (fileInputRef.current) fileInputRef.current.value = "";
    setUploadError("");
    setUploading(true);
    try {
      const token = await getToken();
      if (!token) return;
      await uploadThreadAttachment(token, slug, file);
      await loadData();
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (uploadId: string, filename: string): Promise<void> => {
    const token = await getToken();
    if (!token) return;
    await downloadUpload(token, uploadId, filename);
  };

  if (loading) {
    return (
      <div
        data-testid="ticket-detail-loading"
        className="py-12 text-center text-sm text-muted-foreground"
      >
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
              <Link
                href="/support"
                data-testid="back-to-support-link"
                className="mb-1 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
              >
                <ArrowLeft className="h-3 w-3" />
                Back to DEFT.support
              </Link>
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
              <p className="text-xs text-muted-foreground">
                {ticket.author_name ?? ticket.author_email ?? ticket.author_id}
                {ticket.org_name ? ` · ${ticket.org_name}` : ""}
              </p>
            </div>

            {/* Status — picker for DEFT, read-only badge for customers */}
            {isDeftMember ? (
              <div className="flex flex-col items-end gap-1">
                <select
                  data-testid="status-picker"
                  value={status}
                  onChange={(e) => void handleStatusChange(e.target.value)}
                  disabled={statusSaving}
                  className="h-8 rounded-md border border-border bg-background px-2 text-xs font-medium focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {STATUS_OPTIONS.map(({ value, label }) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
                {statusError && <p className="text-xs text-red-600">{statusError}</p>}
              </div>
            ) : (
              <span
                data-testid="ticket-status-badge"
                className={`shrink-0 rounded-full px-3 py-1 text-xs font-medium ${badgeClass}`}
              >
                {status}
              </span>
            )}
          </div>
        }
        sidebar={
          <div className="flex flex-col gap-4">
            {/* Notification preferences — for ticket owner/customer */}
            {!isDeftMember && <NotificationPrefs ticketSlug={slug} currentLevel={notifLevel} />}

            {/* Ticket metadata */}
            <div className="rounded-lg border border-border bg-background p-4">
              <h4 className="mb-2 text-sm font-semibold text-foreground">Ticket info</h4>
              <dl className="flex flex-col gap-1 text-xs">
                {ticket.ticket_number ? (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">Ticket #</dt>
                    <dd className="font-mono font-semibold">#{ticket.ticket_number}</dd>
                  </div>
                ) : null}
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Status</dt>
                  <dd className="capitalize">{status}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Created</dt>
                  <dd>{new Date(ticket.created_at).toLocaleDateString()}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Updated</dt>
                  <dd>{new Date(ticket.updated_at).toLocaleDateString()}</dd>
                </div>
              </dl>
            </div>

            {/* Attachments */}
            <div className="rounded-lg border border-border bg-background p-4">
              <div className="mb-2 flex items-center justify-between">
                <h4 className="text-sm font-semibold text-foreground">Attachments</h4>
                <label
                  htmlFor="sidebar-file-input"
                  data-testid="sidebar-attach-btn"
                  className="inline-flex cursor-pointer items-center gap-1 rounded-md border border-border px-2 py-1 text-xs hover:bg-accent"
                >
                  <Paperclip className="h-3 w-3" />
                  {uploading ? "Uploading…" : "Add file"}
                </label>
                <input
                  id="sidebar-file-input"
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  onChange={(e) => void handleAttachFile(e)}
                  disabled={uploading}
                />
              </div>
              {uploadError && <p className="mb-1 text-xs text-red-600">{uploadError}</p>}
              {attachments.length === 0 ? (
                <p className="text-xs text-muted-foreground">No attachments.</p>
              ) : (
                <ul data-testid="sidebar-attachments" className="space-y-1">
                  {attachments.map((a) => (
                    <li key={a.id} className="flex items-center gap-1 text-xs">
                      <Paperclip className="h-3 w-3 shrink-0 text-muted-foreground" />
                      <button
                        data-testid={`attachment-download-${a.id}`}
                        onClick={() => void handleDownload(a.id, a.filename)}
                        className="truncate text-foreground underline hover:no-underline"
                      >
                        {a.filename}
                      </button>
                      <span className="shrink-0 text-muted-foreground">
                        ({(a.size / 1024).toFixed(1)} KB)
                      </span>
                    </li>
                  ))}
                </ul>
              )}
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
          currentUserId={currentUserId}
          onMutated={() => void loadData()}
        />
      </ContentEditorLayout>
    </div>
  );
}
