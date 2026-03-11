"use client";

import { useState } from "react";
import { MessageSquare, Pin, Lock, Plus } from "lucide-react";
import type { Thread } from "@/lib/api-types";
import { cn } from "@/lib/utils";
import { ThreadFilters, type ThreadFilterValues } from "./thread-filters";

export interface ThreadListProps {
  /** Threads to display. */
  threads: Thread[];
  /** Whether data is currently loading. */
  loading?: boolean;
  /** Whether there are more pages to load. */
  hasMore?: boolean;
  /** Called when the user requests the next page. */
  onLoadMore?: () => void;
  /** Called when a thread is selected. */
  onSelect?: (threadId: string) => void;
  /** Called when the user clicks create. */
  onCreate?: () => void;
  /** Called when filter/sort values change. */
  onFilterChange?: (values: ThreadFilterValues) => void;
  /** Current filter values (controlled). */
  filterValues?: ThreadFilterValues;
}

/** Format a date string for display. */
function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
    });
  } catch {
    return iso;
  }
}

/** Filterable, sortable thread list with cursor pagination controls. */
export function ThreadList({
  threads,
  loading = false,
  hasMore = false,
  onLoadMore,
  onSelect,
  onCreate,
  onFilterChange,
  filterValues: controlledFilters,
}: ThreadListProps): React.ReactNode {
  const [internalFilters, setInternalFilters] = useState<ThreadFilterValues>({
    sortBy: "created_at",
    sortDir: "desc",
  });

  const filters = controlledFilters ?? internalFilters;

  const handleFilterChange = (values: ThreadFilterValues): void => {
    if (onFilterChange) {
      onFilterChange(values);
    } else {
      setInternalFilters(values);
    }
  };

  return (
    <div data-testid="thread-list" className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">Threads</h2>
        {onCreate && (
          <button
            onClick={onCreate}
            data-testid="thread-create-btn"
            className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            New Thread
          </button>
        )}
      </div>

      {/* Filters */}
      <ThreadFilters values={filters} onChange={handleFilterChange} />

      {/* Loading */}
      {loading && (
        <div
          className="py-8 text-center text-sm text-muted-foreground"
          data-testid="threads-loading"
        >
          Loading threads...
        </div>
      )}

      {/* Empty state */}
      {!loading && threads.length === 0 && (
        <p className="py-8 text-center text-sm text-muted-foreground" data-testid="threads-empty">
          No threads found.
        </p>
      )}

      {/* Thread items */}
      {!loading && threads.length > 0 && (
        <div className="flex flex-col gap-2" data-testid="thread-items">
          {threads.map((thread) => (
            <div
              key={thread.id}
              data-testid={`thread-item-${thread.id}`}
              className={cn(
                "flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-accent",
              )}
              onClick={() => onSelect?.(thread.id)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onSelect?.(thread.id);
                }
              }}
            >
              <MessageSquare className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="truncate text-sm font-medium text-foreground">
                    {thread.title}
                  </span>
                  {thread.is_pinned && (
                    <Pin
                      className="h-3 w-3 shrink-0 text-primary"
                      data-testid={`pin-${thread.id}`}
                    />
                  )}
                  {thread.is_locked && (
                    <Lock
                      className="h-3 w-3 shrink-0 text-muted-foreground"
                      data-testid={`lock-${thread.id}`}
                    />
                  )}
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  {thread.status && (
                    <span
                      className="rounded-full bg-muted px-2 py-0.5"
                      data-testid={`status-${thread.id}`}
                    >
                      {thread.status}
                    </span>
                  )}
                  {thread.priority && (
                    <span data-testid={`priority-${thread.id}`}>{thread.priority}</span>
                  )}
                  <span>↑ {thread.vote_score}</span>
                  <span>{formatDate(thread.created_at)}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Load more */}
      {hasMore && !loading && (
        <div className="flex justify-center">
          <button
            onClick={onLoadMore}
            data-testid="threads-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}
