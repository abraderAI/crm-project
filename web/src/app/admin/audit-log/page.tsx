import { fetchAuditLog } from "@/lib/admin-api";
import { AuditLogViewerWithDirectory } from "@/components/admin/audit-log-viewer-wrapper";

export default async function AdminAuditLogPage(): Promise<React.ReactNode> {
  const { data: entries, page_info } = await fetchAuditLog();

  return (
    <div data-testid="admin-audit-log">
      <AuditLogViewerWithDirectory entries={entries} hasMore={page_info.has_more} />
    </div>
  );
}
