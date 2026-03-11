"use client";

import { useState } from "react";
import { Webhook, Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { WebhookSubscription } from "@/lib/api-types";

export interface WebhookManagerProps {
  /** Existing webhook subscriptions. */
  subscriptions: WebhookSubscription[];
  /** Whether data is loading. */
  loading?: boolean;
  /** Called when a new webhook is created. */
  onCreate: (url: string, eventFilter: string) => void;
  /** Called when a webhook is deleted. */
  onDelete: (subscriptionId: string) => void;
  /** Called when a webhook is toggled active/inactive. */
  onToggle: (subscriptionId: string) => void;
}

/** Webhook subscription management — list, create, delete, toggle active state. */
export function WebhookManager({
  subscriptions,
  loading = false,
  onCreate,
  onDelete,
  onToggle,
}: WebhookManagerProps): React.ReactNode {
  const [showCreate, setShowCreate] = useState(false);
  const [newUrl, setNewUrl] = useState("");
  const [newFilter, setNewFilter] = useState("");
  const [error, setError] = useState("");

  const handleCreate = (e: React.FormEvent): void => {
    e.preventDefault();
    if (!newUrl.trim()) {
      setError("URL is required.");
      return;
    }
    try {
      new URL(newUrl);
    } catch {
      setError("Please enter a valid URL.");
      return;
    }
    setError("");
    onCreate(newUrl.trim(), newFilter.trim());
    setNewUrl("");
    setNewFilter("");
    setShowCreate(false);
  };

  return (
    <div data-testid="webhook-manager" className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Webhook className="h-5 w-5 text-muted-foreground" data-testid="webhook-icon" />
          <h2 className="text-lg font-semibold text-foreground">Webhooks</h2>
          <span
            className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
            data-testid="webhook-count"
          >
            {subscriptions.length}
          </span>
        </div>
        <button
          onClick={() => setShowCreate(!showCreate)}
          data-testid="webhook-create-toggle"
          className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Add Webhook
        </button>
      </div>

      {/* Create form */}
      {showCreate && (
        <form
          onSubmit={handleCreate}
          data-testid="webhook-create-form"
          className="rounded-lg border border-border p-4"
        >
          <div className="flex flex-col gap-3">
            <div className="flex flex-col gap-1">
              <label htmlFor="webhook-url" className="text-sm font-medium text-foreground">
                Endpoint URL
              </label>
              <input
                id="webhook-url"
                type="text"
                value={newUrl}
                onChange={(e) => setNewUrl(e.target.value)}
                placeholder="https://example.com/webhook"
                data-testid="webhook-url-input"
                className={cn(
                  "rounded-md border bg-background px-3 py-2 text-sm text-foreground",
                  error ? "border-destructive" : "border-border",
                )}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label htmlFor="webhook-filter" className="text-sm font-medium text-foreground">
                Event Filter (optional)
              </label>
              <input
                id="webhook-filter"
                type="text"
                value={newFilter}
                onChange={(e) => setNewFilter(e.target.value)}
                placeholder="message.created, thread.updated"
                data-testid="webhook-filter-input"
                className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            {error && (
              <p className="text-xs text-destructive" data-testid="webhook-error">
                {error}
              </p>
            )}
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={loading}
                data-testid="webhook-save-btn"
                className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                Save
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowCreate(false);
                  setError("");
                }}
                data-testid="webhook-cancel-btn"
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
              >
                Cancel
              </button>
            </div>
          </div>
        </form>
      )}

      {/* Loading */}
      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="webhook-loading"
        >
          Loading webhooks...
        </div>
      )}

      {/* Empty state */}
      {!loading && subscriptions.length === 0 && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="webhook-empty">
          No webhook subscriptions.
        </div>
      )}

      {/* Subscription list */}
      {!loading && subscriptions.length > 0 && (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="webhook-list"
        >
          {subscriptions.map((sub) => (
            <div
              key={sub.id}
              className="flex items-center gap-3 px-4 py-3"
              data-testid={`webhook-item-${sub.id}`}
            >
              <div className="min-w-0 flex-1">
                <p
                  className="truncate text-sm font-medium text-foreground"
                  data-testid={`webhook-url-${sub.id}`}
                >
                  {sub.url}
                </p>
                <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                  <span data-testid={`webhook-scope-${sub.id}`}>
                    {sub.scope_type}:{sub.scope_id}
                  </span>
                  {sub.event_filter && (
                    <span data-testid={`webhook-filter-${sub.id}`}>{sub.event_filter}</span>
                  )}
                </div>
              </div>
              <button
                onClick={() => onToggle(sub.id)}
                data-testid={`webhook-toggle-${sub.id}`}
                className={cn(
                  "rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors",
                  sub.is_active
                    ? "bg-green-100 text-green-800 hover:bg-green-200"
                    : "bg-muted text-muted-foreground hover:bg-muted/80",
                )}
              >
                {sub.is_active ? "Active" : "Inactive"}
              </button>
              <button
                onClick={() => onDelete(sub.id)}
                data-testid={`webhook-delete-${sub.id}`}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
