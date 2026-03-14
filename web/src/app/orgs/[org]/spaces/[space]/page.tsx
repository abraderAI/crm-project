import Link from "next/link";
import { ChevronRight, Settings } from "lucide-react";

import { fetchOrg, fetchSpace, fetchBoards } from "@/lib/user-api";
import { EntityListLinked } from "@/components/entities/entity-list-linked";

interface SpacePageProps {
  params: Promise<{ org: string; space: string }>;
}

export default async function SpacePage({ params }: SpacePageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug } = await params;
  const [org, space, { data: boards }] = await Promise.all([
    fetchOrg(orgSlug),
    fetchSpace(orgSlug, spaceSlug),
    fetchBoards(orgSlug, spaceSlug),
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
        <span className="font-medium text-foreground">{space.name}</span>
        <Link
          href={`/orgs/${orgSlug}/spaces/${spaceSlug}/settings`}
          className="ml-auto hover:text-foreground"
          title="Settings"
        >
          <Settings className="h-4 w-4" />
        </Link>
      </nav>

      {space.description && <p className="text-sm text-muted-foreground">{space.description}</p>}

      <EntityListLinked
        entityType="board"
        title="Boards"
        items={boards.map((b) => ({
          id: b.id,
          name: b.name,
          slug: b.slug,
          description: b.description,
        }))}
        hrefPrefix={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards`}
      />
    </div>
  );
}
