"use client";

import { useEffect, useState } from "react";
import { buildHeaders, buildUrl, parseResponse } from "@/lib/api-client";
import type { OrgMembership, PaginatedResponse } from "@/lib/api-types";

export interface AssigneeFilterProps {
  /** Organization slug or ID to load members for. */
  orgId: string;
  /** Currently selected user ID, or null for "All". */
  value: string | null;
  /** Called when the user selects a different assignee. */
  onChange: (userId: string | null) => void;
  /** Auth token for client-side fetch. */
  token?: string | null;
}

/** Member option derived from membership data. */
interface MemberOption {
  userId: string;
  label: string;
}

/** Dropdown filter for selecting an assignee from org members. */
export function AssigneeFilter({
  orgId,
  value,
  onChange,
  token,
}: AssigneeFilterProps): React.ReactNode {
  const [members, setMembers] = useState<MemberOption[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    async function fetchMembers(): Promise<void> {
      try {
        const url = buildUrl(`/orgs/${orgId}/members`);
        const response = await fetch(url, {
          method: "GET",
          headers: buildHeaders(token),
        });
        const data = await parseResponse<PaginatedResponse<OrgMembership>>(response);
        if (active) {
          setMembers(
            data.data.map((m) => ({
              userId: m.user_id,
              label: m.user_id,
            })),
          );
        }
      } catch {
        // Silently handle error — members list stays empty.
      } finally {
        if (active) setLoading(false);
      }
    }

    void fetchMembers();
    return () => {
      active = false;
    };
  }, [orgId, token]);

  function handleChange(e: React.ChangeEvent<HTMLSelectElement>): void {
    const selected = e.target.value;
    onChange(selected === "" ? null : selected);
  }

  return (
    <div data-testid="assignee-filter">
      <select
        data-testid="assignee-select"
        value={value ?? ""}
        onChange={handleChange}
        disabled={loading}
        className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent/50 disabled:opacity-50"
      >
        <option value="" data-testid="assignee-option-all">
          All assignees
        </option>
        {members.map((m) => (
          <option key={m.userId} value={m.userId} data-testid={`assignee-option-${m.userId}`}>
            {m.label}
          </option>
        ))}
      </select>
    </div>
  );
}
