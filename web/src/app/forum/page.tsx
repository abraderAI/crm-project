import { auth } from "@clerk/nextjs/server";
import { MessageSquare } from "lucide-react";

import { fetchGlobalThreads, GLOBAL_SPACES } from "@/lib/global-api";
import type { Thread } from "@/lib/api-types";
import { ForumHeader } from "@/components/forum/forum-header";
import { ThreadCard } from "@/components/forum/thread-card";

/**
 * Forum index page — lists recent threads from global-forum.
 * Public route (no auth required). Individual threads link to /forum/[slug]
 * which is served by the (public) layout for anonymous access.
 */
export default async function ForumIndexPage(): Promise<React.ReactNode> {
  const { userId } = await auth();
  const isAuthenticated = !!userId;

  let threads: Thread[] = [];
  try {
    const result = await fetchGlobalThreads(GLOBAL_SPACES.FORUM, { limit: 50 });
    threads = result.data;
  } catch {
    // Render empty state on API failure — forum is non-critical.
  }

  // Separate pinned threads.
  const pinned = threads.filter((t) => t.is_pinned);
  const regular = threads.filter((t) => !t.is_pinned);

  return (
    <div data-testid="forum-index-page" className="mx-auto max-w-3xl space-y-6 p-6">
      <ForumHeader threadCount={threads.length} isAuthenticated={isAuthenticated} />

      {/* Pinned threads */}
      {pinned.length > 0 && (
        <section data-testid="forum-pinned-section">
          <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            📌 Pinned
          </h2>
          <div className="space-y-3">
            {pinned.map((thread) => (
              <ThreadCard key={thread.id} thread={thread} />
            ))}
          </div>
        </section>
      )}

      {/* All threads */}
      {regular.length > 0 ? (
        <section data-testid="forum-thread-list">
          <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Recent Threads
          </h2>
          <div className="space-y-3">
            {regular.map((thread) => (
              <ThreadCard key={thread.id} thread={thread} />
            ))}
          </div>
        </section>
      ) : (
        threads.length === 0 && (
          <div
            data-testid="forum-empty"
            className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border py-16 text-center"
          >
            <MessageSquare className="h-10 w-10 text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground">
              No discussions yet. Be the first to start one!
            </p>
          </div>
        )
      )}
    </div>
  );
}
