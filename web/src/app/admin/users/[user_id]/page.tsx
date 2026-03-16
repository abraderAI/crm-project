import { fetchAdminUser, fetchMemberships } from "@/lib/admin-api";
import { UserDetail } from "@/components/admin/user-detail";

interface AdminUserDetailPageProps {
  params: Promise<{ user_id: string }>;
}

export default async function AdminUserDetailPage({
  params,
}: AdminUserDetailPageProps): Promise<React.ReactNode> {
  const { user_id } = await params;
  const [user, membershipsRes] = await Promise.all([
    fetchAdminUser(user_id),
    fetchMemberships("default", { user_id }),
  ]);

  return (
    <div data-testid="admin-user-detail-page" className="flex flex-col gap-4">
      <UserDetail user={user} memberships={membershipsRes.data} />
    </div>
  );
}
