import type { ReactNode } from "react";

interface ForumPageProps {
  params: Promise<{ slug: string[] }>;
}

/** Public forum page — renders threads from global-forum space. */
export default async function ForumPage({ params }: ForumPageProps): Promise<ReactNode> {
  const { slug } = await params;
  const path = slug.join("/");

  return (
    <div data-testid="forum-page">
      <h1 className="mb-4 text-2xl font-bold text-foreground">Community Forum</h1>
      <p className="text-muted-foreground">
        Viewing: <code className="text-foreground">{path}</code>
      </p>
      {/* Content will be fetched from global-forum space in Phase 3 */}
    </div>
  );
}
