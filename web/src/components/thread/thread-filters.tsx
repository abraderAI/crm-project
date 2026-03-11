"use client";

export type SortOption = "newest" | "oldest" | "most_votes" | "recently_updated";

export interface ThreadFilterValues {
  status: string;
  priority: string;
  sortBy: SortOption;
  search: string;
}

export interface ThreadFiltersProps {
  values: ThreadFilterValues;
  onChange: (values: ThreadFilterValues) => void;
  statusOptions?: string[];
  priorityOptions?: string[];
}

const DEFAULT_STATUSES = ["all", "open", "in_progress", "closed", "resolved"];
const DEFAULT_PRIORITIES = ["all", "low", "medium", "high", "critical"];
const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: "newest", label: "Newest" },
  { value: "oldest", label: "Oldest" },
  { value: "most_votes", label: "Most votes" },
  { value: "recently_updated", label: "Recently updated" },
];

/** Filter and sort controls for thread lists. */
export function ThreadFilters({
  values,
  onChange,
  statusOptions = DEFAULT_STATUSES,
  priorityOptions = DEFAULT_PRIORITIES,
}: ThreadFiltersProps): React.ReactNode {
  const update = (partial: Partial<ThreadFilterValues>): void => {
    onChange({ ...values, ...partial });
  };

  return (
    <div className="flex flex-wrap items-center gap-3" data-testid="thread-filters">
      {/* Search */}
      <input
        type="search"
        value={values.search}
        onChange={(e) => update({ search: e.target.value })}
        placeholder="Search threads..."
        data-testid="thread-search-input"
        className="h-8 w-48 rounded-md border border-border bg-background px-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
      />

      {/* Status */}
      <select
        value={values.status}
        onChange={(e) => update({ status: e.target.value })}
        data-testid="thread-status-filter"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        {statusOptions.map((s) => (
          <option key={s} value={s}>
            {s === "all" ? "All statuses" : s.replace("_", " ")}
          </option>
        ))}
      </select>

      {/* Priority */}
      <select
        value={values.priority}
        onChange={(e) => update({ priority: e.target.value })}
        data-testid="thread-priority-filter"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        {priorityOptions.map((p) => (
          <option key={p} value={p}>
            {p === "all" ? "All priorities" : p}
          </option>
        ))}
      </select>

      {/* Sort */}
      <select
        value={values.sortBy}
        onChange={(e) => update({ sortBy: e.target.value as SortOption })}
        data-testid="thread-sort-select"
        className="h-8 rounded-md border border-border bg-background px-2 text-sm"
      >
        {SORT_OPTIONS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  );
}
