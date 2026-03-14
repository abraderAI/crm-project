import { fetchMemberships } from "@/lib/admin-api";
import { MembershipView } from "@/components/admin/membership-view";

export default async function AdminMembersPage(): Promise<React.ReactNode> {
  const memberships = await fetchMemberships();

  return <MembershipView initialMembers={memberships.data} />;
}
