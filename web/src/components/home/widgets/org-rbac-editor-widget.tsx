"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Key, Save } from "lucide-react";
import type { OrgMembership, Role, Space } from "@/lib/api-types";
import { fetchOrgMembers, fetchOrgSpaces, updateSpaceRoleOverride } from "@/lib/org-api";

/** Available roles for space-level overrides. */
const OVERRIDE_ROLES: (Role | "inherit")[] = [
  "inherit",
  "viewer",
  "commenter",
  "contributor",
  "moderator",
  "admin",
];

interface OrgRBACEditorWidgetProps {
  /** Auth token for API calls. */
  token: string;
  /** Org ID for space and member lookups. */
  orgId: string;
}

/** Allows admins to set space-level role overrides per member. */
export function OrgRBACEditorWidget({ token, orgId }: OrgRBACEditorWidgetProps): ReactNode {
  const [members, setMembers] = useState<OrgMembership[]>([]);
  const [spaces, setSpaces] = useState<Space[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedMember, setSelectedMember] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const [membersResult, spacesResult] = await Promise.all([
        fetchOrgMembers(token, orgId, { limit: 50 }),
        fetchOrgSpaces(token, orgId),
      ]);
      if (!mountedRef.current) return;
      setMembers(membersResult.data);
      setSpaces(spacesResult.data);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load RBAC data.");
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

  const handleOverrideChange = useCallback(
    async (memberId: string, spaceId: string, role: string) => {
      setSaving(true);
      try {
        const resolvedRole = role === "inherit" ? null : (role as Role);
        await updateSpaceRoleOverride(token, orgId, memberId, spaceId, resolvedRole);
      } catch {
        setError("Failed to save role override.");
      } finally {
        if (mountedRef.current) setSaving(false);
      }
    },
    [token, orgId],
  );

  if (isLoading) {
    return (
      <div data-testid="org-rbac-editor-loading" className="animate-pulse space-y-2">
        <div className="h-4 w-3/4 rounded bg-muted" />
        <div className="h-8 rounded bg-muted" />
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="org-rbac-editor-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (members.length === 0 || spaces.length === 0) {
    return (
      <p data-testid="org-rbac-editor-empty" className="text-sm text-muted-foreground">
        No members or spaces available for RBAC configuration.
      </p>
    );
  }

  return (
    <div data-testid="org-rbac-editor-widget" className="space-y-3">
      <div className="flex items-center gap-2">
        <Key className="h-4 w-4 text-primary" />
        <span className="text-sm font-medium text-foreground">Space Role Overrides</span>
      </div>

      <div>
        <label htmlFor="rbac-member-select" className="mb-1 block text-xs text-muted-foreground">
          Select member
        </label>
        <select
          id="rbac-member-select"
          data-testid="rbac-member-select"
          className="w-full rounded border px-2 py-1 text-sm"
          value={selectedMember ?? ""}
          onChange={(e) => setSelectedMember(e.target.value || null)}
        >
          <option value="">Choose a member...</option>
          {members.map((m) => (
            <option key={m.id} value={m.id}>
              {m.user_id} ({m.role})
            </option>
          ))}
        </select>
      </div>

      {selectedMember && (
        <div data-testid="rbac-space-overrides" className="space-y-1">
          {spaces.map((space) => (
            <div key={space.id} className="flex items-center justify-between text-xs">
              <span data-testid={`rbac-space-name-${space.id}`}>{space.name}</span>
              <div className="flex items-center gap-1">
                <select
                  data-testid={`rbac-role-select-${space.id}`}
                  className="rounded border px-1 py-0.5 text-xs"
                  defaultValue="inherit"
                  onChange={(e) =>
                    void handleOverrideChange(selectedMember, space.id, e.target.value)
                  }
                >
                  {OVERRIDE_ROLES.map((role) => (
                    <option key={role} value={role}>
                      {role === "inherit" ? "Inherit from org" : role}
                    </option>
                  ))}
                </select>
                {saving && <Save className="h-3 w-3 animate-pulse text-muted-foreground" />}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
