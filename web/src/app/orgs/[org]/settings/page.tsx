import Link from "next/link";
import { ChevronRight } from "lucide-react";

import { fetchOrg } from "@/lib/user-api";
import { EntitySettingsView } from "@/components/entities/entity-settings-view";

interface OrgSettingsPageProps {
  params: Promise<{ org: string }>;
}

export default async function OrgSettingsPage({
  params,
}: OrgSettingsPageProps): Promise<React.ReactNode> {
  const { org: orgSlug } = await params;
  const org = await fetchOrg(orgSlug);

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
        <span className="font-medium text-foreground">Settings</span>
      </nav>

      <EntitySettingsView
        entityType="org"
        entitySlug={orgSlug}
        currentValues={{
          id: org.id,
          slug: org.slug,
          name: org.name,
          description: org.description ?? "",
          metadata: org.metadata,
        }}
        deleteRedirect="/"
        cancelHref={`/orgs/${orgSlug}`}
      />
    </div>
  );
}
