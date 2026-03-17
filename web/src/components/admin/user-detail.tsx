"use client";

import { useCallback, useEffect, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { useRouter } from "next/navigation";
import { Ban, Crown, ShieldCheck, Trash2, UserCog, UserPlus, X, AlertTriangle } from "lucide-react";
import type { UserShadow, OrgMembership, ImpersonationResponse } from "@/lib/api-types";
import { clientMutate } from "@/lib/api-client";

/** Format a date string for display. */
function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

/** Format remaining time as "Xh Xm Xs". */
function formatCountdown(ms: number): string {
  if (ms <= 0) return "Expired";
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return `${hours}h ${minutes}m ${seconds}s`;
}

export interface UserDetailProps {
  /** User data from the admin API. */
  user: UserShadow;
  /** Cross-org memberships for this user. */
  memberships: OrgMembership[];
}

/** Admin user detail component with ban/unban, GDPR purge, and impersonation. */
export function UserDetail({ user: initialUser, memberships }: UserDetailProps): React.ReactNode {
  const { getToken } = useAuth();
  const router = useRouter();

  const [user, setUser] = useState<UserShadow>(initialUser);
  const [error, setError] = useState("");

  // --- Ban/unban state ---
  const [showBanDialog, setShowBanDialog] = useState(false);
  const [banReason, setBanReason] = useState("");
  const [banning, setBanning] = useState(false);

  // --- Purge state ---
  const [showPurgeDialog, setShowPurgeDialog] = useState(false);
  const [purgeEmail, setPurgeEmail] = useState("");
  const [purging, setPurging] = useState(false);

  // --- Impersonation state ---
  const [impersonating, setImpersonating] = useState(false);
  const [impersonationExpiresAt, setImpersonationExpiresAt] = useState<string | null>(null);
  const [countdown, setCountdown] = useState("");

  // --- Add to org state ---
  const [showAddToOrgForm, setShowAddToOrgForm] = useState(false);
  const [addToOrgSlug, setAddToOrgSlug] = useState("");
  const [addToOrgRole, setAddToOrgRole] = useState("member");
  const [addingToOrg, setAddingToOrg] = useState(false);
  const [addToOrgSuccess, setAddToOrgSuccess] = useState("");

  // --- Promote to platform admin state ---
  const [promotingToAdmin, setPromotingToAdmin] = useState(false);
  const [promoteSuccess, setPromoteSuccess] = useState("");

  // Countdown timer for impersonation.
  useEffect(() => {
    if (!impersonationExpiresAt) return;
    const update = (): void => {
      const remaining = new Date(impersonationExpiresAt).getTime() - Date.now();
      setCountdown(formatCountdown(remaining));
      if (remaining <= 0) {
        sessionStorage.removeItem("impersonation_token");
        setImpersonating(false);
        setImpersonationExpiresAt(null);
      }
    };
    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [impersonationExpiresAt]);

  // --- Ban/unban ---
  const handleBanToggle = useCallback(async () => {
    setError("");
    setBanning(true);
    try {
      const token = await getToken();
      if (!token) return;

      if (user.is_banned) {
        const updated = await clientMutate<UserShadow>(
          "POST",
          `/admin/users/${user.clerk_user_id}/unban`,
          { token },
        );
        setUser(updated);
      } else {
        const updated = await clientMutate<UserShadow>(
          "POST",
          `/admin/users/${user.clerk_user_id}/ban`,
          { token, body: { reason: banReason } },
        );
        setUser(updated);
      }
      setShowBanDialog(false);
      setBanReason("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed");
    } finally {
      setBanning(false);
    }
  }, [getToken, user.is_banned, user.clerk_user_id, banReason]);

  // --- GDPR purge ---
  const handlePurge = useCallback(async () => {
    setError("");
    setPurging(true);
    try {
      const token = await getToken();
      if (!token) return;

      await clientMutate<void>("DELETE", `/admin/users/${user.clerk_user_id}/purge`, { token });
      router.push("/admin/users");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Purge failed");
    } finally {
      setPurging(false);
    }
  }, [getToken, user.clerk_user_id, router]);

  // --- Impersonate ---
  const handleImpersonate = useCallback(async () => {
    setError("");
    try {
      const token = await getToken();
      if (!token) return;

      const result = await clientMutate<ImpersonationResponse>(
        "POST",
        `/admin/users/${user.clerk_user_id}/impersonate`,
        { token },
      );
      sessionStorage.setItem("impersonation_token", result.token);
      setImpersonating(true);
      setImpersonationExpiresAt(result.expires_at);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Impersonation failed");
    }
  }, [getToken, user.clerk_user_id]);

  const handleClearImpersonation = useCallback(() => {
    sessionStorage.removeItem("impersonation_token");
    setImpersonating(false);
    setImpersonationExpiresAt(null);
  }, []);

  // --- Add to org ---
  const handleAddToOrg = useCallback(async () => {
    if (!addToOrgSlug.trim()) return;
    setError("");
    setAddToOrgSuccess("");
    setAddingToOrg(true);
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("POST", `/orgs/${addToOrgSlug.trim()}/members`, {
        token,
        body: { user_id: user.clerk_user_id, role: addToOrgRole },
      });
      setAddToOrgSlug("");
      setAddToOrgRole("member");
      setShowAddToOrgForm(false);
      setAddToOrgSuccess(`Added to org "${addToOrgSlug.trim()}" as ${addToOrgRole}.`);
      setTimeout(() => setAddToOrgSuccess(""), 4000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add to org");
    } finally {
      setAddingToOrg(false);
    }
  }, [getToken, user.clerk_user_id, addToOrgSlug, addToOrgRole]);

  // --- Promote to platform admin ---
  const handlePromoteToAdmin = useCallback(async () => {
    setError("");
    setPromoteSuccess("");
    setPromotingToAdmin(true);
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<void>("POST", "/admin/platform-admins", {
        token,
        body: { user_id: user.clerk_user_id },
      });
      setPromoteSuccess(`${user.display_name || user.email} promoted to platform admin.`);
      setTimeout(() => setPromoteSuccess(""), 4000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to promote user");
    } finally {
      setPromotingToAdmin(false);
    }
  }, [getToken, user.clerk_user_id, user.display_name, user.email]);

  const purgeEmailMatch = purgeEmail === user.email;

  return (
    <div data-testid="user-detail" className="flex flex-col gap-6">
      {/* Error display */}
      {error && (
        <div
          data-testid="action-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4" />
          {error}
        </div>
      )}

      {/* Impersonation active banner */}
      {impersonating && (
        <div
          data-testid="impersonation-active"
          className="flex items-center justify-between rounded-md bg-amber-50 px-4 py-3 text-sm text-amber-800"
        >
          <div className="flex items-center gap-2">
            <UserCog className="h-4 w-4" />
            <span>Impersonating {user.display_name || user.email}</span>
            <span data-testid="impersonation-countdown" className="font-mono text-xs">
              {countdown}
            </span>
          </div>
          <button
            data-testid="impersonation-clear-btn"
            onClick={handleClearImpersonation}
            className="flex items-center gap-1 rounded-md bg-amber-200 px-2 py-1 text-xs font-medium hover:bg-amber-300"
          >
            <X className="h-3 w-3" />
            Clear
          </button>
        </div>
      )}

      {/* Profile card */}
      <div className="rounded-lg border border-border p-6">
        <div className="flex items-start gap-4">
          {user.avatar_url ? (
            <img
              data-testid="user-avatar"
              src={user.avatar_url}
              alt={user.display_name || user.email}
              className="h-16 w-16 rounded-full object-cover"
            />
          ) : (
            <div
              data-testid="user-avatar-fallback"
              className="flex h-16 w-16 items-center justify-center rounded-full bg-muted text-lg font-semibold text-muted-foreground"
            >
              {(user.display_name || user.email).charAt(0).toUpperCase()}
            </div>
          )}

          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <h2 data-testid="user-display-name" className="text-lg font-semibold text-foreground">
                {user.display_name || user.email}
              </h2>
              {user.is_banned && (
                <span
                  data-testid="user-banned-badge"
                  className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800"
                >
                  Banned
                </span>
              )}
            </div>
            <p data-testid="user-email" className="text-sm text-muted-foreground">
              {user.email}
            </p>
            <div className="flex gap-4 text-xs text-muted-foreground">
              <span data-testid="user-joined">Joined: {formatDate(user.synced_at)}</span>
              <span data-testid="user-last-seen">Last seen: {formatDate(user.last_seen_at)}</span>
            </div>
            {user.is_banned && user.ban_reason && (
              <p data-testid="user-ban-reason" className="mt-1 text-xs text-red-600">
                Ban reason: {user.ban_reason}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-3">
        <button
          data-testid="ban-toggle-btn"
          onClick={() => setShowBanDialog(true)}
          className={`inline-flex items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium ${
            user.is_banned
              ? "bg-green-50 text-green-700 hover:bg-green-100"
              : "bg-red-50 text-red-700 hover:bg-red-100"
          }`}
        >
          {user.is_banned ? <ShieldCheck className="h-4 w-4" /> : <Ban className="h-4 w-4" />}
          {user.is_banned ? "Unban User" : "Ban User"}
        </button>

        <button
          data-testid="purge-btn"
          onClick={() => setShowPurgeDialog(true)}
          className="inline-flex items-center gap-1.5 rounded-md bg-red-50 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-100"
        >
          <Trash2 className="h-4 w-4" />
          GDPR Purge
        </button>

        <button
          data-testid="impersonate-btn"
          onClick={handleImpersonate}
          disabled={impersonating}
          className="inline-flex items-center gap-1.5 rounded-md bg-amber-50 px-3 py-2 text-sm font-medium text-amber-700 hover:bg-amber-100 disabled:opacity-50"
        >
          <UserCog className="h-4 w-4" />
          Impersonate
        </button>

        <button
          data-testid="add-to-org-btn"
          onClick={() => setShowAddToOrgForm((v) => !v)}
          className="inline-flex items-center gap-1.5 rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
        >
          <UserPlus className="h-4 w-4" />
          Add to Org
        </button>

        <button
          data-testid="promote-admin-btn"
          onClick={() => void handlePromoteToAdmin()}
          disabled={promotingToAdmin}
          className="inline-flex items-center gap-1.5 rounded-md bg-purple-50 px-3 py-2 text-sm font-medium text-purple-700 hover:bg-purple-100 disabled:opacity-50"
        >
          <Crown className="h-4 w-4" />
          {promotingToAdmin ? "Promoting..." : "Promote to Platform Admin"}
        </button>
      </div>

      {/* Add-to-org success */}
      {addToOrgSuccess && (
        <div
          data-testid="add-to-org-success"
          className="rounded-md bg-green-50 px-4 py-3 text-sm text-green-700"
        >
          {addToOrgSuccess}
        </div>
      )}

      {/* Promote-to-admin success */}
      {promoteSuccess && (
        <div
          data-testid="promote-admin-success"
          className="rounded-md bg-green-50 px-4 py-3 text-sm text-green-700"
        >
          {promoteSuccess}
        </div>
      )}

      {/* Add to org form */}
      {showAddToOrgForm && (
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
              onClick={() => void handleAddToOrg()}
              disabled={addingToOrg || !addToOrgSlug.trim()}
              className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {addingToOrg ? "Adding..." : "Add"}
            </button>
            <button
              data-testid="add-to-org-cancel"
              onClick={() => { setShowAddToOrgForm(false); setAddToOrgSlug(""); setAddToOrgRole("member"); }}
              className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Ban confirmation dialog */}
      {showBanDialog && (
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
              onClick={handleBanToggle}
              disabled={banning}
              className={`rounded-md px-3 py-2 text-sm font-medium text-white disabled:opacity-50 ${
                user.is_banned ? "bg-green-600 hover:bg-green-700" : "bg-red-600 hover:bg-red-700"
              }`}
            >
              {banning ? "Processing..." : "Confirm"}
            </button>
            <button
              data-testid="ban-cancel-btn"
              onClick={() => {
                setShowBanDialog(false);
                setBanReason("");
              }}
              className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Purge confirmation dialog */}
      {showPurgeDialog && (
        <div
          data-testid="purge-confirm-dialog"
          className="rounded-lg border border-red-200 bg-red-50 p-6 shadow-lg"
        >
          <h3 className="text-base font-semibold text-red-900">GDPR Purge — Irreversible</h3>
          <p className="mt-1 text-sm text-red-700">
            This will permanently delete all data for this user. Type the user&apos;s email to
            confirm:
          </p>
          <p className="mt-1 text-sm font-mono font-medium text-red-800">{user.email}</p>

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
              onClick={handlePurge}
              disabled={!purgeEmailMatch || purging}
              className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
            >
              {purging ? "Purging..." : "Permanently Delete"}
            </button>
            <button
              data-testid="purge-cancel-btn"
              onClick={() => {
                setShowPurgeDialog(false);
                setPurgeEmail("");
              }}
              className="rounded-md bg-muted px-3 py-2 text-sm font-medium text-foreground hover:bg-muted/80"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Memberships table */}
      <div>
        <h3 className="text-base font-semibold text-foreground">Organization Memberships</h3>
        {memberships.length === 0 ? (
          <p data-testid="memberships-empty" className="mt-2 text-sm text-muted-foreground">
            No organization memberships found.
          </p>
        ) : (
          <div className="mt-2 divide-y divide-border rounded-lg border border-border">
            <div className="flex gap-4 bg-muted/50 px-4 py-2 text-xs font-medium text-muted-foreground">
              <span className="w-1/3">Organization</span>
              <span className="w-1/3">Role</span>
              <span className="w-1/3">Joined</span>
            </div>
            {memberships.map((mem) => (
              <div
                key={mem.id}
                data-testid={`membership-row-${mem.id}`}
                className="flex gap-4 px-4 py-3 text-sm"
              >
                <span data-testid={`membership-org-${mem.id}`} className="w-1/3 text-foreground">
                  {mem.org_id}
                </span>
                <span
                  data-testid={`membership-role-${mem.id}`}
                  className="w-1/3 capitalize text-foreground"
                >
                  {mem.role}
                </span>
                <span className="w-1/3 text-muted-foreground">{formatDate(mem.created_at)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
