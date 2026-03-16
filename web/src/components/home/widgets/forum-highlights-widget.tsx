"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { MessageSquare } from "lucide-react";
import type { Thread } from "@/lib/api-types";
import { fetchGlobalThreads, GLOBAL_SPACES } from "@/lib/global-api";

/** Maximum number of forum threads to display. */
const MAX_ITEMS = 5;

/** Displays recent forum threads from global-forum. Renders without auth. */
export function ForumHighlightsWidget(): ReactNode {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchGlobalThreads(GLOBAL_SPACES.FORUM, { limit: MAX_ITEMS });
      if (!mountedRef.current) return;
      setThreads(result.data);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load forum threads.");
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, [load]);

  if (isLoading) {
    return (
      <div data-testid="forum-highlights-loading" className="animate-pulse space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-4 rounded bg-muted" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="forum-highlights-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (threads.length === 0) {
    return (
      <p data-testid="forum-highlights-empty" className="text-sm text-muted-foreground">
        No forum discussions yet. Be the first to start one!
      </p>
    );
  }

  return (
    <ul data-testid="forum-highlights-list" className="space-y-2">
      {threads.map((thread) => (
        <li key={thread.id}>
          <Link
            href={`/forum/${thread.slug}`}
            className="flex items-start gap-2 rounded p-1 text-sm hover:bg-accent/50"
          >
            <MessageSquare className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
            <div className="min-w-0 flex-1">
              <span className="text-foreground">{thread.title}</span>
              {thread.vote_score > 0 && (
                <span className="ml-2 text-xs text-muted-foreground">▲ {thread.vote_score}</span>
              )}
            </div>
          </Link>
        </li>
      ))}
    </ul>
  );
}
