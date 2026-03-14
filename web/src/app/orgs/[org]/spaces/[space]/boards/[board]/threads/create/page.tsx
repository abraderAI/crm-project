import { ThreadCreateView } from "@/components/thread/thread-create-view";

interface CreateThreadPageProps {
  params: Promise<{ org: string; space: string; board: string }>;
}

/** Page for creating a new thread within a board. */
export default async function CreateThreadPage({
  params,
}: CreateThreadPageProps): Promise<React.ReactNode> {
  const { org: orgSlug, space: spaceSlug, board: boardSlug } = await params;

  return (
    <ThreadCreateView
      orgSlug={orgSlug}
      spaceSlug={spaceSlug}
      boardSlug={boardSlug}
      cancelHref={`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}`}
    />
  );
}
