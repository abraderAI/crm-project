"use client";

import { useCallback, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { Eye } from "lucide-react";

import type { RBACPreviewResponse } from "@/lib/api-types";
import { clientMutate } from "@/lib/api-client";

const ENTITY_TYPES = ["org", "space", "board"] as const;

/** Admin dry-run RBAC role preview — resolve a user's effective role for any entity. */
export function RBACPolicyPreview(): React.ReactNode {
  const { getToken } = useAuth();
  const [userId, setUserId] = useState("");
  const [entityType, setEntityType] = useState<string>("org");
  const [entityId, setEntityId] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<RBACPreviewResponse | null>(null);
  const [error, setError] = useState("");
  const [validationError, setValidationError] = useState("");

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setValidationError("");
      setError("");

      if (!userId.trim() || !entityId.trim()) {
        setValidationError("User ID and Entity ID are required.");
        return;
      }

      setLoading(true);
      setResult(null);
      try {
        const token = await getToken();
        if (!token) return;
        const res = await clientMutate<RBACPreviewResponse>("POST", "/admin/rbac-policy/preview", {
          token,
          body: {
            user_id: userId.trim(),
            entity_type: entityType,
            entity_id: entityId.trim(),
          },
        });
        setResult(res);
      } catch {
        setError("Failed to preview role resolution.");
      } finally {
        setLoading(false);
      }
    },
    [getToken, userId, entityType, entityId],
  );

  return (
    <div data-testid="rbac-policy-preview" className="flex flex-col gap-6">
      <div className="flex items-center gap-2">
        <Eye className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold text-foreground">Dry-Run Role Preview</h2>
      </div>

      <form onSubmit={handleSubmit} className="rounded-lg border border-border p-4">
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <label htmlFor="preview-user-id" className="text-xs text-muted-foreground">
              User ID
            </label>
            <input
              id="preview-user-id"
              type="text"
              data-testid="preview-user-id"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="Enter user ID…"
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            />
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor="preview-entity-type" className="text-xs text-muted-foreground">
              Entity Type
            </label>
            <select
              id="preview-entity-type"
              data-testid="preview-entity-type"
              value={entityType}
              onChange={(e) => setEntityType(e.target.value)}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            >
              {ENTITY_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor="preview-entity-id" className="text-xs text-muted-foreground">
              Entity ID
            </label>
            <input
              id="preview-entity-id"
              type="text"
              data-testid="preview-entity-id"
              value={entityId}
              onChange={(e) => setEntityId(e.target.value)}
              placeholder="Enter entity ID…"
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            />
          </div>

          {validationError && (
            <p className="text-xs text-destructive" data-testid="preview-validation-error">
              {validationError}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            data-testid="preview-submit-btn"
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Eye className="h-4 w-4" />
            {loading ? "Resolving…" : "Preview Role"}
          </button>
        </div>
      </form>

      {error && (
        <div
          data-testid="preview-error"
          className="rounded-lg border border-destructive bg-destructive/10 p-4 text-sm text-destructive"
        >
          {error}
        </div>
      )}

      {result && (
        <div data-testid="preview-result" className="rounded-lg border border-border p-4">
          <h3 className="mb-3 text-sm font-medium text-foreground">Resolution Result</h3>
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">User:</span>
              <span className="text-sm text-foreground">{result.user_id}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">Entity:</span>
              <span className="text-sm text-foreground">
                {result.entity_type} / {result.entity_id}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">Resolved Role:</span>
              <span
                data-testid="preview-resolved-role"
                className="rounded-full bg-primary/10 px-2 py-0.5 text-sm font-medium text-primary"
              >
                {result.role}
              </span>
            </div>
            <div className="flex flex-col gap-1" data-testid="preview-permissions">
              <span className="text-xs text-muted-foreground">Permissions:</span>
              <div className="flex flex-wrap gap-1">
                {(result.permissions ?? []).map((perm) => (
                  <span
                    key={perm}
                    className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                  >
                    {perm}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
