"use client";

import { useState } from "react";
import { Plus, Pencil, Trash2, Mail, CheckCircle, XCircle } from "lucide-react";
import { createEmailInbox, deleteEmailInbox, updateEmailInbox } from "@/lib/admin-api";
import { EmailInboxForm } from "./email-inbox-form";
import type { EmailInbox, EmailInboxInput } from "@/lib/api-types";

const ROUTING_LABELS: Record<string, string> = {
  support_ticket: "Support Ticket",
  sales_lead: "Sales Lead",
  general: "General Message",
};

interface EmailInboxListProps {
  /** Organisation ID for API calls. */
  orgId: string;
  /** Initial inbox list (fetched server-side). */
  initialInboxes: EmailInbox[];
}

/**
 * Displays the list of configured email inboxes and provides add / edit / delete
 * actions. State is managed client-side after the initial server-side fetch.
 */
export function EmailInboxList({ orgId, initialInboxes }: EmailInboxListProps): React.ReactNode {
  const [inboxes, setInboxes] = useState<EmailInbox[]>(initialInboxes);
  const [showForm, setShowForm] = useState(false);
  const [editingInbox, setEditingInbox] = useState<EmailInbox | undefined>(undefined);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [listError, setListError] = useState("");

  const handleAdd = (): void => {
    setEditingInbox(undefined);
    setShowForm(true);
  };

  const handleEdit = (inbox: EmailInbox): void => {
    setEditingInbox(inbox);
    setShowForm(true);
  };

  const handleCancel = (): void => {
    setShowForm(false);
    setEditingInbox(undefined);
  };

  const handleSave = async (input: EmailInboxInput): Promise<void> => {
    if (editingInbox) {
      const updated = await updateEmailInbox(orgId, editingInbox.id, input);
      setInboxes((prev) => prev.map((i) => (i.id === updated.id ? updated : i)));
    } else {
      const created = await createEmailInbox(orgId, input);
      setInboxes((prev) => [...prev, created]);
    }
    setShowForm(false);
    setEditingInbox(undefined);
  };

  const handleDelete = async (inbox: EmailInbox): Promise<void> => {
    if (!confirm(`Delete inbox "${inbox.name}"? This will stop monitoring that address.`)) return;

    setDeleting(inbox.id);
    setListError("");
    try {
      await deleteEmailInbox(orgId, inbox.id);
      setInboxes((prev) => prev.filter((i) => i.id !== inbox.id));
    } catch (err) {
      setListError(err instanceof Error ? err.message : "Failed to delete inbox.");
    } finally {
      setDeleting(null);
    }
  };

  return (
    <div data-testid="email-inbox-list" className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold text-foreground">Email Inboxes</h2>
          <p className="text-sm text-muted-foreground">
            Each inbox monitors an IMAP address and routes incoming emails to the configured
            destination.
          </p>
        </div>
        <button
          type="button"
          onClick={handleAdd}
          data-testid="add-inbox-btn"
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Add Inbox
        </button>
      </div>

      {listError && (
        <p className="text-sm text-destructive" data-testid="inbox-list-error">
          {listError}
        </p>
      )}

      {showForm && (
        <EmailInboxForm inbox={editingInbox} onSave={handleSave} onCancel={handleCancel} />
      )}

      {inboxes.length === 0 && !showForm ? (
        <div
          className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border py-10 text-center"
          data-testid="inbox-empty-state"
        >
          <Mail className="h-8 w-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No email inboxes configured.</p>
          <button
            type="button"
            onClick={handleAdd}
            className="text-sm text-primary underline-offset-2 hover:underline"
          >
            Add your first inbox
          </button>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {inboxes.map((inbox) => (
            <div
              key={inbox.id}
              data-testid={`inbox-row-${inbox.id}`}
              className="flex items-center justify-between rounded-lg border border-border bg-background px-4 py-3"
            >
              <div className="flex items-center gap-3">
                {inbox.enabled ? (
                  <CheckCircle className="h-4 w-4 flex-shrink-0 text-green-500" />
                ) : (
                  <XCircle className="h-4 w-4 flex-shrink-0 text-muted-foreground" />
                )}
                <div>
                  <p className="text-sm font-medium text-foreground">{inbox.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {inbox.email_address || inbox.username} &middot;{" "}
                    {ROUTING_LABELS[inbox.routing_action] ?? inbox.routing_action}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => handleEdit(inbox)}
                  data-testid={`edit-inbox-${inbox.id}`}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                  aria-label={`Edit ${inbox.name}`}
                >
                  <Pencil className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(inbox)}
                  disabled={deleting === inbox.id}
                  data-testid={`delete-inbox-${inbox.id}`}
                  className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
                  aria-label={`Delete ${inbox.name}`}
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
