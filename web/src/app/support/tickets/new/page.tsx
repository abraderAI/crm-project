"use client";

import { useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth, useUser } from "@clerk/nextjs";
import { AlertCircle, AlertTriangle, ArrowLeft, LifeBuoy } from "lucide-react";

import { createSupportTicket } from "@/lib/global-api";
import { useTier } from "@/hooks/use-tier";
import { MessageEditor } from "@/components/editor/message-editor";
import { Mail } from "lucide-react";

/**
 * New support ticket creation page at /support/tickets/new.
 *
 * Pre-populates the creator field from the authenticated session.
 * On submit, creates the ticket and redirects to the full ticket editor.
 */
export default function NewTicketPage(): ReactNode {
  const router = useRouter();
  const { getToken } = useAuth();
  const { user } = useUser();
  const { orgId, tier } = useTier();

  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [orphanWarning, setOrphanWarning] = useState("");

  const creatorName = user?.fullName ?? user?.primaryEmailAddress?.emailAddress ?? "You";

  const handleCreate = async (): Promise<void> => {
    if (!title.trim()) return;
    setError("");
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      const ticket = await createSupportTicket(token, {
        title: title.trim(),
        body: body.trim() || undefined,
        org_id: orgId ?? undefined,
        contact_email: contactEmail.trim() || undefined,
      });
      // If a contact email was provided but the returned ticket's author is
      // still the current user, the email wasn't matched to a registered account.
      // Show a warning before redirecting so the agent is aware.
      const emailProvided = contactEmail.trim() !== "";
      const emailNotResolved = emailProvided && ticket.author_id === user?.id;
      if (emailNotResolved) {
        setOrphanWarning(
          `"${contactEmail.trim()}" is not registered yet. The ticket has been saved with that email ` +
            `and will be linked to them automatically when they sign up.`,
        );
        // Delay redirect so the agent can read the warning.
        setTimeout(() => router.push(`/support/tickets/${ticket.slug}`), 4000);
      } else {
        router.push(`/support/tickets/${ticket.slug}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create ticket");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div data-testid="new-ticket-page" className="mx-auto max-w-2xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex flex-col gap-2">
        <Link
          href="/support"
          data-testid="back-to-support-link"
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3 w-3" />
          Back to DEFT.support
        </Link>
        <div className="flex items-center gap-3">
          <LifeBuoy className="h-6 w-6 text-primary" />
          <h1 className="text-xl font-semibold text-foreground">New Support Ticket</h1>
        </div>
      </div>

      <div className="flex flex-col gap-5 rounded-lg border border-border bg-background p-6">
        {/* Creator field (read-only, auto-populated) */}
        <div>
          <label className="text-xs font-medium text-foreground">From</label>
          <p
            data-testid="ticket-creator-display"
            className="mt-1 rounded-md border border-border bg-muted px-3 py-2 text-sm text-muted-foreground"
          >
            {creatorName}
          </p>
        </div>

        {/* Customer email — DEFT members only (tier >= 4) */}
        {tier >= 4 && (
          <div>
            <label htmlFor="new-ticket-email" className="text-xs font-medium text-foreground">
              <span className="flex items-center gap-1">
                <Mail className="h-3 w-3" /> Customer email
              </span>
            </label>
            <input
              id="new-ticket-email"
              data-testid="new-ticket-email"
              type="email"
              value={contactEmail}
              onChange={(e) => setContactEmail(e.target.value)}
              placeholder="customer@example.com"
              className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              Assign this ticket to a customer by email. If they haven&apos;t registered yet, the
              ticket will be linked when they sign up.
            </p>
          </div>
        )}

        {/* Title */}
        <div>
          <label htmlFor="new-ticket-title" className="text-xs font-medium text-foreground">
            Subject <span className="text-red-500">*</span>
          </label>
          <input
            id="new-ticket-title"
            data-testid="new-ticket-title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Briefly describe your issue"
            className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Description — half-page rich editor */}
        <div>
          <label className="text-xs font-medium text-foreground">Description</label>
          <div className="mt-1">
            <MessageEditor
              initialContent={body}
              onSubmit={() => void handleCreate()}
              onChange={setBody}
              placeholder="Describe your issue in detail — include steps to reproduce, screenshots, or any relevant context."
              disabled={submitting}
              showSubmit={false}
              editorMinHeight="40vh"
            />
          </div>
        </div>

        {/* Orphan email warning */}
        {orphanWarning && (
          <div
            data-testid="orphan-email-warning"
            className="flex items-start gap-2 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/40 dark:text-amber-200"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>{orphanWarning} Redirecting to ticket…</span>
          </div>
        )}

        {/* Error */}
        {error && (
          <div
            data-testid="new-ticket-error"
            className="flex items-center gap-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700"
          >
            <AlertTriangle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <button
            onClick={() => router.back()}
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
          >
            Cancel
          </button>
          <button
            data-testid="new-ticket-submit"
            onClick={() => void handleCreate()}
            disabled={submitting || !title.trim()}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {submitting ? "Creating…" : "Open ticket"}
          </button>
        </div>
      </div>
    </div>
  );
}
