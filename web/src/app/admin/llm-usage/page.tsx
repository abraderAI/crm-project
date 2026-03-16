import { fetchLlmUsage } from "@/lib/admin-api";
import { LlmUsageTable } from "@/components/admin/llm-usage-table";

export default async function AdminLlmUsagePage(): Promise<React.ReactNode> {
  const { data } = await fetchLlmUsage();
  return <LlmUsageTable entries={data} />;
}
