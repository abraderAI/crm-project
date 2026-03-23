import Link from "next/link";
import { MessageSquare, Plus } from "lucide-react";
import type { ReactNode } from "react";

interface ForumHeaderProps {
  threadCount: number;
  isAuthenticated: boolean;
}

/** Forum page header with gradient banner, title, and new-thread CTA. */
export function ForumHeader({ threadCount, isAuthenticated }: ForumHeaderProps): ReactNode {
  const newThreadHref = isAuthenticated ? "/forum/new" : "/sign-in?redirect_url=%2Fforum%2Fnew";

  return (
    <div
      data-testid="forum-header"
      className="rounded-2xl bg-gradient-to-br from-primary/10 via-primary/5 to-transparent border border-primary/10 p-6"
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <MessageSquare className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-bold text-foreground">DEFT General Discussion</h1>
          </div>
          <p className="mt-2 text-sm text-muted-foreground max-w-xl">
            Ask questions, share tips, and discuss everything about the Deft framework. From getting
            started to advanced strategies — all are welcome.
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {threadCount} {threadCount === 1 ? "thread" : "threads"}
          </p>
        </div>

        <Link
          href={newThreadHref}
          data-testid="forum-new-thread-btn"
          className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          New Thread
        </Link>
      </div>
    </div>
  );
}
