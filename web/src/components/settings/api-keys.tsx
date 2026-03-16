"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Copy, Key, Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { copyToClipboard } from "@/lib/clipboard";
import type { ApiKey, ApiKeyCreateResponse } from "@/lib/api-types";
import { fetchApiKeys, createApiKey, revokeApiKey } from "@/lib/settings-api";

interface ApiKeysProps {
  /** Clerk auth token for API calls. */
  token: string;
}

type ModalState =
  | { type: "closed" }
  | { type: "create" }
  | { type: "created"; response: ApiKeyCreateResponse }
  | { type: "revoke"; keyId: string; keyName: string };

/** Personal API key management component — list, create, and revoke keys. */
export function ApiKeys({ token }: ApiKeysProps): React.ReactNode {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [modal, setModal] = useState<ModalState>({ type: "closed" });
  const [newKeyName, setNewKeyName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);
  const mountedRef = useRef(true);

  const loadKeys = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await fetchApiKeys(token);
      if (mountedRef.current) setKeys(data);
    } catch {
      // Silently handle — keys will be empty.
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [token]);

  useEffect(() => {
    mountedRef.current = true;
    void loadKeys();
    return () => {
      mountedRef.current = false;
    };
  }, [loadKeys]);

  const handleCreate = async (): Promise<void> => {
    if (!newKeyName.trim()) return;
    setIsSubmitting(true);
    try {
      const response = await createApiKey(token, newKeyName.trim());
      if (mountedRef.current) {
        setModal({ type: "created", response });
        setNewKeyName("");
      }
    } catch {
      // Error handled via ApiError.
    } finally {
      if (mountedRef.current) setIsSubmitting(false);
    }
  };

  const handleCloseCreated = (): void => {
    setModal({ type: "closed" });
    void loadKeys();
  };

  const handleCopy = async (key: string): Promise<void> => {
    await copyToClipboard(key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleRevoke = async (keyId: string): Promise<void> => {
    setIsSubmitting(true);
    try {
      await revokeApiKey(token, keyId);
      if (mountedRef.current) {
        setModal({ type: "closed" });
        void loadKeys();
      }
    } catch {
      // Error handled via ApiError.
    } finally {
      if (mountedRef.current) setIsSubmitting(false);
    }
  };

  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  return (
    <div data-testid="api-keys-section" className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">API Keys</h2>
        <button
          data-testid="create-api-key-btn"
          onClick={() => {
            setNewKeyName("");
            setModal({ type: "create" });
          }}
          className="inline-flex items-center gap-1.5 rounded-md bg-foreground px-3 py-1.5 text-sm font-medium text-background hover:bg-foreground/90"
        >
          <Plus className="h-4 w-4" />
          Create Key
        </button>
      </div>

      <p className="text-sm text-muted-foreground">
        Manage API keys for programmatic access to the DEFT API.
      </p>

      {/* Key list */}
      {isLoading ? (
        <div className="py-8 text-center text-sm text-muted-foreground">Loading...</div>
      ) : keys.length === 0 ? (
        <div className="rounded-lg border border-dashed border-foreground/20 py-8 text-center">
          <Key className="mx-auto mb-2 h-8 w-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No API keys yet</p>
        </div>
      ) : (
        <div className="divide-y divide-foreground/10 rounded-lg border border-foreground/10">
          {keys.map((apiKey) => (
            <div
              key={apiKey.id}
              data-testid={`api-key-row-${apiKey.id}`}
              className="flex items-center justify-between px-4 py-3"
            >
              <div className="space-y-1">
                <p className="text-sm font-medium text-foreground">{apiKey.name}</p>
                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  <span className="font-mono">{apiKey.prefix}...</span>
                  <span>Created {formatDate(apiKey.created_at)}</span>
                  <span>
                    Last used: {apiKey.last_used_at ? formatDate(apiKey.last_used_at) : "Never"}
                  </span>
                </div>
              </div>
              <button
                data-testid="revoke-key-btn"
                onClick={() => setModal({ type: "revoke", keyId: apiKey.id, keyName: apiKey.name })}
                className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm text-red-500 hover:bg-red-500/10"
              >
                <Trash2 className="h-3.5 w-3.5" />
                Revoke
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Create key modal */}
      {modal.type === "create" && (
        <div
          data-testid="create-key-modal"
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        >
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
            <h3 className="mb-4 text-lg font-semibold text-foreground">Create API Key</h3>
            <input
              type="text"
              placeholder="Key name"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              className={cn(
                "mb-4 w-full rounded-md border border-foreground/20 bg-foreground/5 px-3 py-2 text-sm",
                "focus:border-foreground/40 focus:outline-none focus:ring-1 focus:ring-foreground/20",
              )}
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setModal({ type: "closed" })}
                className="rounded-md px-3 py-1.5 text-sm text-muted-foreground hover:bg-foreground/10"
              >
                Cancel
              </button>
              <button
                data-testid="confirm-create-key-btn"
                onClick={handleCreate}
                disabled={!newKeyName.trim() || isSubmitting}
                className="rounded-md bg-foreground px-3 py-1.5 text-sm font-medium text-background hover:bg-foreground/90 disabled:opacity-50"
              >
                {isSubmitting ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Created key display modal */}
      {modal.type === "created" && (
        <div
          data-testid="created-key-modal"
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        >
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
            <h3 className="mb-2 text-lg font-semibold text-foreground">API Key Created</h3>
            <p className="mb-4 text-sm text-muted-foreground">
              Copy your API key now. You won&apos;t be able to see it again.
            </p>
            <div className="mb-4 flex items-center gap-2 rounded-md bg-foreground/5 p-3">
              <code
                data-testid="created-key-value"
                className="flex-1 break-all font-mono text-sm text-foreground"
              >
                {modal.response.key}
              </code>
              <button
                data-testid="copy-key-btn"
                onClick={() => handleCopy(modal.response.key)}
                className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm text-foreground hover:bg-foreground/10"
              >
                <Copy className="h-4 w-4" />
                {copied ? "Copied!" : "Copy"}
              </button>
            </div>
            <div className="flex justify-end">
              <button
                data-testid="close-created-key-btn"
                onClick={handleCloseCreated}
                className="rounded-md bg-foreground px-3 py-1.5 text-sm font-medium text-background hover:bg-foreground/90"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke confirmation dialog */}
      {modal.type === "revoke" && (
        <div
          data-testid="revoke-confirm-dialog"
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        >
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
            <h3 className="mb-2 text-lg font-semibold text-foreground">Revoke API Key</h3>
            <p className="mb-4 text-sm text-muted-foreground">
              Are you sure you want to revoke this API key{" "}
              <strong>&quot;{modal.keyName}&quot;</strong>? This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button
                data-testid="cancel-revoke-btn"
                onClick={() => setModal({ type: "closed" })}
                className="rounded-md px-3 py-1.5 text-sm text-muted-foreground hover:bg-foreground/10"
              >
                Cancel
              </button>
              <button
                data-testid="confirm-revoke-btn"
                onClick={() => handleRevoke(modal.keyId)}
                disabled={isSubmitting}
                className="rounded-md bg-red-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-600 disabled:opacity-50"
              >
                {isSubmitting ? "Revoking..." : "Revoke"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
