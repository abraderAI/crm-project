"use client";

import { useState, type ReactNode } from "react";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle, Send } from "lucide-react";

import type { SupportEntryType } from "@/lib/api-types";
import { createTicketEntry } from "@/lib/support-api";
import { MessageEditor } from "@/components/editor/message-editor";

/** Entry type options visible to DEFT agents. */
const DEFT_ENTRY_TYPES: { value: SupportEntryType; label: string; description: string }[] = [
  { value: "agent_reply", label: "Reply (publish now)", description: "Immediately visible to the customer." },
  { value: "draft", label: "Draft", description: "Saved as a draft — invisible to the customer until published." },
  { value: "context", label: "Internal context", description: "DEFT-only. Never visible to the customer." },
  { value: "customer", label: "Customer message", description: "Add a customer message on their behalf." },
  { value: "system_event", label: "System event", description: "Record a system event (e.g. escalation note)." },
];

/** Props for TicketEntryComposer. */
export interface TicketEntryComposerProps {
  /** Slug of the parent ticket. */
  ticketSlug: string;
  /** Whether the current user is a DEFT member (shows advanced type options). */
  isDeftMember: boolean;
  /** Called after a new entry is successfully created. */
  onCreated?: () => void;
}

/**
 * TicketEntryComposer renders the rich-text input area for adding new entries
 * to a support ticket. DEFT members can choose the entry type; customers are
 * limited to the "customer" type.
 */
export function TicketEntryComposer({
  ticketSlug,
  isDeftMember,
  onCreated,
}: TicketEntryComposerProps): ReactNode {
  const { getToken } = useAuth();
  const [entryType, setEntryType] = useState<SupportEntryType>(
    isDeftMember ? "agent_reply" : "customer",
  );
  const [isDeftOnly, setIsDeftOnly] = useState(false);
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (): Promise<void> => {
    if (!body.trim()) return;
    setError("");
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      await createTicketEntry(token, ticketSlug, {
        type: entryType,
        body,
        is_deft_only: isDeftOnly || entryType === "context",
      });
      setBody("");
      onCreated?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div data-testid="ticket-entry-composer" className="flex flex-col gap-3 p-4">
      {/* Entry type selector — only shown to DEFT members */}
      {isDeftMember && (
        <div>
          <label className="text-xs font-medium text-foreground">Entry type</label>
          <div className="mt-1 flex flex-wrap gap-2">
            {DEFT_ENTRY_TYPES.map(({ value, label }) => (
              <button
                key={value}
                data-testid={`entry-type-btn-${value}`}
                onClick={() => setEntryType(value)}
                className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                  entryType === value
                    ? "bg-primary text-primary-foreground"
                    : "border border-border bg-background text-foreground hover:bg-accent"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
          {/* Description of selected type */}
          <p className="mt-1 text-xs text-muted-foreground">
            {DEFT_ENTRY_TYPES.find((t) => t.value === entryType)?.description ?? ""}
          </p>
        </div>
      )}

      {/* DEFT-only toggle (available for any type when DEFT member) */}
      {isDeftMember && entryType !== "context" && (
        <label className="flex items-center gap-2 text-xs text-muted-foreground">
          <input
            data-testid="deft-only-checkbox"
            type="checkbox"
            checked={isDeftOnly}
            onChange={(e) => setIsDeftOnly(e.target.checked)}
            className="h-3 w-3"
          />
          Hide from customer (DEFT only)
        </label>
      )}

      {/* Rich text editor */}
      <MessageEditor
        initialContent={body}
        onSubmit={() => void handleSubmit()}
        onChange={setBody}
        placeholder="Write a message…"
        disabled={submitting}
        showSubmit={false}
      />

      {/* Error banner */}
      {error && (
        <div
          data-testid="composer-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Submit */}
      <div className="flex justify-end">
        <button
          data-testid="composer-submit-btn"
          onClick={() => void handleSubmit()}
          disabled={submitting || !body.trim()}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          <Send className="h-4 w-4" />
          {submitting ? "Sending…" : "Send"}
        </button>
      </div>
    </div>
  );
}
