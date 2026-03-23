import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowBigUp, Calendar, User } from "lucide-react";

import { fetchGlobalThread, GLOBAL_SPACES } from "@/lib/global-api";
import type { ThreadWithAuthor } from "@/lib/api-types";
import { AuthorAvatar } from "@/components/forum/author-avatar";
import { relativeTime } from "@/components/forum/relative-time";
import { ForumReplies } from "@/components/forum/forum-replies";

interface ForumPageProps {
  params: Promise<{ slug: string[] }>;
}

/** Resolve a display name from the enriched thread author fields. */
function authorDisplayName(thread: ThreadWithAuthor): string {
  if (thread.author_name) return thread.author_name;
  if (thread.author_email) return thread.author_email;
  if (thread.author_id === "system-seed") return "DEFT Team";
  return thread.author_id.slice(0, 12);
}

/** Public forum thread detail page. */
export default async function ForumPage({ params }: ForumPageProps): Promise<ReactNode> {
  const { slug } = await params;
  const threadSlug = slug[0] ?? "";

  let thread: ThreadWithAuthor | null = null;
  try {
    // The API returns ThreadWithAuthor (author_name, author_email enriched).
    thread = (await fetchGlobalThread(GLOBAL_SPACES.FORUM, threadSlug)) as ThreadWithAuthor;
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

  const displayName = authorDisplayName(thread);

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

        <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
          <div className="flex items-center gap-2">
            <AuthorAvatar authorId={thread.author_id} authorName={displayName} size="sm" />
            <span className="flex items-center gap-1 font-medium text-foreground">
              <User className="h-3 w-3" />
              {displayName}
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
            className="mt-6 text-sm leading-relaxed text-foreground whitespace-pre-wrap"
          >
            {thread.body}
          </div>
        )}
      </div>

      {/* Replies section */}
      <ForumReplies threadSlug={threadSlug} isLocked={thread.is_locked} />
    </div>
  );
}
