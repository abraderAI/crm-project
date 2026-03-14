import Link from "next/link";
import { ChevronRight, Plus, Settings } from "lucide-react";

import { fetchOrg, fetchSpace, fetchBoard, fetchThreads } from "@/lib/user-api";
import { BoardView } from "@/components/thread/board-view";

interface BoardPageProps {
  params: Promise<{ org: string; space: string; board: string }>;
}

export default async function BoardPage({ params }: BoardPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug, board: boardSlug } = await params;
  const [org, space, board, { data: threads }] = await Promise.all([
    fetchOrg(orgSlug),
    fetchSpace(orgSlug, spaceSlug),
    fetchBoard(orgSlug, spaceSlug, boardSlug),
    fetchThreads(orgSlug, spaceSlug, boardSlug),
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
        <span className="font-medium text-foreground">{board.name}</span>
        <Link
          href={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/settings`}
          className="ml-auto hover:text-foreground"
          title="Settings"
        >
          <Settings className="h-4 w-4" />
        </Link>
      </nav>

      {board.description && <p className="text-sm text-muted-foreground">{board.description}</p>}

      {!board.is_locked && (
        <div className="flex justify-end">
          <Link
            href={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/create`}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            New Thread
          </Link>
        </div>
      )}

      <BoardView
        threads={threads}
        basePath={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads`}
      />
    </div>
  );
}
