"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Shield, UserMinus, ChevronDown } from "lucide-react";
import type { OrgMembership, Role } from "@/lib/api-types";
import { fetchOrgMembers, updateMemberRole, removeMember } from "@/lib/org-api";

/** Available roles for the role selector. */
const AVAILABLE_ROLES: Role[] = ["viewer", "commenter", "contributor", "moderator", "admin"];

/** Role badge color map. */
const ROLE_STYLES: Record<string, string> = {
  owner: "bg-purple-100 text-purple-800",
  admin: "bg-red-100 text-red-800",
  moderator: "bg-orange-100 text-orange-800",
  contributor: "bg-blue-100 text-blue-800",
  commenter: "bg-green-100 text-green-800",
  viewer: "bg-gray-100 text-gray-800",
};

interface OrgAccessControlWidgetProps {
  /** Auth token for API calls. */
  token: string;
  /** Org ID to manage members for. */
  orgId: string;
}

/** Displays org member list with role badges and edit controls for admins. */
export function OrgAccessControlWidget({ token, orgId }: OrgAccessControlWidgetProps): ReactNode {
  const [members, setMembers] = useState<OrgMembership[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchOrgMembers(token, orgId, { limit: 50 });
      if (!mountedRef.current) return;
      setMembers(result.data);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load members.");
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [token, orgId]);

  useEffect(() => {
    mountedRef.current = true;
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, [load]);

  const handleRoleChange = useCallback(
    async (memberId: string, newRole: Role) => {
      try {
        const updated = await updateMemberRole(token, orgId, memberId, newRole);
        setMembers((prev) =>
          prev.map((m) => (m.id === memberId ? { ...m, role: updated.role } : m)),
        );
        setEditingId(null);
      } catch {
        setError("Failed to update role.");
      }
    },
    [token, orgId],
  );

  const handleRemove = useCallback(
    async (memberId: string) => {
      try {
        await removeMember(token, orgId, memberId);
        setMembers((prev) => prev.filter((m) => m.id !== memberId));
      } catch {
        setError("Failed to remove member.");
      }
    },
    [token, orgId],
  );

  if (isLoading) {
    return (
      <div data-testid="org-access-control-loading" className="animate-pulse space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-6 rounded bg-muted" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="org-access-control-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (members.length === 0) {
    return (
      <p data-testid="org-access-control-empty" className="text-sm text-muted-foreground">
        No members found.
      </p>
    );
  }

  return (
    <div data-testid="org-access-control-widget" className="space-y-2">
      {members.map((member) => {
        const badgeClass = ROLE_STYLES[member.role] ?? ROLE_STYLES["viewer"];
        const isEditing = editingId === member.id;
        const isOwnerRole = member.role === "owner";

        return (
          <div
            key={member.id}
            data-testid={`member-row-${member.id}`}
            className="flex items-center justify-between rounded p-1.5 text-sm hover:bg-accent/50"
          >
            <div className="flex items-center gap-2">
              <Shield className="h-4 w-4 text-muted-foreground" />
              <span className="text-foreground" data-testid={`member-name-${member.id}`}>
                {member.user_id}
              </span>
              <span
                data-testid={`member-role-${member.id}`}
                className={`rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}
              >
                {member.role}
              </span>
            </div>

            {!isOwnerRole && (
              <div className="flex items-center gap-1">
                {isEditing ? (
                  <select
                    data-testid={`role-select-${member.id}`}
                    className="rounded border px-1 py-0.5 text-xs"
                    value={member.role}
                    onChange={(e) => void handleRoleChange(member.id, e.target.value as Role)}
                  >
                    {AVAILABLE_ROLES.map((role) => (
                      <option key={role} value={role}>
                        {role}
                      </option>
                    ))}
                  </select>
                ) : (
                  <button
                    data-testid={`edit-role-${member.id}`}
                    type="button"
                    className="rounded p-0.5 hover:bg-accent"
                    onClick={() => setEditingId(member.id)}
                    aria-label={`Edit role for ${member.user_id}`}
                  >
                    <ChevronDown className="h-3 w-3" />
                  </button>
                )}
                <button
                  data-testid={`remove-member-${member.id}`}
                  type="button"
                  className="rounded p-0.5 text-destructive hover:bg-destructive/10"
                  onClick={() => void handleRemove(member.id)}
                  aria-label={`Remove ${member.user_id}`}
                >
                  <UserMinus className="h-3 w-3" />
                </button>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
