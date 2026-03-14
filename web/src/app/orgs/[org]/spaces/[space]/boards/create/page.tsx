import { EntityCreateView } from "@/components/entities/entity-create-view";

interface CreateBoardPageProps {
  params: Promise<{ org: string; space: string }>;
}

/** Page for creating a new board within a space. */
export default async function CreateBoardPage({
  params,
}: CreateBoardPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug } = await params;

  return (
    <EntityCreateView
      entityKind="board"
      orgSlug={orgSlug}
      spaceSlug={spaceSlug}
      cancelHref={`/orgs/${orgSlug}/spaces/${spaceSlug}`}
    />
  );
}
