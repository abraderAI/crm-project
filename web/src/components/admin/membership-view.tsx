"use client";

import { useCallback, useState } from "react";
import { useAuth } from "@clerk/nextjs";

import type { OrgMembership, Role } from "@/lib/api-types";
import { addMembership, changeMembershipRole, removeMembership } from "@/lib/entity-api";
import { MembershipManager, type MembershipItem } from "./membership-manager";

export interface MembershipViewProps {
  /** Initial memberships fetched on the server. */
  initialMembers: OrgMembership[];
}

/** Map OrgMembership to MembershipItem for the presentational component. */
function toItem(m: OrgMembership): MembershipItem {
  return { id: m.id, user_id: m.user_id, role: m.role };
}

/** Client wrapper wiring MembershipManager to entity-api mutations via Clerk auth. */
export function MembershipView({ initialMembers }: MembershipViewProps): React.ReactNode {
  const { getToken } = useAuth();
  const [members, setMembers] = useState<MembershipItem[]>(initialMembers.map(toItem));

  const handleAdd = useCallback(
    async (userId: string, role: Role) => {
      const token = await getToken();
      if (!token) return;
      const created = await addMembership(token, userId, role);
      setMembers((prev) => [...prev, toItem(created)]);
    },
    [getToken],
  );

  const handleChangeRole = useCallback(
    async (membershipId: string, newRole: Role) => {
      const token = await getToken();
      if (!token) return;
      const updated = await changeMembershipRole(token, membershipId, newRole);
      setMembers((prev) => prev.map((m) => (m.id === membershipId ? toItem(updated) : m)));
    },
    [getToken],
  );

  const handleRemove = useCallback(
    async (membershipId: string) => {
      const token = await getToken();
      if (!token) return;
      await removeMembership(token, membershipId);
      setMembers((prev) => prev.filter((m) => m.id !== membershipId));
    },
    [getToken],
  );

  return (
    <MembershipManager
      members={members}
      onAdd={handleAdd}
      onChangeRole={handleChangeRole}
      onRemove={handleRemove}
      scopeLabel="Platform"
    />
  );
}
