"use client";

export interface KanbanFilterValues {
  assignee: string;
  minScore: number;
  search: string;
}

export interface KanbanFiltersProps {
  values: KanbanFilterValues;
  onChange: (values: KanbanFilterValues) => void;
  assignees: string[];
}

/** Filter controls for the Kanban pipeline board. */
export function KanbanFilters({
  values,
  onChange,
  assignees,
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
    </div>
  );
}
