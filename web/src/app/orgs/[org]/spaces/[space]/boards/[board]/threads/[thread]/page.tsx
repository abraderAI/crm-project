import Link from "next/link";
import { ChevronRight } from "lucide-react";

import { fetchOrg, fetchSpace, fetchBoard, fetchThread, fetchMessages } from "@/lib/user-api";
import { ThreadDetailView } from "@/components/thread/thread-detail-view";

interface ThreadPageProps {
  params: Promise<{ org: string; space: string; board: string; thread: string }>;
}

export default async function ThreadPage({ params }: ThreadPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug, board: boardSlug, thread: threadSlug } = await params;
  const [org, space, board, thread, { data: messages }] = await Promise.all([
    fetchOrg(orgSlug),
    fetchSpace(orgSlug, spaceSlug),
    fetchBoard(orgSlug, spaceSlug, boardSlug),
    fetchThread(orgSlug, spaceSlug, boardSlug, threadSlug),
    fetchMessages(orgSlug, spaceSlug, boardSlug, threadSlug),
  ]);

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        <Link href="/" className="hover:text-foreground">
          Home
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <Link href={`/orgs/${orgSlug}`} className="hover:text-foreground">
          {org.name}
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <Link href={`/orgs/${orgSlug}/spaces/${spaceSlug}`} className="hover:text-foreground">
          {space.name}
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <Link
          href={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}`}
          className="hover:text-foreground"
        >
          {board.name}
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <span className="font-medium text-foreground">{thread.title}</span>
      </nav>

      <ThreadDetailView
        thread={thread}
        messages={messages}
        orgSlug={orgSlug}
        spaceSlug={spaceSlug}
        boardSlug={boardSlug}
        threadSlug={threadSlug}
      />
    </div>
  );
}
