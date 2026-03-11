"use client";

import { Plus } from "lucide-react";
import { EntityCard, type EntityCardProps } from "./entity-card";

export interface EntityListProps {
  /** Entities to display. */
  items: EntityCardProps[];
  /** Title for the list heading. */
  title: string;
  /** Whether to show the create button. */
  showCreate?: boolean;
  /** Called when the create button is clicked. */
  onCreate?: () => void;
  /** Create button label. */
  createLabel?: string;
  /** Whether more items can be loaded. */
  hasMore?: boolean;
  /** Called when the Load More button is clicked. */
  onLoadMore?: () => void;
  /** Whether the list is currently loading. */
  loading?: boolean;
  /** Empty state message. */
  emptyMessage?: string;
}

/** Paginated list of entity cards with optional create button and load-more. */
export function EntityList({
  items,
  title,
  showCreate = false,
  onCreate,
  createLabel = "Create",
  hasMore = false,
  onLoadMore,
  loading = false,
  emptyMessage = "No items found.",
}: EntityListProps): React.ReactNode {
  return (
    <div data-testid="entity-list">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">{title}</h2>
        {showCreate && (
          <button
            onClick={onCreate}
            data-testid="entity-create-btn"
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            {createLabel}
          </button>
        )}
      </div>

      {loading && items.length === 0 ? (
        <div
          className="py-12 text-center text-sm text-muted-foreground"
          data-testid="entity-list-loading"
        >
          Loading...
        </div>
      ) : items.length === 0 ? (
        <div
          className="py-12 text-center text-sm text-muted-foreground"
          data-testid="entity-list-empty"
        >
          {emptyMessage}
        </div>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" data-testid="entity-grid">
            {items.map((item) => (
              <EntityCard key={item.id} {...item} />
            ))}
          </div>

          {hasMore && (
            <div className="mt-4 flex justify-center">
              <button
                onClick={onLoadMore}
                disabled={loading}
                data-testid="entity-load-more"
                className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
              >
                {loading ? "Loading..." : "Load more"}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
