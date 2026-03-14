"use client";

import type { Thread } from "@/lib/api-types";
import { threadsToLeadCards, computePipelineStats } from "@/lib/crm-types";
import { KanbanBoard } from "@/components/crm/kanban-board";
import { PipelineDashboard } from "@/components/crm/pipeline-stats";

export interface CrmPipelineViewProps {
  threads: Thread[];
  /** Per-thread href map (thread id → lead detail path). */
  threadHrefs?: Record<string, string>;
}

/** Client wrapper rendering KanbanBoard + PipelineStats from server-fetched threads. */
export function CrmPipelineView({
  threads,
  threadHrefs,
}: CrmPipelineViewProps): React.ReactNode {
  const cards = threadsToLeadCards(threads);
  const stats = computePipelineStats(cards);

  return (
    <div className="space-y-6">
      <PipelineDashboard stats={stats} />
      <KanbanBoard threads={threads} threadHrefs={threadHrefs} />
    </div>
  );
}
