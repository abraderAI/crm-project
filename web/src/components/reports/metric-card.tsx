"use client";

import Link from "next/link";
import { cn } from "@/lib/utils";

export interface MetricCardProps {
  /** Display label above the value. */
  label: string;
  /** The metric value to display. */
  value: string | number;
  /** Optional secondary label below the value. */
  subLabel?: string;
  /** If set, wraps the card in a Next.js Link. */
  href?: string;
  /** When true, shows a skeleton placeholder in place of the value. */
  loading?: boolean;
}

/** Skeleton placeholder matching the value area. */
function ValueSkeleton(): React.ReactNode {
  return (
    <div data-testid="metric-card-skeleton" className="h-8 w-20 animate-pulse rounded bg-muted" />
  );
}

/** Card content shared between linked and static variants. */
function CardInner({
  label,
  value,
  subLabel,
  loading = false,
}: Omit<MetricCardProps, "href">): React.ReactNode {
  return (
    <div className="flex flex-col gap-1 p-4" data-testid="metric-card-inner">
      <p className="text-xs text-muted-foreground">{label}</p>
      {loading ? (
        <ValueSkeleton />
      ) : (
        <p className="text-2xl font-semibold text-foreground" data-testid="metric-card-value">
          {value}
        </p>
      )}
      {subLabel && (
        <p className="text-xs text-muted-foreground" data-testid="metric-card-sub-label">
          {subLabel}
        </p>
      )}
    </div>
  );
}

/** Reusable metric card for report dashboards. */
export function MetricCard({
  label,
  value,
  subLabel,
  href,
  loading = false,
}: MetricCardProps): React.ReactNode {
  const card = (
    <div
      data-testid="metric-card"
      className={cn(
        "rounded-lg border border-border bg-background",
        href && "transition-colors hover:border-foreground/20 hover:bg-accent/50",
      )}
    >
      <CardInner label={label} value={value} subLabel={subLabel} loading={loading} />
    </div>
  );

  if (href) {
    return (
      <Link href={href} data-testid="metric-card-link">
        {card}
      </Link>
    );
  }

  return card;
}
