"use client";

import { Building2, FolderKanban, Layout } from "lucide-react";
import { cn } from "@/lib/utils";

/** Supported entity types for the card. */
export type EntityType = "org" | "space" | "board";

export interface EntityCardProps {
  id: string;
  name: string;
  slug: string;
  description?: string;
  entityType: EntityType;
  /** Space type label (only for spaces). */
  spaceType?: string;
  href: string;
  metadata?: Record<string, unknown>;
}

const ICONS: Record<EntityType, typeof Building2> = {
  org: Building2,
  space: FolderKanban,
  board: Layout,
};

const LABELS: Record<EntityType, string> = {
  org: "Organization",
  space: "Space",
  board: "Board",
};

/** Parse metadata JSON string safely. */
export function parseMetadata(raw?: string | Record<string, unknown>): Record<string, unknown> {
  if (!raw) return {};
  if (typeof raw === "object") return raw;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    return {};
  } catch {
    return {};
  }
}

/** Count top-level metadata keys. */
export function metadataKeyCount(metadata?: Record<string, unknown>): number {
  if (!metadata) return 0;
  return Object.keys(metadata).length;
}

/** Card component for displaying an entity (org, space, or board). */
export function EntityCard({
  id,
  name,
  slug,
  description,
  entityType,
  spaceType,
  href,
  metadata,
}: EntityCardProps): React.ReactNode {
  const Icon = ICONS[entityType];
  const keyCount = metadataKeyCount(metadata);

  return (
    <a
      href={href}
      data-testid={`entity-card-${id}`}
      className={cn(
        "group flex flex-col gap-2 rounded-lg border border-border bg-background p-4",
        "transition-colors hover:border-primary/50 hover:bg-accent/50",
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <Icon className="h-5 w-5 shrink-0 text-muted-foreground" />
          <h3 className="text-sm font-semibold text-foreground group-hover:text-primary">{name}</h3>
        </div>
        <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
          {spaceType ?? LABELS[entityType]}
        </span>
      </div>
      {description && (
        <p
          className="line-clamp-2 text-sm text-muted-foreground"
          data-testid={`entity-card-desc-${id}`}
        >
          {description}
        </p>
      )}
      <div className="flex items-center gap-3 text-xs text-muted-foreground">
        <span data-testid={`entity-card-slug-${id}`}>/{slug}</span>
        {keyCount > 0 && (
          <span data-testid={`entity-card-meta-${id}`}>
            {keyCount} metadata {keyCount === 1 ? "field" : "fields"}
          </span>
        )}
      </div>
    </a>
  );
}
