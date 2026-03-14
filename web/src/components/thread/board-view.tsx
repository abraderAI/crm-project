"use client";

import { useMemo, useState } from "react";
import type { Thread } from "@/lib/api-types";
import { ThreadFilters, type ThreadFilterValues } from "./thread-filters";
import { ThreadList } from "./thread-list";

export interface BoardViewProps {
  /** All threads for this board (fetched server-side). */
  threads: Thread[];
  /** Base path for constructing thread links. */
  basePath: string;
}

const DEFAULT_FILTERS: ThreadFilterValues = {
  status: "all",
  priority: "all",
  sortBy: "newest",
  search: "",
};

/** Filter threads by status, priority, and search query. */
function filterThreads(threads: Thread[], filters: ThreadFilterValues): Thread[] {
  return threads.filter((t) => {
    if (filters.status !== "all" && t.status !== filters.status) return false;
    if (filters.priority !== "all" && t.priority !== filters.priority) return false;
    if (filters.search && !t.title.toLowerCase().includes(filters.search.toLowerCase())) {
      return false;
    }
    return true;
  });
}

/** Sort threads by the selected sort option. */
function sortThreads(threads: Thread[], sortBy: ThreadFilterValues["sortBy"]): Thread[] {
  const sorted = [...threads];
  switch (sortBy) {
    case "newest":
      return sorted.sort((a, b) => b.created_at.localeCompare(a.created_at));
    case "oldest":
      return sorted.sort((a, b) => a.created_at.localeCompare(b.created_at));
    case "most_votes":
      return sorted.sort((a, b) => b.vote_score - a.vote_score);
    case "recently_updated":
      return sorted.sort((a, b) => b.updated_at.localeCompare(a.updated_at));
    default:
      return sorted;
  }
}

/** Client wrapper rendering ThreadFilters + ThreadList with client-side filter/sort. */
export function BoardView({ threads, basePath }: BoardViewProps): React.ReactNode {
  const [filters, setFilters] = useState<ThreadFilterValues>(DEFAULT_FILTERS);

  const visibleThreads = useMemo(
    () => sortThreads(filterThreads(threads, filters), filters.sortBy),
    [threads, filters],
  );

  return (
    <div className="space-y-4" data-testid="board-view">
      <ThreadFilters values={filters} onChange={setFilters} />
      <ThreadList threads={visibleThreads} basePath={basePath} />
    </div>
  );
}
