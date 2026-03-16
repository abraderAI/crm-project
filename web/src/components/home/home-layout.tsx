"use client";

import type { ReactNode } from "react";
import type { WidgetConfig } from "@/lib/tier-types";
import { Widget } from "./widget";

/** Registry entry mapping widget_id to a title and render function. */
export interface WidgetRegistryEntry {
  title: string;
  render: () => ReactNode;
}

/** Map of widget_id to its registry entry. */
export type WidgetRegistry = Record<string, WidgetRegistryEntry>;

export interface HomeLayoutProps {
  /** Ordered list of widget configs (visibility + order). */
  layout: WidgetConfig[];
  /** Registry of available widgets with their titles and render functions. */
  registry: WidgetRegistry;
  /** Optional additional CSS classes for the grid container. */
  className?: string;
}

/**
 * Renders a responsive CSS grid of Widget components based on layout config.
 * Widgets not in the registry are silently skipped.
 */
export function HomeLayout({ layout, registry, className }: HomeLayoutProps): ReactNode {
  const visibleWidgets = layout.filter((w) => w.visible && registry[w.widget_id]);

  if (visibleWidgets.length === 0) {
    return (
      <div data-testid="home-layout-empty" className="py-8 text-center text-muted-foreground">
        No widgets to display.
      </div>
    );
  }

  return (
    <div
      data-testid="home-layout"
      className={`grid gap-4 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3 ${className ?? ""}`}
    >
      {visibleWidgets.map((w) => {
        const entry = registry[w.widget_id];
        if (!entry) return null;
        return (
          <Widget key={w.widget_id} id={w.widget_id} title={entry.title}>
            {entry.render()}
          </Widget>
        );
      })}
    </div>
  );
}
