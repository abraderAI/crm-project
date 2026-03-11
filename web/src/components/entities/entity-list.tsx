"use client";

import { Plus } from "lucide-react";
import { EntityCard, type EntityType } from "./entity-card";

export interface EntityListItem {
  id: string;
  name: string;
  slug: string;
  description?: string;
  metadata?: string;
}

export interface EntityListProps {
  /** The type of entities in this list. */
  entityType: EntityType;
  /** Entity items to display. */
  items: EntityListItem[];
  /** Whether data is currently loading. */
  loading?: boolean;
  /** Called when the user clicks the create button. */
  onCreate?: () => void;
  /** Called when the user clicks an entity card. */
  onSelect?: (id: string) => void;
  /** Builds an href for each entity card. */
  getHref?: (item: EntityListItem) => string;
  /** Whether there are more pages. */
  hasMore?: boolean;
  /** Called when the user requests the next page. */
  onLoadMore?: () => void;
  /** Label for the entity type, e.g. "Organizations". */
  title: string;
}

/** Paginated entity list with optional create button and load-more pagination. */
export function EntityList({
  entityType,
  items,
  loading = false,
  onCreate,
  onSelect,
  getHref,
  hasMore = false,
  onLoadMore,
  title,
}: EntityListProps): React.ReactNode {
  return (
    <div data-testid="entity-list" className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-foreground">{title}</h1>
        {onCreate && (
          <button
            onClick={onCreate}
            data-testid="entity-create-btn"
            className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            Create
          </button>
        )}
      </div>

      {/* Empty state */}
      {!loading && items.length === 0 && (
        <p className="py-8 text-center text-sm text-muted-foreground" data-testid="empty-state">
          No {title.toLowerCase()} found.
        </p>
      )}

      {/* Loading state */}
      {loading && (
        <div className="py-8 text-center text-sm text-muted-foreground" data-testid="loading-state">
          Loading...
        </div>
      )}

      {/* Entity cards */}
      {!loading && items.length > 0 && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" data-testid="entity-grid">
          {items.map((item) => (
            <EntityCard
              key={item.id}
              id={item.id}
              name={item.name}
              slug={item.slug}
              description={item.description}
              entityType={entityType}
              metadata={item.metadata}
              onClick={onSelect}
              href={getHref?.(item)}
            />
          ))}
        </div>
      )}

      {/* Load more */}
      {hasMore && !loading && (
        <div className="flex justify-center pt-2">
          <button
            onClick={onLoadMore}
            data-testid="load-more-btn"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-accent"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}
