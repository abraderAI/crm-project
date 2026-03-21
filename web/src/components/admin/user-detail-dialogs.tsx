"use client";

import { useMemo, useState } from "react";
import { Building2, Plus, Search } from "lucide-react";
import type { AdminOrgDetail, UserShadow } from "@/lib/api-types";

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
  /** Available orgs to select from. */
  orgs: AdminOrgDetail[];
  /** Whether the org list is still loading. */
  orgsLoading: boolean;
  /** Currently selected org, or null if none. */
  selectedOrg: AdminOrgDetail | null;
  /** Select an org from the list. */
  setSelectedOrg: (org: AdminOrgDetail | null) => void;
  /** Role to assign in the org. */
  addToOrgRole: string;
  setAddToOrgRole: (v: string) => void;
  /** Whether the add-to-org mutation is in progress. */
  addingToOrg: boolean;
  /** Called to submit the selected org + role. */
  onSubmit: () => void;
  /** Called to close the form. */
  onCancel: () => void;
  /** Called to create a new org; resolves with the created org. */
  onCreateOrg: (name: string, description?: string) => Promise<AdminOrgDetail>;
}

export function UserAddToOrgForm({
  orgs,
  orgsLoading,
  selectedOrg,
  setSelectedOrg,
  addToOrgRole,
  setAddToOrgRole,
  addingToOrg,
  onSubmit,
  onCancel,
  onCreateOrg,
}: UserAddToOrgFormProps): React.ReactNode {
  const [search, setSearch] = useState("");
  const [showCreateOrg, setShowCreateOrg] = useState(false);
  const [newOrgName, setNewOrgName] = useState("");
  const [newOrgDesc, setNewOrgDesc] = useState("");
  const [creatingOrg, setCreatingOrg] = useState(false);
  const [createError, setCreateError] = useState("");

  const filteredOrgs = useMemo(() => {
    if (!search.trim()) return orgs;
    const q = search.toLowerCase();
    return orgs.filter((o) => o.name.toLowerCase().includes(q) || o.slug.toLowerCase().includes(q));
  }, [orgs, search]);

  const handleCreateOrg = async (): Promise<void> => {
    if (!newOrgName.trim()) return;
    setCreateError("");
    setCreatingOrg(true);
    try {
      const created = await onCreateOrg(newOrgName.trim(), newOrgDesc.trim() || undefined);
      setSelectedOrg(created);
      setNewOrgName("");
      setNewOrgDesc("");
      setShowCreateOrg(false);
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to create org");
    } finally {
      setCreatingOrg(false);
    }
  };

  return (
    <div
      data-testid="add-to-org-form"
      className="rounded-lg border border-border bg-background p-4"
    >
      <h3 className="text-sm font-semibold text-foreground">Add to Organization</h3>

      {/* Selected org indicator */}
      {selectedOrg && (
        <div
          data-testid="selected-org-indicator"
          className="mt-2 flex items-center gap-2 rounded-md bg-green-50 px-3 py-2 text-sm text-green-800"
        >
          <Building2 className="h-4 w-4" />
          <span>
            Selected: <strong>{selectedOrg.name}</strong> ({selectedOrg.slug})
          </span>
          <button
            data-testid="clear-selected-org"
            onClick={() => setSelectedOrg(null)}
            className="ml-auto text-xs text-green-600 hover:text-green-800"
          >
            Change
          </button>
        </div>
      )}

      {/* Org picker (shown when no org is selected) */}
      {!selectedOrg && (
        <div className="mt-3">
          {orgsLoading ? (
            <p
              data-testid="orgs-loading"
              className="py-4 text-center text-sm text-muted-foreground"
            >
              Loading organizations...
            </p>
          ) : (
            <>
              {/* Search + Create new row */}
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input
                    data-testid="org-search-input"
                    type="text"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder="Search orgs by name or slug..."
                    className="w-full rounded-md border border-border bg-background py-2 pl-9 pr-3 text-sm text-foreground"
                  />
                </div>
                <button
                  data-testid="show-create-org-btn"
                  onClick={() => setShowCreateOrg((v) => !v)}
                  className="inline-flex items-center gap-1 rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
                >
                  <Plus className="h-4 w-4" />
                  New Org
                </button>
              </div>

              {/* Inline create org form */}
              {showCreateOrg && (
                <div
                  data-testid="create-org-inline"
                  className="mt-2 rounded-md border border-dashed border-border bg-muted/30 p-3"
                >
                  <div className="flex flex-col gap-2">
                    <input
                      data-testid="new-org-name-input"
                      type="text"
                      value={newOrgName}
                      onChange={(e) => setNewOrgName(e.target.value)}
                      placeholder="Organization name *"
                      className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                    />
                    <input
                      data-testid="new-org-desc-input"
                      type="text"
                      value={newOrgDesc}
                      onChange={(e) => setNewOrgDesc(e.target.value)}
                      placeholder="Description (optional)"
                      className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                    />
                    {createError && (
                      <p data-testid="create-org-error" className="text-xs text-red-600">
                        {createError}
                      </p>
                    )}
                    <div className="flex gap-2">
                      <button
                        data-testid="create-org-submit-btn"
                        onClick={() => void handleCreateOrg()}
                        disabled={creatingOrg || !newOrgName.trim()}
                        className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                      >
                        {creatingOrg ? "Creating..." : "Create & Select"}
                      </button>
                      <button
                        data-testid="create-org-cancel-btn"
                        onClick={() => {
                          setShowCreateOrg(false);
                          setNewOrgName("");
                          setNewOrgDesc("");
                          setCreateError("");
                        }}
                        className="rounded-md bg-muted px-3 py-1.5 text-sm font-medium text-foreground hover:bg-muted/80"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* Org list */}
              <div
                data-testid="org-picker-list"
                className="mt-2 max-h-48 overflow-y-auto rounded-md border border-border divide-y divide-border"
              >
                {filteredOrgs.length === 0 ? (
                  <p
                    data-testid="org-picker-empty"
                    className="px-3 py-4 text-center text-sm text-muted-foreground"
                  >
                    {orgs.length === 0
                      ? "No organizations exist yet."
                      : "No orgs match your search."}
                  </p>
                ) : (
                  filteredOrgs.map((org) => (
                    <button
                      key={org.id}
                      data-testid={`org-picker-item-${org.id}`}
                      onClick={() => setSelectedOrg(org)}
                      className="flex w-full items-center gap-3 px-3 py-2 text-left hover:bg-muted/50 transition-colors"
                    >
                      <Building2 className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <div className="flex flex-col">
                        <span className="text-sm font-medium text-foreground">{org.name}</span>
                        <span className="text-xs text-muted-foreground">{org.slug}</span>
                      </div>
                      <span className="ml-auto text-xs text-muted-foreground">
                        {org.member_count} members
                      </span>
                    </button>
                  ))
                )}
              </div>
            </>
          )}
        </div>
      )}

      {/* Role selector + actions (always visible) */}
      <div className="mt-3 flex flex-wrap items-end gap-3">
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
          disabled={addingToOrg || !selectedOrg}
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
