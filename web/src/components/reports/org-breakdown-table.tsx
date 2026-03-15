"use client";

import { useCallback, useMemo, useState } from "react";
import Link from "next/link";
import { ChevronDown, ChevronUp } from "lucide-react";
import { cn } from "@/lib/utils";
import type { OrgSalesSummary, OrgSupportSummary } from "@/lib/reporting-types";

/** Variant determines which columns are displayed. */
export type OrgBreakdownVariant = "support" | "sales";

export interface OrgBreakdownTableProps {
  variant: OrgBreakdownVariant;
  data: OrgSupportSummary[] | OrgSalesSummary[];
}

type SortDirection = "asc" | "desc";

interface SortState {
  column: string;
  direction: SortDirection;
}

/** Format a number as currency: "$X,XXX". */
function formatCurrency(value: number | null): string {
  if (value === null || value === undefined) return "–";
  return `$${value.toLocaleString("en-US", { maximumFractionDigits: 0 })}`;
}

/** Format hours: "X.X hrs" or "–" for null. */
function formatHours(value: number | null): string {
  if (value === null || value === undefined) return "–";
  return `${value.toFixed(1)} hrs`;
}

/** Format percentage: "X.X%". */
function formatPercent(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

/** Column definition for the table. */
interface ColumnDef<T> {
  key: string;
  label: string;
  getValue: (row: T) => string | number | null;
  getSortValue: (row: T) => string | number;
  render: (row: T) => React.ReactNode;
}

/** Build support variant column definitions. */
function buildSupportColumns(): ColumnDef<OrgSupportSummary>[] {
  return [
    {
      key: "org_name",
      label: "Org",
      getValue: (r) => r.org_name,
      getSortValue: (r) => r.org_name.toLowerCase(),
      render: (r) => (
        <Link
          href={`/orgs/${r.org_slug}/reports/support`}
          className="text-primary underline-offset-4 hover:underline"
          data-testid={`org-link-${r.org_slug}`}
        >
          {r.org_name}
        </Link>
      ),
    },
    {
      key: "open_count",
      label: "Open Tickets",
      getValue: (r) => r.open_count,
      getSortValue: (r) => r.open_count,
      render: (r) => <span>{r.open_count}</span>,
    },
    {
      key: "overdue_count",
      label: "Overdue",
      getValue: (r) => r.overdue_count,
      getSortValue: (r) => r.overdue_count,
      render: (r) => (
        <span>
          {r.overdue_count > 0 ? (
            <span
              className="inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700"
              data-testid={`overdue-badge-${r.org_slug}`}
            >
              {r.overdue_count}
            </span>
          ) : (
            r.overdue_count
          )}
        </span>
      ),
    },
    {
      key: "avg_resolution_hours",
      label: "Avg Resolution",
      getValue: (r) => r.avg_resolution_hours,
      getSortValue: (r) => r.avg_resolution_hours ?? -1,
      render: (r) => <span>{formatHours(r.avg_resolution_hours)}</span>,
    },
    {
      key: "avg_first_response_hours",
      label: "Avg First Response",
      getValue: (r) => r.avg_first_response_hours,
      getSortValue: (r) => r.avg_first_response_hours ?? -1,
      render: (r) => <span>{formatHours(r.avg_first_response_hours)}</span>,
    },
    {
      key: "total_in_range",
      label: "Total (in range)",
      getValue: (r) => r.total_in_range,
      getSortValue: (r) => r.total_in_range,
      render: (r) => <span>{r.total_in_range}</span>,
    },
  ];
}

/** Build sales variant column definitions. */
function buildSalesColumns(): ColumnDef<OrgSalesSummary>[] {
  return [
    {
      key: "org_name",
      label: "Org",
      getValue: (r) => r.org_name,
      getSortValue: (r) => r.org_name.toLowerCase(),
      render: (r) => (
        <Link
          href={`/orgs/${r.org_slug}/reports/sales`}
          className="text-primary underline-offset-4 hover:underline"
          data-testid={`org-link-${r.org_slug}`}
        >
          {r.org_name}
        </Link>
      ),
    },
    {
      key: "total_leads",
      label: "Total Leads",
      getValue: (r) => r.total_leads,
      getSortValue: (r) => r.total_leads,
      render: (r) => <span>{r.total_leads}</span>,
    },
    {
      key: "win_rate",
      label: "Win Rate",
      getValue: (r) => r.win_rate,
      getSortValue: (r) => r.win_rate,
      render: (r) => <span>{formatPercent(r.win_rate)}</span>,
    },
    {
      key: "avg_deal_value",
      label: "Avg Deal Value",
      getValue: (r) => r.avg_deal_value,
      getSortValue: (r) => r.avg_deal_value ?? -1,
      render: (r) => <span>{formatCurrency(r.avg_deal_value)}</span>,
    },
    {
      key: "open_pipeline_count",
      label: "Open Pipeline",
      getValue: (r) => r.open_pipeline_count,
      getSortValue: (r) => r.open_pipeline_count,
      render: (r) => <span>{r.open_pipeline_count}</span>,
    },
  ];
}

/** Sortable column header with chevron indicator. */
function SortableHeader({
  label,
  columnKey,
  sort,
  onSort,
}: {
  label: string;
  columnKey: string;
  sort: SortState;
  onSort: (key: string) => void;
}): React.ReactNode {
  const isActive = sort.column === columnKey;
  return (
    <th
      className="cursor-pointer select-none whitespace-nowrap px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground hover:text-foreground"
      onClick={() => onSort(columnKey)}
      data-testid={`sort-header-${columnKey}`}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {isActive &&
          (sort.direction === "asc" ? (
            <ChevronUp className="h-3 w-3" data-testid="sort-icon-asc" />
          ) : (
            <ChevronDown className="h-3 w-3" data-testid="sort-icon-desc" />
          ))}
      </span>
    </th>
  );
}

/** Table showing per-org breakdown with client-side sortable columns. */
export function OrgBreakdownTable({ variant, data }: OrgBreakdownTableProps): React.ReactNode {
  const [sort, setSort] = useState<SortState>({ column: "org_name", direction: "asc" });

  const handleSort = useCallback((column: string) => {
    setSort((prev) => ({
      column,
      direction: prev.column === column && prev.direction === "asc" ? "desc" : "asc",
    }));
  }, []);

  const columns = useMemo(() => {
    if (variant === "support") return buildSupportColumns();
    return buildSalesColumns();
  }, [variant]);

  const sortedData = useMemo(() => {
    const col = columns.find((c) => c.key === sort.column);
    if (!col) return [...data];
    const sorted = [...data].sort((a, b) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const aVal = col.getSortValue(a as any);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const bVal = col.getSortValue(b as any);
      if (typeof aVal === "string" && typeof bVal === "string") {
        return sort.direction === "asc" ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      }
      const aNum = Number(aVal);
      const bNum = Number(bVal);
      return sort.direction === "asc" ? aNum - bNum : bNum - aNum;
    });
    return sorted;
  }, [data, sort, columns]);

  return (
    <div
      className="overflow-x-auto rounded-lg border border-border"
      data-testid="org-breakdown-table"
    >
      <table className="w-full text-sm">
        <thead className="border-b border-border bg-muted/50">
          <tr>
            {columns.map((col) => (
              <SortableHeader
                key={col.key}
                label={col.label}
                columnKey={col.key}
                sort={sort}
                onSort={handleSort}
              />
            ))}
          </tr>
        </thead>
        <tbody>
          {sortedData.length === 0 ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-8 text-center text-muted-foreground"
                data-testid="org-breakdown-empty"
              >
                No data
              </td>
            </tr>
          ) : (
            sortedData.map((row) => (
              <tr
                key={
                  variant === "support"
                    ? (row as OrgSupportSummary).org_id
                    : (row as OrgSalesSummary).org_id
                }
                className={cn(
                  "border-b border-border last:border-b-0 transition-colors hover:bg-muted/30",
                )}
                data-testid="org-breakdown-row"
              >
                {columns.map((col) => (
                  <td key={col.key} className="whitespace-nowrap px-4 py-3">
                    {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                    {col.render(row as any)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
