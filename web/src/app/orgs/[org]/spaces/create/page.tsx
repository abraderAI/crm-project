import { EntityCreateView } from "@/components/entities/entity-create-view";

interface CreateSpacePageProps {
  params: Promise<{ org: string }>;
}

/** Page for creating a new space within an org. */
export default async function CreateSpacePage({
  params,
}: CreateSpacePageProps): Promise<React.ReactNode> {
  const { org: orgSlug } = await params;

  return <EntityCreateView entityKind="space" orgSlug={orgSlug} cancelHref={`/orgs/${orgSlug}`} />;
}
