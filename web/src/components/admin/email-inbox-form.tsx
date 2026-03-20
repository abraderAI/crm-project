"use client";

import { useState } from "react";
import { Save, X } from "lucide-react";
import type { EmailInbox, EmailInboxInput, RoutingAction } from "@/lib/api-types";

const ROUTING_LABELS: Record<RoutingAction, string> = {
  support_ticket: "Support Ticket",
  sales_lead: "Sales Lead",
  general: "General Message",
};

const ROUTING_DESCRIPTIONS: Record<RoutingAction, string> = {
  support_ticket: "Creates a ticket in the Support space",
  sales_lead: "Creates a lead in the CRM space",
  general: "Creates a thread in the General space",
};

interface EmailInboxFormProps {
  /** Existing inbox to edit, or undefined for a new inbox. */
  inbox?: EmailInbox;
  /** Called when the form is submitted with valid data. */
  onSave: (input: EmailInboxInput) => Promise<void>;
  /** Called when the form is cancelled. */
  onCancel: () => void;
}

/** Form for creating or editing an email inbox configuration. */
export function EmailInboxForm({ inbox, onSave, onCancel }: EmailInboxFormProps): React.ReactNode {
  const [name, setName] = useState(inbox?.name ?? "");
  const [emailAddress, setEmailAddress] = useState(inbox?.email_address ?? "");
  const [imapHost, setImapHost] = useState(inbox?.imap_host ?? "");
  const [imapPort, setImapPort] = useState(String(inbox?.imap_port ?? 993));
  const [username, setUsername] = useState(inbox?.username ?? "");
  const [password, setPassword] = useState("");
  const [mailbox, setMailbox] = useState(inbox?.mailbox ?? "INBOX");
  const [routingAction, setRoutingAction] = useState<RoutingAction>(
    inbox?.routing_action ?? "support_ticket",
  );
  const [enabled, setEnabled] = useState(inbox?.enabled ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const isNew = !inbox;

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    setError("");
    setSaving(true);

    const input: EmailInboxInput = {
      name,
      email_address: emailAddress,
      imap_host: imapHost,
      imap_port: Number(imapPort) || 993,
      username,
      mailbox: mailbox || "INBOX",
      routing_action: routingAction,
      enabled,
    };
    // Only include password if the user typed one.
    if (password.trim()) {
      input.password = password.trim();
    }

    try {
      await onSave(input);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save inbox.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      data-testid="email-inbox-form"
      className="rounded-lg border border-border bg-background p-6"
    >
      <h3 className="text-base font-semibold text-foreground">
        {isNew ? "Add Email Inbox" : "Edit Email Inbox"}
      </h3>

      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* Name */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-name" className="text-sm font-medium text-foreground">
            Inbox Name <span className="text-destructive">*</span>
          </label>
          <input
            id="inbox-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Support"
            required
            data-testid="inbox-name"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* Display Email Address */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-email" className="text-sm font-medium text-foreground">
            Email Address
          </label>
          <input
            id="inbox-email"
            type="email"
            value={emailAddress}
            onChange={(e) => setEmailAddress(e.target.value)}
            placeholder="support@yourdomain.com"
            data-testid="inbox-email-address"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* IMAP Host */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-imap-host" className="text-sm font-medium text-foreground">
            IMAP Host <span className="text-destructive">*</span>
          </label>
          <input
            id="inbox-imap-host"
            type="text"
            value={imapHost}
            onChange={(e) => setImapHost(e.target.value)}
            placeholder="imap.gmail.com"
            required
            data-testid="inbox-imap-host"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* IMAP Port */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-imap-port" className="text-sm font-medium text-foreground">
            IMAP Port <span className="text-destructive">*</span>
          </label>
          <input
            id="inbox-imap-port"
            type="number"
            value={imapPort}
            onChange={(e) => setImapPort(e.target.value)}
            min={1}
            max={65535}
            required
            data-testid="inbox-imap-port"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* Username */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-username" className="text-sm font-medium text-foreground">
            Username <span className="text-destructive">*</span>
          </label>
          <input
            id="inbox-username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="support@yourdomain.com"
            required
            data-testid="inbox-username"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* Password */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-password" className="text-sm font-medium text-foreground">
            Password / App Password{" "}
            {!isNew && (
              <span className="text-xs font-normal text-muted-foreground">
                (leave blank to keep existing)
              </span>
            )}
            {isNew && <span className="text-destructive">*</span>}
          </label>
          <input
            id="inbox-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={isNew ? "App password" : "Enter new password to update"}
            required={isNew}
            data-testid="inbox-password"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* Mailbox */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-mailbox" className="text-sm font-medium text-foreground">
            Mailbox
          </label>
          <input
            id="inbox-mailbox"
            type="text"
            value={mailbox}
            onChange={(e) => setMailbox(e.target.value)}
            placeholder="INBOX"
            data-testid="inbox-mailbox"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* Routing Action */}
        <div className="flex flex-col gap-1">
          <label htmlFor="inbox-routing" className="text-sm font-medium text-foreground">
            When email arrives, create a…
          </label>
          <select
            id="inbox-routing"
            value={routingAction}
            onChange={(e) => setRoutingAction(e.target.value as RoutingAction)}
            data-testid="inbox-routing-action"
            className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          >
            {(Object.keys(ROUTING_LABELS) as RoutingAction[]).map((action) => (
              <option key={action} value={action}>
                {ROUTING_LABELS[action]}
              </option>
            ))}
          </select>
          <p className="text-xs text-muted-foreground">{ROUTING_DESCRIPTIONS[routingAction]}</p>
        </div>
      </div>

      {/* Enabled toggle */}
      <div className="mt-4 flex items-center gap-3">
        <label htmlFor="inbox-enabled" className="text-sm font-medium text-foreground">
          Active
        </label>
        <button
          id="inbox-enabled"
          type="button"
          role="switch"
          aria-checked={enabled}
          onClick={() => setEnabled(!enabled)}
          data-testid="inbox-enabled-toggle"
          className={`relative h-6 w-11 rounded-full transition-colors ${enabled ? "bg-primary" : "bg-muted"}`}
        >
          <span
            className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${enabled ? "translate-x-5" : "translate-x-0"}`}
          />
        </button>
      </div>

      {error && (
        <p className="mt-3 text-xs text-destructive" data-testid="inbox-form-error">
          {error}
        </p>
      )}

      <div className="mt-5 flex gap-2">
        <button
          type="submit"
          disabled={saving}
          data-testid="inbox-save-btn"
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving…" : "Save Inbox"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          disabled={saving}
          data-testid="inbox-cancel-btn"
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent disabled:opacity-50"
        >
          <X className="h-4 w-4" />
          Cancel
        </button>
      </div>
    </form>
  );
}
