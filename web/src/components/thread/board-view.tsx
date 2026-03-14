"use client";

import { useMemo, useState } from "react";
import type { Thread, ThreadSortOption } from "@/lib/api-types";
import { ThreadFilters, type ThreadFilterValues, type SortOption } from "./thread-filters";
import { ThreadList } from "./thread-list";
import { VoteSort } from "@/components/community/vote-sort";

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

/** Map SortOption to ThreadSortOption for VoteSort display. */
function toVoteSortOption(sortBy: SortOption): ThreadSortOption {
  switch (sortBy) {
    case "most_votes":
      return "votes";
    case "oldest":
      return "oldest";
    default:
      return "newest";
  }
}

/** Map ThreadSortOption to SortOption for filter state. */
function fromVoteSortOption(option: ThreadSortOption): SortOption {
  switch (option) {
    case "votes":
      return "most_votes";
    case "oldest":
      return "oldest";
    default:
      return "newest";
  }
}

/** Client wrapper rendering ThreadFilters + VoteSort + ThreadList with client-side filter/sort. */
export function BoardView({ threads, basePath }: BoardViewProps): React.ReactNode {
  const [filters, setFilters] = useState<ThreadFilterValues>(DEFAULT_FILTERS);

  const visibleThreads = useMemo(
    () => sortThreads(filterThreads(threads, filters), filters.sortBy),
    [threads, filters],
  );

  const handleVoteSortChange = (option: ThreadSortOption): void => {
    setFilters((prev) => ({ ...prev, sortBy: fromVoteSortOption(option) }));
  };

  return (
    <div className="space-y-4" data-testid="board-view">
      <ThreadFilters values={filters} onChange={setFilters} />
      <VoteSort value={toVoteSortOption(filters.sortBy)} onChange={handleVoteSortChange} />
      <ThreadList threads={visibleThreads} basePath={basePath} />
    </div>
  );
}
