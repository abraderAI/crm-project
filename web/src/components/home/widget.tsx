"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface WidgetProps {
  /** Unique widget identifier. */
  id: string;
  /** Widget title displayed in the header. */
  title: string;
  /** Widget content. */
  children: ReactNode;
  /** Whether the widget is visible. Hidden widgets are not rendered. */
  visible?: boolean;
  /** Optional additional CSS classes. */
  className?: string;
}

/** Base widget component for home screen grid. Renders a card with title and content. */
export function Widget({ id, title, children, visible = true, className }: WidgetProps): ReactNode {
  if (!visible) {
    return null;
  }

  return (
    <div
      data-testid={`widget-${id}`}
      data-widget-id={id}
      className={cn("rounded-lg border border-border bg-background p-4 shadow-sm", className)}
    >
      <h3 className="mb-3 text-sm font-semibold text-foreground" data-testid={`widget-title-${id}`}>
        {title}
      </h3>
      <div data-testid={`widget-content-${id}`}>{children}</div>
    </div>
  );
}
