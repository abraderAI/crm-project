"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { MessageCircle } from "lucide-react";
import type { Thread } from "@/lib/api-types";
import { fetchUserForumActivity } from "@/lib/global-api";

/** Maximum number of forum activity items to display. */
const MAX_ITEMS = 5;

interface MyForumActivityWidgetProps {
  /** Auth token for fetching user's forum activity. */
  token: string;
}

/** Displays the user's recent posts and replies in global-forum. */
export function MyForumActivityWidget({ token }: MyForumActivityWidgetProps): ReactNode {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fetchUserForumActivity(token, { limit: MAX_ITEMS });
      if (!mountedRef.current) return;
      setThreads(result.data);
    } catch {
      if (!mountedRef.current) return;
      setError("Failed to load your forum activity.");
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [token]);

  useEffect(() => {
    mountedRef.current = true;
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, [load]);

  if (isLoading) {
    return (
      <div data-testid="my-forum-activity-loading" className="animate-pulse space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-4 rounded bg-muted" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p data-testid="my-forum-activity-error" className="text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (threads.length === 0) {
    return (
      <p data-testid="my-forum-activity-empty" className="text-sm text-muted-foreground">
        You haven&apos;t posted in the forum yet.{" "}
        <Link href="/forum" className="font-medium text-primary hover:underline">
          Join the conversation
        </Link>
      </p>
    );
  }

  return (
    <ul data-testid="my-forum-activity-list" className="space-y-2">
      {threads.map((thread) => (
        <li key={thread.id}>
          <Link
            href={`/forum/${thread.slug}`}
            className="flex items-start gap-2 rounded p-1 text-sm hover:bg-accent/50"
          >
            <MessageCircle className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
            <span className="text-foreground">{thread.title}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
