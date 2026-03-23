import Link from "next/link";
import { ArrowBigUp, MessageSquare, Pin } from "lucide-react";
import type { ReactNode } from "react";

import type { Thread } from "@/lib/api-types";
import { AuthorAvatar } from "./author-avatar";
import { relativeTime } from "./relative-time";
import { stripHtml } from "./strip-html";

interface ThreadCardProps {
  thread: Thread;
}

/** A single thread card in the forum list. */
export function ThreadCard({ thread }: ThreadCardProps): ReactNode {
  return (
    <Link
      href={`/forum/${thread.slug}`}
      data-testid={`forum-thread-card-${thread.id}`}
      className="group flex gap-4 rounded-xl border border-border bg-background p-4 shadow-sm transition-all hover:border-primary/30 hover:shadow-md"
    >
      {/* Vote column */}
      <div className="flex flex-col items-center gap-0.5 pt-0.5">
        <ArrowBigUp className="h-5 w-5 text-muted-foreground transition-colors group-hover:text-primary" />
        <span className="text-sm font-semibold text-foreground">{thread.vote_score}</span>
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          {thread.is_pinned && (
            <Pin className="h-3.5 w-3.5 shrink-0 text-amber-500" data-testid="pin-icon" />
          )}
          <h3 className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors line-clamp-1">
            {thread.title}
          </h3>
        </div>

        {thread.body && (
          <p className="mt-1 text-xs text-muted-foreground line-clamp-2">
            {stripHtml(thread.body)}
          </p>
        )}

        <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
          <AuthorAvatar authorId={thread.author_id} size="sm" />
          <span>{relativeTime(thread.created_at)}</span>
          {(thread.messages?.length ?? 0) > 0 && (
            <span className="flex items-center gap-1">
              <MessageSquare className="h-3 w-3" />
              {thread.messages?.length}
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}
