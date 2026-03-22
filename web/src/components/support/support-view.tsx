"use client";

import { useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle, LifeBuoy, Plus, X } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import { createSupportTicket } from "@/lib/global-api";

/** Badge styles keyed by ticket status. */
const STATUS_STYLES: Record<string, string> = {
  open: "bg-yellow-100 text-yellow-800",
  assigned: "bg-purple-100 text-purple-800",
  pending: "bg-blue-100 text-blue-800",
  resolved: "bg-green-100 text-green-800",
  closed: "bg-gray-100 text-gray-800",
};

export interface SupportViewProps {
  /** Initial list of support tickets (from SSR). */
  initialTickets: Thread[];
}

/**
 * Support ticket list with inline create form.
 * Displays the user's tickets with status badges; lets them submit new tickets.
 */
export function SupportView({ initialTickets }: SupportViewProps): ReactNode {
  const { getToken } = useAuth();

  const [tickets, setTickets] = useState<Thread[]>(initialTickets);
  const [showCreate, setShowCreate] = useState(false);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  /** Submit a new support ticket and prepend it to the list on success. */
  const handleCreate = async (): Promise<void> => {
    if (!title.trim()) return;
    setError("");
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      const newTicket = await createSupportTicket(token, {
        title: title.trim(),
        body: body.trim() || undefined,
      });
      setTickets((prev) => [newTicket, ...prev]);
      setTitle("");
      setBody("");
      setShowCreate(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create ticket");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div data-testid="support-view" className="flex flex-col gap-4">
      {/* Error banner */}
      {error && (
        <div
          data-testid="support-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Header + new ticket toggle */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">Support Tickets</h2>
        <button
          data-testid="new-ticket-btn"
          onClick={() => setShowCreate((v) => !v)}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          {showCreate ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          {showCreate ? "Cancel" : "New Ticket"}
        </button>
      </div>

      {/* Inline create form */}
      {showCreate && (
        <div
          data-testid="create-ticket-form"
          className="rounded-lg border border-border bg-background p-4"
        >
          <h3 className="mb-3 text-sm font-semibold text-foreground">New Support Ticket</h3>
          <div className="flex flex-col gap-3">
            <div>
              <label htmlFor="ticket-title" className="text-xs font-medium text-foreground">
                Title <span className="text-red-500">*</span>
              </label>
              <input
                id="ticket-title"
                data-testid="ticket-title-input"
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Describe your issue"
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label htmlFor="ticket-body" className="text-xs font-medium text-foreground">
                Details
              </label>
              <textarea
                id="ticket-body"
                data-testid="ticket-body-input"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder="Additional details (optional)"
                rows={3}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div className="flex gap-2">
              <button
                data-testid="ticket-submit-btn"
                onClick={() => void handleCreate()}
                disabled={submitting || !title.trim()}
                className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {submitting ? "Submitting..." : "Submit Ticket"}
              </button>
              <button
                data-testid="ticket-cancel-btn"
                onClick={() => {
                  setShowCreate(false);
                  setTitle("");
                  setBody("");
                }}
                className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Ticket list */}
      {tickets.length === 0 ? (
        <p data-testid="support-empty" className="text-sm text-muted-foreground">
          No support tickets yet. Use &quot;New Ticket&quot; if you need help.
        </p>
      ) : (
        <ul
          data-testid="support-ticket-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {tickets.map((ticket) => {
            const status = ticket.status ?? "open";
            const badgeClass = STATUS_STYLES[status] ?? STATUS_STYLES["open"];
            return (
              <li
                key={ticket.id}
                data-testid={`ticket-row-${ticket.id}`}
                className="flex items-center justify-between px-4 py-3"
              >
                <div className="flex items-start gap-2">
                  <LifeBuoy className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                  <span className="text-sm text-foreground">{ticket.title}</span>
                </div>
                <span
                  data-testid={`ticket-status-${ticket.id}`}
                  className={`rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}
                >
                  {status}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
