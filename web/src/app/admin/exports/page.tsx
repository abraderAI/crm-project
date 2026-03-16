import { fetchExports } from "@/lib/admin-api";
import { ExportManager } from "@/components/admin/export-manager";

export default async function AdminExportsPage(): Promise<React.ReactNode> {
  const exports = await fetchExports();

  return <ExportManager initialExports={exports.data} />;
}
