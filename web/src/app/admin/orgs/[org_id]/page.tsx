import { notFound } from "next/navigation";
import { fetchAdminOrg } from "@/lib/admin-api";
import { OrgDetailAdmin } from "@/components/admin/org-detail-admin";

interface Props {
  params: Promise<{ org_id: string }>;
}

/** Admin org detail page — fetches org by ID and renders OrgDetailAdmin. */
export default async function AdminOrgDetailPage({ params }: Props): Promise<React.ReactNode> {
  const { org_id } = await params;

  let org;
  try {
    org = await fetchAdminOrg(org_id);
  } catch {
    notFound();
  }

  return <OrgDetailAdmin org={org} />;
}
