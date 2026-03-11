"use client";

import { useState, useCallback } from "react";
import type { Thread } from "@/lib/api-types";
import type { PipelineStage } from "@/lib/crm-types";
import {
  PIPELINE_STAGES,
  STAGE_LABELS,
  threadsToLeadCards,
  groupByStage,
  filterLeadsByAssignee,
  filterLeadsByMinScore,
  getUniqueAssignees,
} from "@/lib/crm-types";
import { KanbanCard, StageBadge } from "./kanban-card";
import { KanbanFilters, type KanbanFilterValues } from "./kanban-filters";

export interface KanbanBoardProps {
  /** Threads representing leads in the pipeline. */
  threads: Thread[];
  /** Stages to display as columns (defaults to all PIPELINE_STAGES). */
  stages?: readonly PipelineStage[];
  /** Called when a lead card is clicked. */
  onCardClick?: (threadId: string) => void;
  /** Called when a card is dropped on a new stage (drag-drop transition). */
  onStageChange?: (threadId: string, newStage: PipelineStage) => void;
  /** Base path for constructing lead detail links. */
  basePath?: string;
  /** Whether the board is in loading state. */
  loading?: boolean;
}

/** Kanban pipeline board — columns = stages, cards = leads with metadata. */
export function KanbanBoard({
  threads,
  stages = PIPELINE_STAGES,
  onCardClick,
  onStageChange,
  basePath,
  loading = false,
}: KanbanBoardProps): React.ReactNode {
  const [filters, setFilters] = useState<KanbanFilterValues>({
    assignee: "all",
    minScore: 0,
    search: "",
  });
  const [dragOverStage, setDragOverStage] = useState<PipelineStage | null>(null);

  const allCards = threadsToLeadCards(threads);
  const assignees = getUniqueAssignees(allCards);

  // Apply filters
  let filtered = filterLeadsByAssignee(allCards, filters.assignee);
  filtered = filterLeadsByMinScore(filtered, filters.minScore);
  if (filters.search) {
    const q = filters.search.toLowerCase();
    filtered = filtered.filter(
      (c) =>
        c.thread.title.toLowerCase().includes(q) ||
        (c.lead.company?.toLowerCase().includes(q) ?? false),
    );
  }

  const grouped = groupByStage(filtered);

  const handleDragOver = useCallback((e: React.DragEvent, stage: PipelineStage) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragOverStage(stage);
  }, []);

  const handleDragLeave = useCallback(() => {
    setDragOverStage(null);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent, stage: PipelineStage) => {
      e.preventDefault();
      setDragOverStage(null);
      const threadId = e.dataTransfer.getData("text/plain");
      if (threadId && onStageChange) {
        onStageChange(threadId, stage);
      }
    },
    [onStageChange],
  );

  if (loading) {
    return (
      <div
        className="flex items-center justify-center py-12 text-sm text-muted-foreground"
        data-testid="kanban-loading"
      >
        Loading pipeline...
      </div>
    );
  }

  return (
    <div data-testid="kanban-board">
      {/* Filters */}
      <KanbanFilters values={filters} onChange={setFilters} assignees={assignees} />

      {/* Board */}
      <div className="mt-4 flex gap-3 overflow-x-auto pb-4" data-testid="kanban-columns">
        {stages.map((stage) => {
          const cards = grouped[stage] ?? [];
          return (
            <div
              key={stage}
              className={`flex w-72 min-w-[18rem] flex-col rounded-lg border bg-muted/30 ${
                dragOverStage === stage ? "border-primary bg-primary/5" : "border-border"
              }`}
              data-testid={`kanban-column-${stage}`}
              onDragOver={(e) => handleDragOver(e, stage)}
              onDragLeave={handleDragLeave}
              onDrop={(e) => handleDrop(e, stage)}
            >
              {/* Column header */}
              <div
                className="flex items-center justify-between px-3 py-2"
                data-testid={`kanban-header-${stage}`}
              >
                <span className="text-sm font-semibold text-foreground">{STAGE_LABELS[stage]}</span>
                <StageBadge stage={stage} count={cards.length} />
              </div>

              {/* Cards */}
              <div className="flex-1 space-y-2 px-2 pb-2" data-testid={`kanban-cards-${stage}`}>
                {cards.map((card) => (
                  <KanbanCard
                    key={card.thread.id}
                    card={card}
                    onClick={onCardClick}
                    href={basePath ? `${basePath}/${card.thread.slug}` : undefined}
                  />
                ))}
                {cards.length === 0 && (
                  <div
                    className="py-4 text-center text-xs text-muted-foreground"
                    data-testid={`kanban-empty-${stage}`}
                  >
                    No leads
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
