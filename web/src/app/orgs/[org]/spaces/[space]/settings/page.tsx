import Link from "next/link";
import { ChevronRight } from "lucide-react";

import { fetchOrg, fetchSpace } from "@/lib/user-api";
import { EntitySettingsView } from "@/components/entities/entity-settings-view";

interface SpaceSettingsPageProps {
  params: Promise<{ org: string; space: string }>;
}

export default async function SpaceSettingsPage({
  params,
}: SpaceSettingsPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug } = await params;
  const [org, space] = await Promise.all([fetchOrg(orgSlug), fetchSpace(orgSlug, spaceSlug)]);

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
        <span className="font-medium text-foreground">Settings</span>
      </nav>

      <EntitySettingsView
        entityType="space"
        entitySlug={spaceSlug}
        orgSlug={orgSlug}
        currentValues={{
          id: space.id,
          slug: space.slug,
          name: space.name,
          description: space.description ?? "",
          metadata: space.metadata,
          type: space.type,
        }}
        deleteRedirect={`/orgs/${orgSlug}`}
        cancelHref={`/orgs/${orgSlug}/spaces/${spaceSlug}`}
      />
    </div>
  );
}
