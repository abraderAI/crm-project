"use client";

import { MessageSquare, Pin, Lock, ThumbsUp } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Thread } from "@/lib/api-types";

export interface ThreadListProps {
  threads: Thread[];
  /** Base path for constructing thread links. */
  basePath: string;
  hasMore?: boolean;
  onLoadMore?: () => void;
  loading?: boolean;
  emptyMessage?: string;
}

/** Format a date string for display. */
export function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  } catch {
    return dateStr;
  }
}

/** Thread list with pinned, locked, vote score indicators and load-more pagination. */
export function ThreadList({
  threads,
  basePath,
  hasMore = false,
  onLoadMore,
  loading = false,
  emptyMessage = "No threads yet.",
}: ThreadListProps): React.ReactNode {
  if (threads.length === 0 && !loading) {
    return (
      <div
        className="py-8 text-center text-sm text-muted-foreground"
        data-testid="thread-list-empty"
      >
        {emptyMessage}
      </div>
    );
  }

  return (
    <div data-testid="thread-list">
      <div className="divide-y divide-border rounded-lg border border-border">
        {threads.map((thread) => (
          <a
            key={thread.id}
            href={`${basePath}/${thread.slug}`}
            data-testid={`thread-item-${thread.id}`}
            className={cn(
              "flex items-start gap-3 px-4 py-3 transition-colors hover:bg-accent/50",
              thread.is_pinned && "bg-accent/30",
            )}
          >
            {/* Vote score */}
            <div
              className="flex flex-col items-center pt-0.5 text-muted-foreground"
              data-testid={`thread-votes-${thread.id}`}
            >
              <ThumbsUp className="h-3.5 w-3.5" />
              <span className="text-xs font-medium">{thread.vote_score}</span>
            </div>

            {/* Content */}
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h3 className="truncate text-sm font-medium text-foreground">{thread.title}</h3>
                {thread.is_pinned && (
                  <Pin
                    className="h-3.5 w-3.5 shrink-0 text-primary"
                    data-testid={`thread-pin-${thread.id}`}
                  />
                )}
                {thread.is_locked && (
                  <Lock
                    className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
                    data-testid={`thread-lock-${thread.id}`}
                  />
                )}
              </div>
              <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                {thread.status && (
                  <span data-testid={`thread-status-${thread.id}`}>{thread.status}</span>
                )}
                {thread.priority && (
                  <span data-testid={`thread-priority-${thread.id}`}>{thread.priority}</span>
                )}
                <span>{formatDate(thread.created_at)}</span>
                {thread.messages && (
                  <span className="flex items-center gap-1">
                    <MessageSquare className="h-3 w-3" />
                    {thread.messages.length}
                  </span>
                )}
              </div>
            </div>
          </a>
        ))}
      </div>

      {hasMore && (
        <div className="mt-4 flex justify-center">
          <button
            onClick={onLoadMore}
            disabled={loading}
            data-testid="thread-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
          >
            {loading ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
