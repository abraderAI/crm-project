"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { BookOpen } from "lucide-react";
import type { Thread } from "@/lib/api-types";
import { fetchGlobalThreads, GLOBAL_SPACES } from "@/lib/global-api";

/** Maximum number of doc threads to display. */
const MAX_ITEMS = 5;

/** Displays recent documentation threads from global-docs. Renders without auth. */
export function DocsHighlightsWidget(): ReactNode {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchGlobalThreads(GLOBAL_SPACES.DOCS, { limit: MAX_ITEMS });
      if (!mountedRef.current) return;
      setThreads(result.data);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load documentation.");
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
      <div data-testid="docs-highlights-loading" className="animate-pulse space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-4 rounded bg-muted" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="docs-highlights-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (threads.length === 0) {
    return (
      <p data-testid="docs-highlights-empty" className="text-sm text-muted-foreground">
        No documentation available yet.
      </p>
    );
  }

  return (
    <ul data-testid="docs-highlights-list" className="space-y-2">
      {threads.map((thread) => (
        <li key={thread.id}>
          <Link
            href={`/docs/${thread.slug}`}
            className="flex items-start gap-2 rounded p-1 text-sm hover:bg-accent/50"
          >
            <BookOpen className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
            <span className="text-foreground">{thread.title}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
