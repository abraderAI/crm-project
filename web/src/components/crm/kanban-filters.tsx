"use client";

import { LayoutGrid, List } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  STAGE_LABELS,
  PIPELINE_STAGES,
  OPPORTUNITY_TYPES,
  OPPORTUNITY_TYPE_LABELS,
  LEAD_SOURCES,
  LEAD_SOURCE_LABELS,
} from "@/lib/crm-types";

export interface KanbanFilterValues {
  assignee: string;
  minScore: number;
  search: string;
  stage: string;
  opportunityType: string;
  leadSource: string;
}

export interface KanbanFiltersProps {
  values: KanbanFilterValues;
  onChange: (values: KanbanFilterValues) => void;
  assignees: string[];
  /** Current view mode. */
  viewMode?: "kanban" | "list";
  /** Called when view mode changes. */
  onViewModeChange?: (mode: "kanban" | "list") => void;
}

/** Filter controls for the Kanban pipeline board with extended filters. */
export function KanbanFilters({
  values,
  onChange,
  assignees,
  viewMode = "kanban",
  onViewModeChange,
}: KanbanFiltersProps): React.ReactNode {
  const update = (partial: Partial<KanbanFilterValues>): void => {
    onChange({ ...values, ...partial });
  };

  return (
    <div className="flex flex-wrap items-center gap-3" data-testid="kanban-filters">
      {/* Search */}
      <input
        type="search"
        value={values.search}
        onChange={(e) => update({ search: e.target.value })}
        placeholder="Search leads..."
        data-testid="kanban-search-input"
        className="h-8 w-48 rounded-md border border-border bg-background px-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
      />

      {/* Assignee filter */}
      <select
        value={values.assignee}
        onChange={(e) => update({ assignee: e.target.value })}
        data-testid="kanban-assignee-filter"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        <option value="all">All assignees</option>
        {assignees.map((a) => (
          <option key={a} value={a}>
            {a}
          </option>
        ))}
      </select>

      {/* Stage filter */}
      <select
        value={values.stage ?? "all"}
        onChange={(e) => update({ stage: e.target.value })}
        data-testid="kanban-stage-filter"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        <option value="all">All stages</option>
        {PIPELINE_STAGES.map((s) => (
          <option key={s} value={s}>
            {STAGE_LABELS[s]}
          </option>
        ))}
      </select>

      {/* Opportunity type filter */}
      <select
        value={values.opportunityType ?? "all"}
        onChange={(e) => update({ opportunityType: e.target.value })}
        data-testid="kanban-type-filter"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        <option value="all">All types</option>
        {OPPORTUNITY_TYPES.map((t) => (
          <option key={t} value={t}>
            {OPPORTUNITY_TYPE_LABELS[t]}
          </option>
        ))}
      </select>

      {/* Lead source filter */}
      <select
        value={values.leadSource ?? "all"}
        onChange={(e) => update({ leadSource: e.target.value })}
        data-testid="kanban-source-filter"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        <option value="all">All sources</option>
        {LEAD_SOURCES.map((s) => (
          <option key={s} value={s}>
            {LEAD_SOURCE_LABELS[s]}
          </option>
        ))}
      </select>

      {/* Min score filter */}
      <label className="flex items-center gap-1 text-xs text-muted-foreground">
        Min score:
        <input
          type="number"
          min={0}
          max={100}
          value={values.minScore}
          onChange={(e) => update({ minScore: Number(e.target.value) || 0 })}
          data-testid="kanban-score-filter"
          className="h-8 w-16 rounded-md border border-border bg-background px-2 text-sm"
        />
      </label>

      {/* View toggle */}
      {onViewModeChange && (
        <div className="ml-auto flex gap-1" data-testid="kanban-view-toggle">
          <button
            onClick={() => onViewModeChange("kanban")}
            className={cn(
              "rounded-md p-1.5 transition-colors",
              viewMode === "kanban"
                ? "bg-primary/10 text-primary"
                : "text-muted-foreground hover:bg-accent",
            )}
            aria-label="Kanban view"
            data-testid="kanban-view-btn"
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
          <button
            onClick={() => onViewModeChange("list")}
            className={cn(
              "rounded-md p-1.5 transition-colors",
              viewMode === "list"
                ? "bg-primary/10 text-primary"
                : "text-muted-foreground hover:bg-accent",
            )}
            aria-label="List view"
            data-testid="list-view-btn"
          >
            <List className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  );
}
