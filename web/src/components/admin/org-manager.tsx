"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { Building2, Plus, X, AlertTriangle, CheckCircle } from "lucide-react";
import type { AdminOrgDetail } from "@/lib/api-types";
import { clientMutate } from "@/lib/api-client";

/** Props for OrgManager. */
export interface OrgManagerProps {
  /** Initial org list from the server. */
  initialOrgs: AdminOrgDetail[];
}

/** Admin org list with inline create, suspend/unsuspend actions. */
export function OrgManager({ initialOrgs }: OrgManagerProps): React.ReactNode {
  const { getToken } = useAuth();
  const router = useRouter();

  const [orgs, setOrgs] = useState<AdminOrgDetail[]>(initialOrgs);
  const [error, setError] = useState("");
  const [successMsg, setSuccessMsg] = useState("");

  // --- Create org state ---
  const [showCreate, setShowCreate] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createDesc, setCreateDesc] = useState("");
  const [creating, setCreating] = useState(false);

  // --- Suspend state ---
  const [suspendTarget, setSuspendTarget] = useState<AdminOrgDetail | null>(null);
  const [suspendReason, setSuspendReason] = useState("");
  const [suspending, setSuspending] = useState(false);

  // --- Generic action loading ---
  const [loadingId, setLoadingId] = useState<string | null>(null);

  const showSuccess = (msg: string): void => {
    setSuccessMsg(msg);
    setTimeout(() => setSuccessMsg(""), 3000);
  };

  // --- Create org ---
  const handleCreate = useCallback(async () => {
    if (!createName.trim()) return;
    setError("");
    setCreating(true);
    try {
      const token = await getToken();
      if (!token) return;
      const created = await clientMutate<AdminOrgDetail>("POST", "/orgs", {
        token,
        body: { name: createName.trim(), description: createDesc.trim() || undefined },
      });
      setOrgs((prev) => [created, ...prev]);
      setCreateName("");
      setCreateDesc("");
      setShowCreate(false);
      showSuccess(`Org "${created.name}" created.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create org");
    } finally {
      setCreating(false);
    }
  }, [getToken, createName, createDesc]);

  // --- Suspend org ---
  const handleSuspend = useCallback(async () => {
    if (!suspendTarget) return;
    setError("");
    setSuspending(true);
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("POST", `/admin/orgs/${suspendTarget.id}/suspend`, {
        token,
        body: { reason: suspendReason },
      });
      setOrgs((prev) =>
        prev.map((o) =>
          o.id === suspendTarget.id
            ? { ...o, suspended_at: new Date().toISOString(), suspend_reason: suspendReason }
            : o,
        ),
      );
      showSuccess(`Org "${suspendTarget.name}" suspended.`);
      setSuspendTarget(null);
      setSuspendReason("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to suspend org");
    } finally {
      setSuspending(false);
    }
  }, [getToken, suspendTarget, suspendReason]);

  // --- Unsuspend org ---
  const handleUnsuspend = useCallback(
    async (org: AdminOrgDetail) => {
      setError("");
      setLoadingId(org.id);
      try {
        const token = await getToken();
        if (!token) return;
        await clientMutate<void>("POST", `/admin/orgs/${org.id}/unsuspend`, { token });
        setOrgs((prev) =>
          prev.map((o) =>
            o.id === org.id ? { ...o, suspended_at: null, suspend_reason: "" } : o,
          ),
        );
        showSuccess(`Org "${org.name}" unsuspended.`);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to unsuspend org");
      } finally {
        setLoadingId(null);
      }
    },
    [getToken],
  );

  const formatDate = (dateStr: string): string => {
    try {
      return new Date(dateStr).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div data-testid="org-manager" className="flex flex-col gap-4">
      {/* Feedback */}
      {error && (
        <div
          data-testid="org-manager-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}
      {successMsg && (
        <div
          data-testid="org-manager-success"
          className="flex items-center gap-2 rounded-md bg-green-50 px-4 py-3 text-sm text-green-700"
        >
          <CheckCircle className="h-4 w-4 shrink-0" />
          {successMsg}
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">
          Organizations ({orgs.length})
        </h2>
        <button
          data-testid="create-org-btn"
          onClick={() => setShowCreate((v) => !v)}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Create Org
        </button>
      </div>

      {/* Inline create form */}
      {showCreate && (
        <div
          data-testid="create-org-form"
          className="rounded-lg border border-border bg-muted/30 p-4"
        >
          <h3 className="mb-3 text-sm font-semibold text-foreground">New Organization</h3>
          <div className="flex flex-col gap-3">
            <div>
              <label htmlFor="create-org-name" className="text-xs font-medium text-foreground">
                Name <span className="text-red-500">*</span>
              </label>
              <input
                id="create-org-name"
                data-testid="create-org-name-input"
                type="text"
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
                placeholder="Organization name"
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label htmlFor="create-org-desc" className="text-xs font-medium text-foreground">
                Description
              </label>
              <input
                id="create-org-desc"
                data-testid="create-org-desc-input"
                type="text"
                value={createDesc}
                onChange={(e) => setCreateDesc(e.target.value)}
                placeholder="Optional description"
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div className="flex gap-2">
              <button
                data-testid="create-org-submit"
                onClick={handleCreate}
                disabled={creating || !createName.trim()}
                className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create"}
              </button>
              <button
                data-testid="create-org-cancel"
                onClick={() => {
                  setShowCreate(false);
                  setCreateName("");
                  setCreateDesc("");
                }}
                className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Org table */}
      {orgs.length === 0 ? (
        <div
          data-testid="org-list-empty"
          className="flex flex-col items-center gap-2 py-12 text-sm text-muted-foreground"
        >
          <Building2 className="h-8 w-8 opacity-30" />
          No organizations found.
        </div>
      ) : (
        <div
          data-testid="org-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {/* Table header */}
          <div className="hidden grid-cols-[2fr_1fr_1fr_1fr_auto] gap-4 bg-muted/50 px-4 py-2 text-xs font-medium text-muted-foreground sm:grid">
            <span>Name / Slug</span>
            <span>Tier</span>
            <span>Members</span>
            <span>Created</span>
            <span>Actions</span>
          </div>

          {orgs.map((org) => {
            const isSuspended = !!org.suspended_at;
            return (
              <div
                key={org.id}
                data-testid={`org-row-${org.id}`}
                className="grid grid-cols-1 gap-2 px-4 py-3 sm:grid-cols-[2fr_1fr_1fr_1fr_auto] sm:items-center sm:gap-4"
              >
                {/* Name / Slug */}
                <div className="flex flex-col gap-0.5">
                  <Link
                    href={`/admin/orgs/${org.id}`}
                    data-testid={`org-name-link-${org.id}`}
                    className="text-sm font-medium text-foreground hover:underline"
                  >
                    {org.name}
                  </Link>
                  <span className="text-xs text-muted-foreground">{org.slug}</span>
                  {isSuspended && (
                    <span
                      data-testid={`org-suspended-badge-${org.id}`}
                      className="mt-0.5 w-fit rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700"
                    >
                      Suspended
                    </span>
                  )}
                </div>

                {/* Tier */}
                <span className="text-sm capitalize text-foreground">
                  {org.billing_tier ?? "—"}
                </span>

                {/* Members */}
                <span data-testid={`org-member-count-${org.id}`} className="text-sm text-foreground">
                  {org.member_count}
                </span>

                {/* Created */}
                <span className="text-xs text-muted-foreground">{formatDate(org.created_at)}</span>

                {/* Actions */}
                <div className="flex items-center gap-2">
                  {isSuspended ? (
                    <button
                      data-testid={`unsuspend-btn-${org.id}`}
                      onClick={() => void handleUnsuspend(org)}
                      disabled={loadingId === org.id}
                      className="rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 hover:bg-green-100 disabled:opacity-50"
                    >
                      {loadingId === org.id ? "..." : "Unsuspend"}
                    </button>
                  ) : (
                    <button
                      data-testid={`suspend-btn-${org.id}`}
                      onClick={() => setSuspendTarget(org)}
                      className="rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 hover:bg-amber-100"
                    >
                      Suspend
                    </button>
                  )}
                  <Link
                    href={`/admin/orgs/${org.id}`}
                    data-testid={`org-detail-link-${org.id}`}
                    className="rounded-md bg-muted px-2 py-1 text-xs font-medium text-foreground hover:bg-muted/80"
                    onClick={() => router.push(`/admin/orgs/${org.id}`)}
                  >
                    Details
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Suspend confirmation dialog */}
      {suspendTarget && (
        <div
          data-testid="suspend-confirm-dialog"
          className="rounded-lg border border-amber-200 bg-amber-50 p-6 shadow-lg"
        >
          <div className="flex items-start justify-between">
            <h3 className="text-base font-semibold text-amber-900">
              Suspend &quot;{suspendTarget.name}&quot;
            </h3>
            <button
              data-testid="suspend-dialog-close"
              onClick={() => {
                setSuspendTarget(null);
                setSuspendReason("");
              }}
              className="text-amber-600 hover:text-amber-800"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="mt-1 text-sm text-amber-700">
            Users in this org will lose access until it is unsuspended.
          </p>
          <div className="mt-3">
            <label htmlFor="suspend-reason" className="text-sm font-medium text-amber-900">
              Reason (optional)
            </label>
            <textarea
              id="suspend-reason"
              data-testid="suspend-reason-input"
              value={suspendReason}
              onChange={(e) => setSuspendReason(e.target.value)}
              placeholder="Enter reason for suspension..."
              rows={2}
              className="mt-1 w-full rounded-md border border-amber-300 bg-white px-3 py-2 text-sm text-foreground"
            />
          </div>
          <div className="mt-4 flex gap-2">
            <button
              data-testid="suspend-confirm-btn"
              onClick={handleSuspend}
              disabled={suspending}
              className="rounded-md bg-amber-600 px-3 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50"
            >
              {suspending ? "Suspending..." : "Confirm Suspend"}
            </button>
            <button
              data-testid="suspend-cancel-btn"
              onClick={() => {
                setSuspendTarget(null);
                setSuspendReason("");
              }}
              className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
