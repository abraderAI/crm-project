"use client";

import { Building2, FolderOpen, Layout } from "lucide-react";
import { cn } from "@/lib/utils";

/** Entity type determines the icon and styling. */
export type EntityType = "org" | "space" | "board";

export interface EntityCardProps {
  /** Unique entity ID. */
  id: string;
  /** Display name. */
  name: string;
  /** URL slug. */
  slug: string;
  /** Optional description. */
  description?: string;
  /** Entity type for icon selection. */
  entityType: EntityType;
  /** Optional metadata string (JSON) to display key details. */
  metadata?: string;
  /** Click handler when the card is selected. */
  onClick?: (id: string) => void;
  /** Optional href for navigation. */
  href?: string;
}

const ICONS: Record<EntityType, typeof Building2> = {
  org: Building2,
  space: FolderOpen,
  board: Layout,
};

const LABELS: Record<EntityType, string> = {
  org: "Organization",
  space: "Space",
  board: "Board",
};

/** Parse metadata JSON safely, returning key-value pairs for display. */
function parseMetadata(metadata?: string): Record<string, unknown> | null {
  if (!metadata || metadata === "{}" || metadata === "") return null;
  try {
    const parsed: unknown = JSON.parse(metadata);
    if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    return null;
  } catch {
    return null;
  }
}

/** Card component for displaying an entity (org, space, or board) in a list. */
export function EntityCard({
  id,
  name,
  slug,
  description,
  entityType,
  metadata,
  onClick,
  href,
}: EntityCardProps): React.ReactNode {
  const Icon = ICONS[entityType];
  const label = LABELS[entityType];
  const parsed = parseMetadata(metadata);

  const content = (
    <div
      data-testid={`entity-card-${id}`}
      className={cn(
        "group flex flex-col gap-2 rounded-lg border border-border bg-background p-4 transition-colors",
        (onClick ?? href) && "cursor-pointer hover:border-primary/50 hover:bg-accent",
      )}
      onClick={onClick ? () => onClick(id) : undefined}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onClick(id);
              }
            }
          : undefined
      }
    >
      <div className="flex items-center gap-2">
        <Icon className="h-5 w-5 shrink-0 text-muted-foreground" data-testid="entity-icon" />
        <div className="flex-1 truncate">
          <h3 className="truncate text-sm font-semibold text-foreground">{name}</h3>
          <span className="text-xs text-muted-foreground">
            {label} · {slug}
          </span>
        </div>
      </div>
      {description && (
        <p className="line-clamp-2 text-sm text-muted-foreground" data-testid="entity-description">
          {description}
        </p>
      )}
      {parsed && Object.keys(parsed).length > 0 && (
        <div className="flex flex-wrap gap-1" data-testid="entity-metadata">
          {Object.entries(parsed)
            .slice(0, 3)
            .map(([key, value]) => (
              <span
                key={key}
                className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
              >
                {key}: {String(value)}
              </span>
            ))}
        </div>
      )}
    </div>
  );

  if (href) {
    return (
      <a href={href} className="block no-underline" data-testid={`entity-link-${id}`}>
        {content}
      </a>
    );
  }

  return content;
}
