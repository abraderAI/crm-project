"use client";

import type { UserShadow } from "@/lib/api-types";

// --- Ban / Unban Dialog ---

export interface UserBanDialogProps {
  user: Pick<UserShadow, "is_banned" | "display_name" | "email">;
  banReason: string;
  setBanReason: (v: string) => void;
  banning: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function UserBanDialog({
  user,
  banReason,
  setBanReason,
  banning,
  onConfirm,
  onCancel,
}: UserBanDialogProps): React.ReactNode {
  return (
    <div
      data-testid="ban-confirm-dialog"
      className="rounded-lg border border-border bg-background p-6 shadow-lg"
    >
      <h3 className="text-base font-semibold text-foreground">
        {user.is_banned ? "Unban User" : "Ban User"}
      </h3>
      <p className="mt-1 text-sm text-muted-foreground">
        {user.is_banned
          ? `Are you sure you want to unban ${user.display_name || user.email}?`
          : `Are you sure you want to ban ${user.display_name || user.email}?`}
      </p>

      {!user.is_banned && (
        <div className="mt-3">
          <label htmlFor="ban-reason" className="text-sm font-medium text-foreground">
            Reason (optional)
          </label>
          <textarea
            id="ban-reason"
            data-testid="ban-reason-input"
            value={banReason}
            onChange={(e) => setBanReason(e.target.value)}
            placeholder="Enter reason for ban..."
            rows={3}
            className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>
      )}

      <div className="mt-4 flex gap-2">
        <button
          data-testid="ban-confirm-btn"
          onClick={onConfirm}
          disabled={banning}
          className={`rounded-md px-3 py-2 text-sm font-medium text-white disabled:opacity-50 ${
            user.is_banned ? "bg-green-600 hover:bg-green-700" : "bg-red-600 hover:bg-red-700"
          }`}
        >
          {banning ? "Processing..." : "Confirm"}
        </button>
        <button
          data-testid="ban-cancel-btn"
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

export interface UserPurgeDialogProps {
  userEmail: string;
  purgeEmail: string;
  setPurgeEmail: (v: string) => void;
  purging: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function UserPurgeDialog({
  userEmail,
  purgeEmail,
  setPurgeEmail,
  purging,
  onConfirm,
  onCancel,
}: UserPurgeDialogProps): React.ReactNode {
  const purgeEmailMatch = purgeEmail === userEmail;

  return (
    <div
      data-testid="purge-confirm-dialog"
      className="rounded-lg border border-red-200 bg-red-50 p-6 shadow-lg"
    >
      <h3 className="text-base font-semibold text-red-900">GDPR Purge — Irreversible</h3>
      <p className="mt-1 text-sm text-red-700">
        This will permanently delete all data for this user. Type the user&apos;s email to confirm:
      </p>
      <p className="mt-1 text-sm font-mono font-medium text-red-800">{userEmail}</p>

      <input
        data-testid="purge-email-input"
        type="text"
        value={purgeEmail}
        onChange={(e) => setPurgeEmail(e.target.value)}
        placeholder="Type user email to confirm..."
        className="mt-3 w-full rounded-md border border-red-300 bg-white px-3 py-2 text-sm text-foreground"
      />

      <div className="mt-4 flex gap-2">
        <button
          data-testid="purge-confirm-btn"
          onClick={onConfirm}
          disabled={!purgeEmailMatch || purging}
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

// --- Add to Org Form ---

export interface UserAddToOrgFormProps {
  addToOrgSlug: string;
  setAddToOrgSlug: (v: string) => void;
  addToOrgRole: string;
  setAddToOrgRole: (v: string) => void;
  addingToOrg: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

export function UserAddToOrgForm({
  addToOrgSlug,
  setAddToOrgSlug,
  addToOrgRole,
  setAddToOrgRole,
  addingToOrg,
  onSubmit,
  onCancel,
}: UserAddToOrgFormProps): React.ReactNode {
  return (
    <div
      data-testid="add-to-org-form"
      className="rounded-lg border border-border bg-background p-4"
    >
      <h3 className="text-sm font-semibold text-foreground">Add to Organization</h3>
      <div className="mt-3 flex flex-wrap items-end gap-3">
        <div>
          <label htmlFor="add-to-org-slug" className="text-xs font-medium text-foreground">
            Org Slug <span className="text-red-500">*</span>
          </label>
          <input
            id="add-to-org-slug"
            data-testid="add-to-org-slug-input"
            type="text"
            value={addToOrgSlug}
            onChange={(e) => setAddToOrgSlug(e.target.value)}
            placeholder="my-org-slug"
            className="mt-1 block w-48 rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          />
        </div>
        <div>
          <label htmlFor="add-to-org-role" className="text-xs font-medium text-foreground">
            Role
          </label>
          <select
            id="add-to-org-role"
            data-testid="add-to-org-role-select"
            value={addToOrgRole}
            onChange={(e) => setAddToOrgRole(e.target.value)}
            className="mt-1 block rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
          >
            <option value="member">Member</option>
            <option value="admin">Admin</option>
            <option value="owner">Owner</option>
          </select>
        </div>
        <button
          data-testid="add-to-org-submit"
          onClick={onSubmit}
          disabled={addingToOrg || !addToOrgSlug.trim()}
          className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {addingToOrg ? "Adding..." : "Add"}
        </button>
        <button
          data-testid="add-to-org-cancel"
          onClick={onCancel}
          className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}
