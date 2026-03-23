"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { ArrowLeft, LifeBuoy, Paperclip } from "lucide-react";

import type { DeftMember, SupportEntry, ThreadWithAuthor, Upload } from "@/lib/api-types";
import {
  fetchSupportTicket,
  updateSupportTicket,
  fetchThreadAttachments,
  uploadThreadAttachment,
  downloadUpload,
} from "@/lib/global-api";
import { fetchTicketEntries, createTicketEntry, fetchDeftMembers } from "@/lib/support-api";
import { useTier } from "@/hooks/use-tier";
import { useUser } from "@clerk/nextjs";
import { useUserDirectory } from "@/lib/use-user-directory";
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

/** Parse assigned_to from thread metadata JSON. */
function parseAssignedTo(metadata: string): string {
  try {
    const parsed = JSON.parse(metadata) as Record<string, unknown>;
    if (typeof parsed["assigned_to"] === "string") return parsed["assigned_to"];
  } catch {
    /* ignore */
  }
  return "";
}

const STATUS_BADGE: Record<string, string> = {
  open: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  assigned: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  resolved: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  closed: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200",
};

const STATUS_OPTIONS = [
  { value: "open", label: "Open" },
  { value: "assigned", label: "Assigned" },
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

  // Assignee picker state (DEFT-only).
  const [deftMembers, setDeftMembers] = useState<DeftMember[]>([]);
  const [assigneeSaving, setAssigneeSaving] = useState(false);
  const [assigneeError, setAssigneeError] = useState("");

  // Attachment upload state.
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const mountedRef = useRef(true);

  // Tier 4+ (DEFT employees, platform admins) are considered DEFT members.
  const isDeftMember = tier >= 4;
  const currentUserId = user?.id ?? "";
  const userDir = useUserDirectory();

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

  // Fetch DEFT members for the assignee picker (DEFT-only).
  useEffect(() => {
    if (!isDeftMember) return;
    void (async () => {
      try {
        const token = await getToken();
        if (!token) return;
        const members = await fetchDeftMembers(token);
        if (mountedRef.current) setDeftMembers(members);
      } catch {
        // Silently fall back to empty list.
      }
    })();
  }, [isDeftMember, getToken]);

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

  /** DEFT-only: change assignee and auto-insert a system_event entry. */
  const handleAssigneeChange = async (newAssignee: string): Promise<void> => {
    if (!ticket || newAssignee === parseAssignedTo(ticket.metadata)) return;
    setAssigneeError("");
    setAssigneeSaving(true);
    try {
      const token = await getToken();
      if (!token) return;
      await updateSupportTicket(token, slug, { assigned_to: newAssignee });
      const assigneeName = newAssignee
        ? (deftMembers.find((m) => m.user_id === newAssignee)?.display_name ?? newAssignee)
        : "Unassigned";
      const now = new Date().toLocaleString();
      await createTicketEntry(token, slug, {
        type: "system_event",
        body: `<p>Assigned to <strong>${assigneeName}</strong> — ${now}</p>`,
        is_deft_only: false,
      });
      await loadData();
    } catch (err) {
      setAssigneeError(err instanceof Error ? err.message : "Failed to update assignee");
    } finally {
      setAssigneeSaving(false);
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
  const assignedTo = parseAssignedTo(ticket.metadata);

  /**
   * Requestor label — prioritises contact_email (the address a DEFT member
   * entered when creating the ticket on behalf of someone else), then falls
   * back through the author resolution chain.
   */
  const requestorLabel =
    ticket.contact_email ??
    ticket.author_name ??
    ticket.author_email ??
    ticket.author_id;

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
              <p
                data-testid="ticket-requestor-label"
                className="text-xs text-muted-foreground"
              >
                {requestorLabel}
                {ticket.org_name ? ` · ${ticket.org_name}` : ""}
              </p>
            </div>

            {/* Status — picker for DEFT, read-only badge for requestors */}
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
            {/* Notification preferences — for ticket owner/requestor */}
            {!isDeftMember && <NotificationPrefs ticketSlug={slug} currentLevel={notifLevel} />}

            {/* Assignee picker — DEFT-only */}
            {isDeftMember && (
              <div className="rounded-lg border border-border bg-background p-4">
                <h4 className="mb-2 text-sm font-semibold text-foreground">Assignee</h4>
                <select
                  data-testid="assignee-picker"
                  value={assignedTo}
                  onChange={(e) => void handleAssigneeChange(e.target.value)}
                  disabled={assigneeSaving}
                  className="h-8 w-full rounded-md border border-border bg-background px-2 text-xs font-medium focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="">Unassigned</option>
                  {deftMembers.map((m) => (
                    <option key={m.user_id} value={m.user_id}>
                      {m.display_name || m.email}
                    </option>
                  ))}
                </select>
                {assigneeError && <p className="mt-1 text-xs text-red-600">{assigneeError}</p>}
              </div>
            )}

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
                  <dt className="text-muted-foreground">Requestor</dt>
                  <dd
                    data-testid="sidebar-requestor-value"
                    className="truncate text-right"
                  >
                    {requestorLabel}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Status</dt>
                  <dd className="capitalize">{status}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Assignee</dt>
                  <dd>{assignedTo ? userDir.format(assignedTo) : "Unassigned"}</dd>
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
          formatUser={userDir.format}
          resolveUser={userDir.resolve}
        />
      </ContentEditorLayout>
    </div>
  );
}
