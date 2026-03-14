import Link from "next/link";
import { ChevronRight } from "lucide-react";

import { fetchOrg, fetchSpace, fetchBoard } from "@/lib/user-api";
import { EntitySettingsView } from "@/components/entities/entity-settings-view";

interface BoardSettingsPageProps {
  params: Promise<{ org: string; space: string; board: string }>;
}

export default async function BoardSettingsPage({
  params,
}: BoardSettingsPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug, board: boardSlug } = await params;
  const [org, space, board] = await Promise.all([
    fetchOrg(orgSlug),
    fetchSpace(orgSlug, spaceSlug),
    fetchBoard(orgSlug, spaceSlug, boardSlug),
  ]);

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
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
        <span className="font-medium text-foreground">Settings</span>
      </nav>

      <EntitySettingsView
        entityType="board"
        entitySlug={boardSlug}
        orgSlug={orgSlug}
        spaceSlug={spaceSlug}
        currentValues={{
          id: board.id,
          slug: board.slug,
          name: board.name,
          description: board.description ?? "",
          metadata: board.metadata,
        }}
        deleteRedirect={`/orgs/${orgSlug}/spaces/${spaceSlug}`}
        cancelHref={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}`}
      />
    </div>
  );
}
