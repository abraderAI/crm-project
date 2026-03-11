"use client";

import { ChevronRight } from "lucide-react";

/** A single breadcrumb segment. */
export interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface BreadcrumbsProps {
  items: BreadcrumbItem[];
}

/** Breadcrumb navigation showing the current hierarchy path. */
export function Breadcrumbs({ items }: BreadcrumbsProps): React.ReactNode {
  if (items.length === 0) return null;

  return (
    <nav aria-label="Breadcrumb" data-testid="breadcrumbs">
      <ol className="flex items-center gap-1 text-sm text-foreground/60">
        {items.map((item, index) => {
          const isLast = index === items.length - 1;
          return (
            <li key={`${item.label}-${index}`} className="flex items-center gap-1">
              {index > 0 && <ChevronRight className="h-3.5 w-3.5 shrink-0" />}
              {isLast || !item.href ? (
                <span
                  className={isLast ? "font-medium text-foreground" : ""}
                  aria-current={isLast ? "page" : undefined}
                  data-testid={`breadcrumb-${index}`}
                >
                  {item.label}
                </span>
              ) : (
                <a
                  href={item.href}
                  className="transition-colors hover:text-foreground"
                  data-testid={`breadcrumb-${index}`}
                >
                  {item.label}
                </a>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
