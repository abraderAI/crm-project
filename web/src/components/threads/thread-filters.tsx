"use client";

import { Filter, SortAsc, SortDesc } from "lucide-react";

export type SortField = "created_at" | "updated_at" | "vote_score" | "title";
export type SortDirection = "asc" | "desc";

export interface ThreadFilterValues {
  status?: string;
  priority?: string;
  assignedTo?: string;
  sortBy: SortField;
  sortDir: SortDirection;
}

export interface ThreadFiltersProps {
  /** Current filter values. */
  values: ThreadFilterValues;
  /** Called when any filter or sort value changes. */
  onChange: (values: ThreadFilterValues) => void;
  /** Available status options. */
  statusOptions?: string[];
  /** Available priority options. */
  priorityOptions?: string[];
}

const SORT_FIELDS: { value: SortField; label: string }[] = [
  { value: "created_at", label: "Created" },
  { value: "updated_at", label: "Updated" },
  { value: "vote_score", label: "Votes" },
  { value: "title", label: "Title" },
];

/** Filter bar for thread lists with status, priority, assignee, and sort controls. */
export function ThreadFilters({
  values,
  onChange,
  statusOptions = ["open", "in_progress", "resolved", "closed"],
  priorityOptions = ["low", "medium", "high", "critical"],
}: ThreadFiltersProps): React.ReactNode {
  const update = (partial: Partial<ThreadFilterValues>): void => {
    onChange({ ...values, ...partial });
  };

  const toggleDirection = (): void => {
    update({ sortDir: values.sortDir === "asc" ? "desc" : "asc" });
  };

  return (
    <div
      data-testid="thread-filters"
      className="flex flex-wrap items-center gap-3 rounded-lg border border-border bg-background p-3"
    >
      <Filter className="h-4 w-4 text-muted-foreground" />

      {/* Status filter */}
      <select
        value={values.status ?? ""}
        onChange={(e) => update({ status: e.target.value || undefined })}
        data-testid="filter-status"
        className="rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground"
        aria-label="Filter by status"
      >
        <option value="">All statuses</option>
        {statusOptions.map((s) => (
          <option key={s} value={s}>
            {s.replace("_", " ")}
          </option>
        ))}
      </select>

      {/* Priority filter */}
      <select
        value={values.priority ?? ""}
        onChange={(e) => update({ priority: e.target.value || undefined })}
        data-testid="filter-priority"
        className="rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground"
        aria-label="Filter by priority"
      >
        <option value="">All priorities</option>
        {priorityOptions.map((p) => (
          <option key={p} value={p}>
            {p}
          </option>
        ))}
      </select>

      {/* Assignee filter */}
      <input
        type="text"
        placeholder="Assigned to..."
        value={values.assignedTo ?? ""}
        onChange={(e) => update({ assignedTo: e.target.value || undefined })}
        data-testid="filter-assigned"
        className="rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground placeholder:text-muted-foreground"
        aria-label="Filter by assignee"
      />

      {/* Spacer */}
      <div className="flex-1" />

      {/* Sort controls */}
      <div className="flex items-center gap-1">
        <select
          value={values.sortBy}
          onChange={(e) => update({ sortBy: e.target.value as SortField })}
          data-testid="sort-field"
          className="rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground"
          aria-label="Sort by"
        >
          {SORT_FIELDS.map((f) => (
            <option key={f.value} value={f.value}>
              {f.label}
            </option>
          ))}
        </select>
        <button
          onClick={toggleDirection}
          aria-label={values.sortDir === "asc" ? "Sort ascending" : "Sort descending"}
          data-testid="sort-direction"
          className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          {values.sortDir === "asc" ? (
            <SortAsc className="h-4 w-4" />
          ) : (
            <SortDesc className="h-4 w-4" />
          )}
        </button>
      </div>
    </div>
  );
}
