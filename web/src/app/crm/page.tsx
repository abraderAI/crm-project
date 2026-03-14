import { fetchOrgs, fetchSpaces, fetchBoards, fetchThreads } from "@/lib/user-api";
import type { Thread } from "@/lib/api-types";
import { CrmPipelineView } from "./crm-pipeline-view";

/**
 * CRM Pipeline page — aggregates threads from all CRM-type spaces across
 * the user's organizations and renders a Kanban board with pipeline stats.
 */
export default async function CrmPage(): Promise<React.ReactNode> {
  const { data: orgs } = await fetchOrgs();

  // Gather all threads from CRM-type spaces across all orgs.
  const allThreads: Thread[] = [];
  const threadHrefs: Record<string, string> = {};

  for (const org of orgs) {
    const { data: spaces } = await fetchSpaces(org.slug, { type: "crm" });
    for (const space of spaces) {
      const { data: boards } = await fetchBoards(org.slug, space.slug);
      for (const board of boards) {
        const { data: threads } = await fetchThreads(org.slug, space.slug, board.slug);
        for (const thread of threads) {
          threadHrefs[thread.id] =
            `/crm/leads/${org.slug}/${space.slug}/${board.slug}/${thread.slug}`;
        }
        allThreads.push(...threads);
      }
    }
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6 p-6">
      <h1 className="text-xl font-bold text-foreground">CRM Pipeline</h1>
      <CrmPipelineView threads={allThreads} threadHrefs={threadHrefs} />
    </div>
  );
}
