import { Search as SearchIcon } from "lucide-react";

import { fetchSearch } from "@/lib/user-api";
import type { SearchResult } from "@/lib/api-types";

interface SearchPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

/** Render a single search result. */
function SearchResultItem({ result }: { result: SearchResult }): React.ReactNode {
  return (
    <div className="rounded-lg border border-border bg-background p-4">
      <div className="flex items-center gap-2">
        <span className="rounded bg-muted px-1.5 py-0.5 text-xs font-medium text-muted-foreground">
          {result.entity_type}
        </span>
        <span className="text-sm font-medium text-foreground">{result.title}</span>
      </div>
      {result.snippet && <p className="mt-1 text-xs text-muted-foreground">{result.snippet}</p>}
    </div>
  );
}

/** Search page — full-text search across all entities. */
export default async function SearchPage({
  searchParams,
}: SearchPageProps): Promise<React.ReactNode> {
  const params = await searchParams;
  const query = typeof params.q === "string" ? params.q : "";
  const results = query ? await fetchSearch(query) : null;

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <h1 className="text-xl font-bold text-foreground">Search</h1>

      {/* Search form */}
      <form action="/search" method="GET" className="flex gap-2">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            name="q"
            defaultValue={query}
            placeholder="Search threads, orgs, spaces..."
            className="w-full rounded-md border border-border bg-background py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
        <button
          type="submit"
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          Search
        </button>
      </form>

      {/* Results */}
      {results && (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            {results.data.length} result{results.data.length !== 1 ? "s" : ""} for &ldquo;{query}
            &rdquo;
          </p>
          {results.data.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No results found. Try a different query.
            </p>
          )}
          {results.data.map((result, i) => (
            <SearchResultItem
              key={`${result.entity_type}-${result.entity_id}-${i}`}
              result={result}
            />
          ))}
        </div>
      )}
    </div>
  );
}
