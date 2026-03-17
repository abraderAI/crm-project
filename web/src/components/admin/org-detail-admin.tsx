"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { ChevronLeft, AlertTriangle, CheckCircle, Pencil, X } from "lucide-react";
import type { AdminOrgDetail } from "@/lib/api-types";
import { clientMutate } from "@/lib/api-client";

/** Props for OrgDetailAdmin. */
export interface OrgDetailAdminProps {
  /** Initial org detail from the server. */
  org: AdminOrgDetail;
}

/** Admin org detail with edit, suspend/unsuspend, transfer ownership, and purge. */
export function OrgDetailAdmin({ org: initialOrg }: OrgDetailAdminProps): React.ReactNode {
  const { getToken } = useAuth();
  const router = useRouter();

  const [org, setOrg] = useState<AdminOrgDetail>(initialOrg);
  const [error, setError] = useState("");
  const [successMsg, setSuccessMsg] = useState("");

  // --- Edit state ---
  const [showEdit, setShowEdit] = useState(false);
  const [editName, setEditName] = useState(org.name);
  const [editDesc, setEditDesc] = useState(org.description ?? "");
  const [saving, setSaving] = useState(false);

  // --- Suspend state ---
  const [showSuspendDialog, setShowSuspendDialog] = useState(false);
  const [suspendReason, setSuspendReason] = useState("");
  const [suspending, setSuspending] = useState(false);

  // --- Transfer state ---
  const [showTransferDialog, setShowTransferDialog] = useState(false);
  const [newOwnerUserId, setNewOwnerUserId] = useState("");
  const [transferring, setTransferring] = useState(false);

  // --- Purge state ---
  const [showPurgeDialog, setShowPurgeDialog] = useState(false);
  const [purgeConfirm, setPurgeConfirm] = useState("");
  const [purging, setPurging] = useState(false);

  const showSuccess = (msg: string): void => {
    setSuccessMsg(msg);
    setTimeout(() => setSuccessMsg(""), 3000);
  };

  const isSuspended = !!org.suspended_at;
  const purgeToken = `purge ${org.slug}`;

  // --- Save edit ---
  const handleSaveEdit = useCallback(async () => {
    if (!editName.trim()) return;
    setError("");
    setSaving(true);
    try {
      const token = await getToken();
      if (!token) return;
      const updated = await clientMutate<AdminOrgDetail>("PATCH", `/orgs/${org.slug}`, {
        token,
        body: { name: editName.trim(), description: editDesc.trim() || undefined },
      });
      setOrg((prev) => ({ ...prev, name: updated.name, description: updated.description }));
      setShowEdit(false);
      showSuccess("Organization updated.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update org");
    } finally {
      setSaving(false);
    }
  }, [getToken, org.slug, editName, editDesc]);

  // --- Suspend ---
  const handleSuspend = useCallback(async () => {
    setError("");
    setSuspending(true);
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("POST", `/admin/orgs/${org.id}/suspend`, {
        token,
        body: { reason: suspendReason },
      });
      setOrg((prev) => ({
        ...prev,
        suspended_at: new Date().toISOString(),
        suspend_reason: suspendReason,
      }));
      setShowSuspendDialog(false);
      setSuspendReason("");
      showSuccess("Organization suspended.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to suspend org");
    } finally {
      setSuspending(false);
    }
  }, [getToken, org.id, suspendReason]);

  // --- Unsuspend ---
  const handleUnsuspend = useCallback(async () => {
    setError("");
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("POST", `/admin/orgs/${org.id}/unsuspend`, { token });
      setOrg((prev) => ({ ...prev, suspended_at: null, suspend_reason: "" }));
      showSuccess("Organization unsuspended.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to unsuspend org");
    }
  }, [getToken, org.id]);

  // --- Transfer ownership ---
  const handleTransfer = useCallback(async () => {
    if (!newOwnerUserId.trim()) return;
    setError("");
    setTransferring(true);
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("POST", `/admin/orgs/${org.id}/transfer-ownership`, {
        token,
        body: { new_owner_user_id: newOwnerUserId.trim() },
      });
      setShowTransferDialog(false);
      setNewOwnerUserId("");
      showSuccess("Ownership transferred.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to transfer ownership");
    } finally {
      setTransferring(false);
    }
  }, [getToken, org.id, newOwnerUserId]);

  // --- Purge ---
  const handlePurge = useCallback(async () => {
    if (purgeConfirm !== purgeToken) return;
    setError("");
    setPurging(true);
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("DELETE", `/admin/orgs/${org.id}/purge`, {
        token,
        body: { confirm: purgeToken },
      });
      router.push("/admin/orgs");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to purge org");
      setPurging(false);
    }
  }, [getToken, org.id, purgeConfirm, purgeToken, router]);

  const formatDate = (dateStr?: string | null): string => {
    if (!dateStr) return "—";
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
    <div data-testid="org-detail-admin" className="flex flex-col gap-6">
      {/* Back link */}
      <Link
        href="/admin/orgs"
        data-testid="back-to-orgs"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="h-4 w-4" />
        All Organizations
      </Link>

      {/* Feedback */}
      {error && (
        <div
          data-testid="org-detail-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}
      {successMsg && (
        <div
          data-testid="org-detail-success"
          className="flex items-center gap-2 rounded-md bg-green-50 px-4 py-3 text-sm text-green-700"
        >
          <CheckCircle className="h-4 w-4 shrink-0" />
          {successMsg}
        </div>
      )}

      {/* Header card */}
      <div className="rounded-lg border border-border p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 data-testid="org-detail-name" className="text-xl font-bold text-foreground">
                {org.name}
              </h2>
              {isSuspended && (
                <span
                  data-testid="org-detail-suspended-badge"
                  className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700"
                >
                  Suspended
                </span>
              )}
            </div>
            <p data-testid="org-detail-slug" className="mt-0.5 text-sm text-muted-foreground">
              /{org.slug}
            </p>
            {org.description && (
              <p data-testid="org-detail-description" className="mt-2 text-sm text-foreground">
                {org.description}
              </p>
            )}
          </div>
          <button
            data-testid="edit-org-btn"
            onClick={() => {
              setEditName(org.name);
              setEditDesc(org.description ?? "");
              setShowEdit((v) => !v);
            }}
            className="inline-flex items-center gap-1.5 rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
          >
            <Pencil className="h-3.5 w-3.5" />
            Edit
          </button>
        </div>

        {/* Inline edit form */}
        {showEdit && (
          <div data-testid="edit-org-form" className="mt-4 flex flex-col gap-3 border-t border-border pt-4">
            <div>
              <label htmlFor="edit-org-name" className="text-xs font-medium text-foreground">
                Name <span className="text-red-500">*</span>
              </label>
              <input
                id="edit-org-name"
                data-testid="edit-org-name-input"
                type="text"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div>
              <label htmlFor="edit-org-desc" className="text-xs font-medium text-foreground">
                Description
              </label>
              <input
                id="edit-org-desc"
                data-testid="edit-org-desc-input"
                type="text"
                value={editDesc}
                onChange={(e) => setEditDesc(e.target.value)}
                placeholder="Optional description"
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              />
            </div>
            <div className="flex gap-2">
              <button
                data-testid="edit-org-save"
                onClick={handleSaveEdit}
                disabled={saving || !editName.trim()}
                className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {saving ? "Saving..." : "Save"}
              </button>
              <button
                data-testid="edit-org-cancel"
                onClick={() => setShowEdit(false)}
                className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {[
          { label: "Members", value: org.member_count, testId: "stat-members" },
          { label: "Spaces", value: org.space_count, testId: "stat-spaces" },
          { label: "Boards", value: org.board_count, testId: "stat-boards" },
          { label: "Threads", value: org.thread_count, testId: "stat-threads" },
        ].map(({ label, value, testId }) => (
          <div key={label} className="rounded-lg border border-border p-4 text-center">
            <p className="text-2xl font-bold text-foreground" data-testid={testId}>
              {value}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground">{label}</p>
          </div>
        ))}
      </div>

      {/* Metadata */}
      <div className="rounded-lg border border-border p-4 text-sm">
        <h3 className="mb-3 font-semibold text-foreground">Details</h3>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 sm:grid-cols-3">
          <div>
            <dt className="text-xs text-muted-foreground">Billing Tier</dt>
            <dd className="capitalize text-foreground">{org.billing_tier ?? "—"}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">Payment Status</dt>
            <dd className="capitalize text-foreground">{org.payment_status ?? "—"}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">Created</dt>
            <dd className="text-foreground">{formatDate(org.created_at)}</dd>
          </div>
          {isSuspended && (
            <div className="col-span-2 sm:col-span-3">
              <dt className="text-xs text-muted-foreground">Suspended reason</dt>
              <dd className="text-red-700">{org.suspend_reason || "No reason provided"}</dd>
            </div>
          )}
        </dl>
      </div>

      {/* Admin actions */}
      <div className="flex flex-wrap gap-3">
        {isSuspended ? (
          <button
            data-testid="unsuspend-org-btn"
            onClick={() => void handleUnsuspend()}
            className="inline-flex items-center gap-1.5 rounded-md bg-green-50 px-3 py-2 text-sm font-medium text-green-700 hover:bg-green-100"
          >
            Unsuspend Org
          </button>
        ) : (
          <button
            data-testid="suspend-org-btn"
            onClick={() => setShowSuspendDialog(true)}
            className="inline-flex items-center gap-1.5 rounded-md bg-amber-50 px-3 py-2 text-sm font-medium text-amber-700 hover:bg-amber-100"
          >
            Suspend Org
          </button>
        )}

        <button
          data-testid="transfer-ownership-btn"
          onClick={() => setShowTransferDialog(true)}
          className="inline-flex items-center gap-1.5 rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
        >
          Transfer Ownership
        </button>

        <button
          data-testid="purge-org-btn"
          onClick={() => setShowPurgeDialog(true)}
          className="inline-flex items-center gap-1.5 rounded-md bg-red-50 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-100"
        >
          GDPR Purge
        </button>
      </div>

      {/* Suspend dialog */}
      {showSuspendDialog && (
        <div
          data-testid="suspend-dialog"
          className="rounded-lg border border-amber-200 bg-amber-50 p-6 shadow-lg"
        >
          <div className="flex items-start justify-between">
            <h3 className="text-base font-semibold text-amber-900">Suspend Organization</h3>
            <button
              data-testid="suspend-dialog-close"
              onClick={() => { setShowSuspendDialog(false); setSuspendReason(""); }}
              className="text-amber-600 hover:text-amber-800"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="mt-1 text-sm text-amber-700">
            Members of &quot;{org.name}&quot; will lose access until unsuspended.
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
              placeholder="Enter reason..."
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
              onClick={() => { setShowSuspendDialog(false); setSuspendReason(""); }}
              className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Transfer ownership dialog */}
      {showTransferDialog && (
        <div
          data-testid="transfer-dialog"
          className="rounded-lg border border-border bg-background p-6 shadow-lg"
        >
          <div className="flex items-start justify-between">
            <h3 className="text-base font-semibold text-foreground">Transfer Ownership</h3>
            <button
              data-testid="transfer-dialog-close"
              onClick={() => { setShowTransferDialog(false); setNewOwnerUserId(""); }}
              className="text-muted-foreground hover:text-foreground"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            The current owners will be demoted to admin. The new owner will be added as owner.
          </p>
          <div className="mt-3">
            <label htmlFor="new-owner" className="text-sm font-medium text-foreground">
              New Owner User ID <span className="text-red-500">*</span>
            </label>
            <input
              id="new-owner"
              data-testid="new-owner-input"
              type="text"
              value={newOwnerUserId}
              onChange={(e) => setNewOwnerUserId(e.target.value)}
              placeholder="user_xxxxx (Clerk user ID)"
              className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            />
          </div>
          <div className="mt-4 flex gap-2">
            <button
              data-testid="transfer-confirm-btn"
              onClick={handleTransfer}
              disabled={transferring || !newOwnerUserId.trim()}
              className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {transferring ? "Transferring..." : "Transfer"}
            </button>
            <button
              data-testid="transfer-cancel-btn"
              onClick={() => { setShowTransferDialog(false); setNewOwnerUserId(""); }}
              className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Purge dialog */}
      {showPurgeDialog && (
        <div
          data-testid="purge-dialog"
          className="rounded-lg border border-red-200 bg-red-50 p-6 shadow-lg"
        >
          <div className="flex items-start justify-between">
            <h3 className="text-base font-semibold text-red-900">GDPR Purge — Irreversible</h3>
            <button
              data-testid="purge-dialog-close"
              onClick={() => { setShowPurgeDialog(false); setPurgeConfirm(""); }}
              className="text-red-600 hover:text-red-800"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="mt-1 text-sm text-red-700">
            This will permanently delete this org and all associated data. Type the confirmation
            phrase to proceed:
          </p>
          <p className="mt-1 font-mono text-sm font-medium text-red-800">{purgeToken}</p>
          <input
            data-testid="purge-confirm-input"
            type="text"
            value={purgeConfirm}
            onChange={(e) => setPurgeConfirm(e.target.value)}
            placeholder={`Type "${purgeToken}" to confirm`}
            className="mt-3 w-full rounded-md border border-red-300 bg-white px-3 py-2 text-sm text-foreground"
          />
          <div className="mt-4 flex gap-2">
            <button
              data-testid="purge-confirm-btn"
              onClick={handlePurge}
              disabled={purgeConfirm !== purgeToken || purging}
              className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
            >
              {purging ? "Purging..." : "Permanently Delete"}
            </button>
            <button
              data-testid="purge-cancel-btn"
              onClick={() => { setShowPurgeDialog(false); setPurgeConfirm(""); }}
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
