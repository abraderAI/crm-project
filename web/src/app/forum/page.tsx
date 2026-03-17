import Link from "next/link";
import { MessageSquare, Plus } from "lucide-react";

import { fetchGlobalThreads, GLOBAL_SPACES } from "@/lib/global-api";
import type { Thread } from "@/lib/api-types";

/**
 * Forum index page — lists recent threads from global-forum.
 * Public route (no auth required). Individual threads link to /forum/[slug]
 * which is served by the (public) layout for anonymous access.
 */
export default async function ForumIndexPage(): Promise<React.ReactNode> {
  let threads: Thread[] = [];
  try {
    const result = await fetchGlobalThreads(GLOBAL_SPACES.FORUM, { limit: 20 });
    threads = result.data;
  } catch {
    // Render empty state on API failure — forum is non-critical.
  }

  return (
    <div data-testid="forum-index-page" className="mx-auto max-w-3xl space-y-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-foreground">Community Forum</h1>
        <Link
          href="/sign-in"
          data-testid="forum-sign-in-link"
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Post a Thread
        </Link>
      </div>

      {threads.length === 0 ? (
        <p data-testid="forum-empty" className="text-sm text-muted-foreground">
          No forum discussions yet. Be the first to start one!
        </p>
      ) : (
        <ul
          data-testid="forum-thread-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {threads.map((thread) => (
            <li key={thread.id} data-testid={`forum-thread-${thread.id}`}>
              <Link
                href={`/forum/${thread.slug}`}
                className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50"
              >
                <MessageSquare className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                <div className="min-w-0 flex-1">
                  <span className="text-sm font-medium text-foreground">{thread.title}</span>
                  {thread.vote_score > 0 && (
                    <span className="ml-2 text-xs text-muted-foreground">
                      ▲ {thread.vote_score}
                    </span>
                  )}
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
