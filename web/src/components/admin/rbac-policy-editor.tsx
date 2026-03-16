"use client";

import { useCallback, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { Save, ShieldCheck } from "lucide-react";

import type { EffectivePolicy } from "@/lib/api-types";
import { clientMutate } from "@/lib/api-client";

const RESOLUTION_STRATEGIES = ["highest_role", "lowest_role", "most_specific"];

export interface RBACPolicyEditorProps {
  /** Current effective RBAC policy from the server. */
  policy: EffectivePolicy;
}

/** Admin RBAC policy editor — structured form for resolution strategy, hierarchy, and defaults. */
export function RBACPolicyEditor({ policy }: RBACPolicyEditorProps): React.ReactNode {
  const { getToken } = useAuth();
  const [strategy, setStrategy] = useState(policy.resolution.strategy);
  const [orgDefault, setOrgDefault] = useState(policy.defaults.org_member_role);
  const [spaceDefault, setSpaceDefault] = useState(policy.defaults.space_member_role);
  const [boardDefault, setBoardDefault] = useState(policy.defaults.board_member_role);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">("idle");

  const roles = policy.roles.hierarchy;

  const handleSave = useCallback(async () => {
    setSaving(true);
    setSaveStatus("idle");
    try {
      const token = await getToken();
      if (!token) return;
      await clientMutate<EffectivePolicy>("PATCH", "/admin/rbac-policy", {
        token,
        body: {
          defaults: {
            org_member_role: orgDefault,
            space_member_role: spaceDefault,
            board_member_role: boardDefault,
          },
        },
      });
      setSaveStatus("success");
    } catch {
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  }, [getToken, orgDefault, spaceDefault, boardDefault]);

  return (
    <div data-testid="rbac-policy-editor" className="flex flex-col gap-6">
      <div className="flex items-center gap-2">
        <ShieldCheck className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold text-foreground">RBAC Policy Editor</h2>
      </div>

      {/* Resolution strategy */}
      <div className="rounded-lg border border-border p-4">
        <h3 className="mb-3 text-sm font-medium text-foreground">Resolution Strategy</h3>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <label htmlFor="strategy-select" className="text-xs text-muted-foreground">
              Strategy
            </label>
            <select
              id="strategy-select"
              data-testid="strategy-select"
              value={strategy}
              onChange={(e) => setStrategy(e.target.value)}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            >
              {RESOLUTION_STRATEGIES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </div>
          <div className="flex flex-col gap-1" data-testid="resolution-order">
            <span className="text-xs text-muted-foreground">Resolution Order</span>
            <span className="text-sm text-foreground">{policy.resolution.order.join(" → ")}</span>
          </div>
        </div>
      </div>

      {/* Role hierarchy */}
      <div className="rounded-lg border border-border p-4">
        <h3 className="mb-3 text-sm font-medium text-foreground">Role Hierarchy</h3>
        <div className="flex flex-col gap-2" data-testid="role-hierarchy">
          {roles.map((role, index) => (
            <div
              key={role}
              data-testid={`hierarchy-role-${role}`}
              className="flex items-center gap-3 rounded-md border border-border px-3 py-2"
            >
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-medium text-muted-foreground">
                {index + 1}
              </span>
              <span className="text-sm font-medium text-foreground">{role}</span>
              <span
                className="ml-auto text-xs text-muted-foreground"
                data-testid={`role-permissions-${role}`}
              >
                {(policy.roles.permissions[role] ?? []).join(", ")}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Default role assignments */}
      <div className="rounded-lg border border-border p-4">
        <h3 className="mb-3 text-sm font-medium text-foreground">Default Role Assignments</h3>
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="flex flex-col gap-1">
            <label htmlFor="default-org-role" className="text-xs text-muted-foreground">
              Org Member Role
            </label>
            <select
              id="default-org-role"
              data-testid="default-org-role"
              value={orgDefault}
              onChange={(e) => setOrgDefault(e.target.value)}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            >
              {roles.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="default-space-role" className="text-xs text-muted-foreground">
              Space Member Role
            </label>
            <select
              id="default-space-role"
              data-testid="default-space-role"
              value={spaceDefault}
              onChange={(e) => setSpaceDefault(e.target.value)}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            >
              {roles.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="default-board-role" className="text-xs text-muted-foreground">
              Board Member Role
            </label>
            <select
              id="default-board-role"
              data-testid="default-board-role"
              value={boardDefault}
              onChange={(e) => setBoardDefault(e.target.value)}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
            >
              {roles.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Save button + status */}
      <div className="flex items-center gap-3">
        <button
          data-testid="policy-save-btn"
          disabled={saving}
          onClick={handleSave}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving…" : "Save Policy"}
        </button>
        {saveStatus === "success" && (
          <span data-testid="save-success" className="text-sm text-green-600">
            Policy saved successfully.
          </span>
        )}
        {saveStatus === "error" && (
          <span data-testid="save-error" className="text-sm text-destructive">
            Failed to save policy.
          </span>
        )}
      </div>
    </div>
  );
}
