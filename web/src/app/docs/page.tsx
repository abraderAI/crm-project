import Link from "next/link";
import { BookOpen } from "lucide-react";

import { fetchGlobalThreads, GLOBAL_SPACES } from "@/lib/global-api";
import type { Thread } from "@/lib/api-types";

/**
 * Docs index page — lists recent articles from global-docs.
 * Public route (no auth required). Individual articles link to /docs/[slug].
 */
export default async function DocsIndexPage(): Promise<React.ReactNode> {
  let articles: Thread[] = [];
  try {
    const result = await fetchGlobalThreads(GLOBAL_SPACES.DOCS, { limit: 20 });
    articles = result.data;
  } catch {
    // Render empty state on API failure — docs are non-critical.
  }

  return (
    <div data-testid="docs-index-page" className="mx-auto max-w-3xl space-y-6 p-6">
      <h1 className="text-xl font-bold text-foreground">Documentation</h1>

      {articles.length === 0 ? (
        <p data-testid="docs-empty" className="text-sm text-muted-foreground">
          No documentation available yet.
        </p>
      ) : (
        <ul
          data-testid="docs-article-list"
          className="divide-y divide-border rounded-lg border border-border"
        >
          {articles.map((article) => (
            <li key={article.id} data-testid={`docs-article-${article.id}`}>
              <Link
                href={`/docs/${article.slug}`}
                className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50"
              >
                <BookOpen className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                <span className="text-sm font-medium text-foreground">{article.title}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
