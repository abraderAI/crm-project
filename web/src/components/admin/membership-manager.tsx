"use client";

import { useState } from "react";
import { Users, Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Role } from "@/lib/api-types";

export interface MembershipItem {
  id: string;
  user_id: string;
  role: Role;
}

export interface MembershipManagerProps {
  /** Current members. */
  members: MembershipItem[];
  /** Whether data is loading. */
  loading?: boolean;
  /** Called when a member is added. */
  onAdd: (userId: string, role: Role) => void;
  /** Called when a member's role is changed. */
  onChangeRole: (membershipId: string, newRole: Role) => void;
  /** Called when a member is removed. */
  onRemove: (membershipId: string) => void;
  /** Scope label for display, e.g. "Organization", "Space". */
  scopeLabel?: string;
}

const ROLES: Role[] = ["viewer", "commenter", "contributor", "moderator", "admin", "owner"];

/** Membership management — list, add, change role, remove members. */
export function MembershipManager({
  members,
  loading = false,
  onAdd,
  onChangeRole,
  onRemove,
  scopeLabel = "Organization",
}: MembershipManagerProps): React.ReactNode {
  const [showAdd, setShowAdd] = useState(false);
  const [newUserId, setNewUserId] = useState("");
  const [newRole, setNewRole] = useState<Role>("viewer");
  const [error, setError] = useState("");

  const handleAdd = (e: React.FormEvent): void => {
    e.preventDefault();
    if (!newUserId.trim()) {
      setError("User ID is required.");
      return;
    }
    setError("");
    onAdd(newUserId.trim(), newRole);
    setNewUserId("");
    setNewRole("viewer");
    setShowAdd(false);
  };

  return (
    <div data-testid="membership-manager" className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Users className="h-5 w-5 text-muted-foreground" data-testid="membership-icon" />
          <h2 className="text-lg font-semibold text-foreground">{scopeLabel} Members</h2>
          <span
            className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
            data-testid="member-count"
          >
            {members.length}
          </span>
        </div>
        <button
          onClick={() => setShowAdd(!showAdd)}
          data-testid="member-add-toggle"
          className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Add Member
        </button>
      </div>

      {/* Add form */}
      {showAdd && (
        <form
          onSubmit={handleAdd}
          data-testid="member-add-form"
          className="rounded-lg border border-border p-4"
        >
          <div className="flex flex-col gap-3">
            <div className="flex flex-col gap-1">
              <label htmlFor="member-user-id" className="text-sm font-medium text-foreground">
                User ID
              </label>
              <input
                id="member-user-id"
                type="text"
                value={newUserId}
                onChange={(e) => setNewUserId(e.target.value)}
                placeholder="Enter user ID..."
                data-testid="member-user-input"
                className={cn(
                  "rounded-md border bg-background px-3 py-2 text-sm text-foreground",
                  error ? "border-destructive" : "border-border",
                )}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label htmlFor="member-role" className="text-sm font-medium text-foreground">
                Role
              </label>
              <select
                id="member-role"
                value={newRole}
                onChange={(e) => setNewRole(e.target.value as Role)}
                data-testid="member-role-select"
                className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </select>
            </div>
            {error && (
              <p className="text-xs text-destructive" data-testid="member-error">
                {error}
              </p>
            )}
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={loading}
                data-testid="member-save-btn"
                className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                Add
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowAdd(false);
                  setError("");
                }}
                data-testid="member-cancel-btn"
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
          data-testid="membership-loading"
        >
          Loading members...
        </div>
      )}

      {/* Empty */}
      {!loading && members.length === 0 && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="membership-empty"
        >
          No members.
        </div>
      )}

      {/* Member list */}
      {!loading && members.length > 0 && (
        <div
          className="divide-y divide-border rounded-lg border border-border"
          data-testid="member-list"
        >
          {members.map((m) => (
            <div
              key={m.id}
              className="flex items-center gap-3 px-4 py-2.5"
              data-testid={`member-item-${m.id}`}
            >
              <span
                className="text-sm font-medium text-foreground"
                data-testid={`member-user-${m.id}`}
              >
                {m.user_id}
              </span>
              <select
                value={m.role}
                onChange={(e) => onChangeRole(m.id, e.target.value as Role)}
                data-testid={`member-role-${m.id}`}
                className="ml-auto rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground"
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </select>
              <button
                onClick={() => onRemove(m.id)}
                data-testid={`member-remove-${m.id}`}
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
