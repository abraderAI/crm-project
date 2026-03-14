import { fetchAuditLog } from "@/lib/admin-api";
import { AuditLogViewer } from "@/components/admin/audit-log-viewer";

export default async function AdminAuditLogPage(): Promise<React.ReactNode> {
  const { data: entries, page_info } = await fetchAuditLog();

  return (
    <div data-testid="admin-audit-log">
      <AuditLogViewer entries={entries} hasMore={page_info.has_more} />
    </div>
  );
}
