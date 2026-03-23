import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowBigUp, Calendar, User } from "lucide-react";

import { fetchGlobalThread, GLOBAL_SPACES } from "@/lib/global-api";
import type { Thread } from "@/lib/api-types";
import { AuthorAvatar } from "@/components/forum/author-avatar";
import { relativeTime } from "@/components/forum/relative-time";

interface ForumPageProps {
  params: Promise<{ slug: string[] }>;
}

/** Public forum thread detail page. */
export default async function ForumPage({ params }: ForumPageProps): Promise<ReactNode> {
  const { slug } = await params;
  const threadSlug = slug[0] ?? "";

  let thread: Thread | null = null;
  try {
    thread = await fetchGlobalThread(GLOBAL_SPACES.FORUM, threadSlug);
  } catch {
    // Fall through to not-found state.
  }

  if (!thread) {
    return (
      <div data-testid="forum-thread-not-found" className="py-16 text-center">
        <p className="text-sm text-muted-foreground">Thread not found.</p>
        <Link href="/forum" className="mt-4 inline-block text-sm text-primary hover:underline">
          ← Back to forum
        </Link>
      </div>
    );
  }

  return (
    <div data-testid="forum-thread-detail" className="mx-auto max-w-3xl space-y-6 p-6">
      {/* Back link */}
      <Link
        href="/forum"
        data-testid="back-to-forum"
        className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to forum
      </Link>

      {/* Thread header card */}
      <div className="rounded-2xl border border-border bg-background p-6 shadow-sm">
        <h1 className="text-xl font-bold text-foreground">{thread.title}</h1>

        <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
          <div className="flex items-center gap-2">
            <AuthorAvatar authorId={thread.author_id} size="sm" />
            <span className="flex items-center gap-1">
              <User className="h-3 w-3" />
              {thread.author_id === "system-seed" ? "DEFT Team" : thread.author_id.slice(0, 8)}
            </span>
          </div>
          <span className="flex items-center gap-1">
            <Calendar className="h-3 w-3" />
            {relativeTime(thread.created_at)}
          </span>
          <span className="flex items-center gap-1">
            <ArrowBigUp className="h-4 w-4" />
            {thread.vote_score} votes
          </span>
        </div>

        {/* Thread body */}
        {thread.body && (
          <div
            data-testid="forum-thread-body"
            className="mt-6 prose prose-sm max-w-none text-foreground whitespace-pre-wrap"
          >
            {thread.body}
          </div>
        )}
      </div>

      {/* Replies placeholder */}
      <div
        data-testid="forum-replies-placeholder"
        className="rounded-xl border border-dashed border-border p-8 text-center"
      >
        <p className="text-sm text-muted-foreground">
          Replies coming soon. Sign in to be notified when this feature launches.
        </p>
      </div>
    </div>
  );
}
