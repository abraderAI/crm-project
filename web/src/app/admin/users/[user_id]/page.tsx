import { fetchAdminUser } from "@/lib/admin-api";
import { UserDetail } from "@/components/admin/user-detail";
import type { OrgMembershipEnriched } from "@/lib/api-types";

interface AdminUserDetailPageProps {
  params: Promise<{ user_id: string }>;
}

/**
 * Admin user detail page.
 * The backend GET /v1/admin/users/{id} returns enriched memberships with org names,
 * so we use those directly instead of a separate memberships fetch.
 */
export default async function AdminUserDetailPage({
  params,
}: AdminUserDetailPageProps): Promise<React.ReactNode> {
  const { user_id } = await params;
  const detail = await fetchAdminUser(user_id);

  return (
    <div data-testid="admin-user-detail-page" className="flex flex-col gap-4">
      <UserDetail
        user={detail}
        memberships={
          (detail as unknown as { memberships?: OrgMembershipEnriched[] }).memberships ?? []
        }
      />
    </div>
  );
}
