import Link from "next/link";
import { ChevronRight, Settings } from "lucide-react";

import { fetchOrg, fetchSpaces } from "@/lib/user-api";
import { EntityListLinked } from "@/components/entities/entity-list-linked";

interface OrgPageProps {
  params: Promise<{ org: string }>;
}

export default async function OrgPage({ params }: OrgPageProps): Promise<React.ReactNode> {
  const { org: orgSlug } = await params;
  const [org, { data: spaces }] = await Promise.all([fetchOrg(orgSlug), fetchSpaces(orgSlug)]);

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        <Link href="/" className="hover:text-foreground">
          Home
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <span className="font-medium text-foreground">{org.name}</span>
        <Link
          href={`/orgs/${orgSlug}/settings`}
          className="ml-auto hover:text-foreground"
          title="Settings"
        >
          <Settings className="h-4 w-4" />
        </Link>
      </nav>

      {org.description && <p className="text-sm text-muted-foreground">{org.description}</p>}

      <EntityListLinked
        entityType="space"
        title="Spaces"
        items={spaces.map((s) => ({
          id: s.id,
          name: s.name,
          slug: s.slug,
          description: s.description,
        }))}
        hrefPrefix={`/orgs/${orgSlug}/spaces`}
      />
    </div>
  );
}
