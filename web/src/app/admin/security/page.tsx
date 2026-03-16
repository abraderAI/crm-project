import { fetchRecentLogins, fetchFailedAuths } from "@/lib/admin-api";
import { SecurityTabs } from "@/components/admin/security-tabs";

export default async function AdminSecurityPage(): Promise<React.ReactNode> {
  const [loginsResult, failedResult] = await Promise.all([fetchRecentLogins(), fetchFailedAuths()]);

  return (
    <div data-testid="admin-security">
      <SecurityTabs
        recentLogins={loginsResult.data}
        failedAuths={failedResult.data}
        loginsHasMore={loginsResult.page_info.has_more}
        failedHasMore={failedResult.page_info.has_more}
      />
    </div>
  );
}
