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
      // Backend uses the member's user_id as path param, not the membership UUID.
      const member = members.find((m) => m.id === membershipId);
      if (!member) return;
      const updated = await changeMembershipRole(token, member.user_id, newRole);
      setMembers((prev) => prev.map((m) => (m.id === membershipId ? toItem(updated) : m)));
    },
    [getToken, members],
  );

  const handleRemove = useCallback(
    async (membershipId: string) => {
      const token = await getToken();
      if (!token) return;
      // Backend uses the member's user_id as path param, not the membership UUID.
      const member = members.find((m) => m.id === membershipId);
      if (!member) return;
      await removeMembership(token, member.user_id);
      setMembers((prev) => prev.filter((m) => m.id !== membershipId));
    },
    [getToken, members],
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
