import { fetchAdminOrgs } from "@/lib/admin-api";
import { OrgManager } from "@/components/admin/org-manager";

/** Admin orgs list page — displays all orgs via OrgManager. */
export default async function AdminOrgsPage(): Promise<React.ReactNode> {
  const { data: orgs } = await fetchAdminOrgs();

  return <OrgManager initialOrgs={orgs} />;
}
