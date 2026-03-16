import type { ReactNode } from "react";

interface DocsPageProps {
  params: Promise<{ slug: string[] }>;
}

/** Public documentation page — renders content from global-docs space. */
export default async function DocsPage({ params }: DocsPageProps): Promise<ReactNode> {
  const { slug } = await params;
  const path = slug.join("/");

  return (
    <div data-testid="docs-page">
      <h1 className="mb-4 text-2xl font-bold text-foreground">Documentation</h1>
      <p className="text-muted-foreground">
        Viewing: <code className="text-foreground">{path}</code>
      </p>
      {/* Content will be fetched from global-docs space in Phase 3 */}
    </div>
  );
}
