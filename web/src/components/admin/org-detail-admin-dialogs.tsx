"use client";

import { X } from "lucide-react";

// --- Suspend Dialog ---

export interface OrgSuspendDialogProps {
  orgName: string;
  suspendReason: string;
  setSuspendReason: (v: string) => void;
  suspending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function OrgSuspendDialog({
  orgName,
  suspendReason,
  setSuspendReason,
  suspending,
  onConfirm,
  onCancel,
}: OrgSuspendDialogProps): React.ReactNode {
  return (
    <div
      data-testid="suspend-dialog"
      className="rounded-lg border border-amber-200 bg-amber-50 p-6 shadow-lg"
    >
      <div className="flex items-start justify-between">
        <h3 className="text-base font-semibold text-amber-900">Suspend Organization</h3>
        <button
          data-testid="suspend-dialog-close"
          onClick={onCancel}
          className="text-amber-600 hover:text-amber-800"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <p className="mt-1 text-sm text-amber-700">
        Members of &quot;{orgName}&quot; will lose access until unsuspended.
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
          onClick={onConfirm}
          disabled={suspending}
          className="rounded-md bg-amber-600 px-3 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50"
        >
          {suspending ? "Suspending..." : "Confirm Suspend"}
        </button>
        <button
          data-testid="suspend-cancel-btn"
          onClick={onCancel}
          className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

// --- Transfer Ownership Dialog ---

export interface OrgTransferDialogProps {
  newOwnerUserId: string;
  setNewOwnerUserId: (v: string) => void;
  transferring: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function OrgTransferDialog({
  newOwnerUserId,
  setNewOwnerUserId,
  transferring,
  onConfirm,
  onCancel,
}: OrgTransferDialogProps): React.ReactNode {
  return (
    <div
      data-testid="transfer-dialog"
      className="rounded-lg border border-border bg-background p-6 shadow-lg"
    >
      <div className="flex items-start justify-between">
        <h3 className="text-base font-semibold text-foreground">Transfer Ownership</h3>
        <button
          data-testid="transfer-dialog-close"
          onClick={onCancel}
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
          onClick={onConfirm}
          disabled={transferring || !newOwnerUserId.trim()}
          className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {transferring ? "Transferring..." : "Transfer"}
        </button>
        <button
          data-testid="transfer-cancel-btn"
          onClick={onCancel}
          className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

// --- GDPR Purge Dialog ---

export interface OrgPurgeDialogProps {
  purgeToken: string;
  purgeConfirm: string;
  setPurgeConfirm: (v: string) => void;
  purging: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function OrgPurgeDialog({
  purgeToken,
  purgeConfirm,
  setPurgeConfirm,
  purging,
  onConfirm,
  onCancel,
}: OrgPurgeDialogProps): React.ReactNode {
  return (
    <div
      data-testid="purge-dialog"
      className="rounded-lg border border-red-200 bg-red-50 p-6 shadow-lg"
    >
      <div className="flex items-start justify-between">
        <h3 className="text-base font-semibold text-red-900">GDPR Purge — Irreversible</h3>
        <button
          data-testid="purge-dialog-close"
          onClick={onCancel}
          className="text-red-600 hover:text-red-800"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <p className="mt-1 text-sm text-red-700">
        This will permanently delete this org and all associated data. Type the confirmation phrase
        to proceed:
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
          onClick={onConfirm}
          disabled={purgeConfirm !== purgeToken || purging}
          className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
        >
          {purging ? "Purging..." : "Permanently Delete"}
        </button>
        <button
          data-testid="purge-cancel-btn"
          onClick={onCancel}
          className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}
