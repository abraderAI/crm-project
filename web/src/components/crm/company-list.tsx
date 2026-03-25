"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { Building2, Plus, AlertTriangle, Search } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import { parseLeadData, COMPANY_STATUS_LABELS, type CompanyStatus } from "@/lib/crm-types";
import { fetchCompanies } from "@/lib/crm-api";
import { useTier } from "@/hooks/use-tier";

/** Filter values for the company list. */
interface CompanyFilterValues {
  search: string;
  status: string;
}

const DEFAULT_FILTERS: CompanyFilterValues = { search: "", status: "all" };

/** Company list view with sortable table, filters, and cursor pagination. */
export function CompanyList(): React.ReactNode {
  const { tier, orgId, isLoading: tierLoading } = useTier();
  const { getToken } = useAuth();

  const [companies, setCompanies] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [filters, setFilters] = useState<CompanyFilterValues>(DEFAULT_FILTERS);
  const [sortField, setSortField] = useState<string>("created_at");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");
  const mountedRef = useRef(true);

  const hasAccess = tier >= 4;

  const loadCompanies = useCallback(
    async (cursor?: string): Promise<void> => {
      if (!orgId) return;
      setIsLoading(true);
      setError(null);
      try {
        const token = await getToken();
        if (!token) return;

        const params: Record<string, string> = {
          limit: "50",
          sort: sortField,
          order: sortDir,
        };
        if (cursor) params.cursor = cursor;
        if (filters.status !== "all") params["metadata[status]"] = filters.status;
        if (filters.search) params.search = filters.search;

        const result = await fetchCompanies(token, orgId, params);
        if (!mountedRef.current) return;

        if (cursor) {
          setCompanies((prev) => [...prev, ...result.data]);
        } else {
          setCompanies(result.data);
        }
        setHasMore(result.page_info.has_more);
        setNextCursor(result.page_info.next_cursor);
      } catch (err) {
        if (mountedRef.current) {
          setError(err instanceof Error ? err.message : "Failed to load companies");
        }
      } finally {
        if (mountedRef.current) setIsLoading(false);
      }
    },
    [getToken, orgId, sortField, sortDir, filters],
  );

  useEffect(() => {
    mountedRef.current = true;
    if (!tierLoading && hasAccess) {
      void loadCompanies();
    }
    return () => {
      mountedRef.current = false;
    };
  }, [tierLoading, hasAccess, loadCompanies]);

  const toggleSort = (field: string): void => {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  if (tierLoading) {
    return (
      <div
        data-testid="company-list-loading"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        Loading...
      </div>
    );
  }

  if (!hasAccess) {
    return (
      <div
        data-testid="company-list-denied"
        className="flex flex-col items-center gap-3 py-12 text-center"
      >
        <AlertTriangle className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm font-medium text-foreground">Access Denied</p>
      </div>
    );
  }

  return (
    <div data-testid="company-list" className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Building2 className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold text-foreground">Companies</h2>
          {!isLoading && (
            <span
              className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
              data-testid="company-count"
            >
              {companies.length}
            </span>
          )}
        </div>
        <Link
          href="/crm/companies/new"
          className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          data-testid="company-create-btn"
        >
          <Plus className="h-4 w-4" />
          New Company
        </Link>
      </div>

      {/* Error */}
      {error && (
        <div
          data-testid="company-list-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3" data-testid="company-filters">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="search"
            value={filters.search}
            onChange={(e) => setFilters((f) => ({ ...f, search: e.target.value }))}
            placeholder="Search companies..."
            data-testid="company-search-input"
            className="h-8 w-56 rounded-md border border-border bg-background pl-8 pr-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <select
          value={filters.status}
          onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
          data-testid="company-status-filter"
          className="h-8 rounded-md border border-border bg-background px-2 text-sm"
        >
          <option value="all">All statuses</option>
          {Object.entries(COMPANY_STATUS_LABELS).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>

      {/* Table */}
      {isLoading && companies.length === 0 ? (
        <div
          data-testid="company-table-loading"
          className="py-8 text-center text-sm text-muted-foreground"
        >
          Loading companies...
        </div>
      ) : companies.length === 0 ? (
        <div
          data-testid="company-table-empty"
          className="py-8 text-center text-sm text-muted-foreground"
        >
          No companies found.
        </div>
      ) : (
        <div
          className="overflow-x-auto rounded-lg border border-border"
          data-testid="company-table"
        >
          <table className="w-full text-sm">
            <thead className="border-b border-border bg-muted/50">
              <tr>
                <th
                  className="cursor-pointer px-4 py-2 text-left font-medium text-foreground"
                  onClick={() => toggleSort("title")}
                  data-testid="company-sort-name"
                >
                  Name {sortField === "title" && (sortDir === "asc" ? "↑" : "↓")}
                </th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Industry</th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Status</th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Owner</th>
                <th
                  className="cursor-pointer px-4 py-2 text-left font-medium text-foreground"
                  onClick={() => toggleSort("created_at")}
                  data-testid="company-sort-date"
                >
                  Created {sortField === "created_at" && (sortDir === "asc" ? "↑" : "↓")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {companies.map((company) => {
                const meta = parseLeadData(company.metadata);
                return (
                  <tr
                    key={company.id}
                    className="transition-colors hover:bg-accent/50"
                    data-testid={`company-row-${company.id}`}
                  >
                    <td className="px-4 py-2">
                      <Link
                        href={`/crm/companies/${company.id}`}
                        className="font-medium text-primary hover:underline"
                      >
                        {company.title}
                      </Link>
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">{meta.source ?? "—"}</td>
                    <td className="px-4 py-2">
                      <span className="rounded-full bg-muted px-2 py-0.5 text-xs">
                        {COMPANY_STATUS_LABELS[meta.crm_type as CompanyStatus] ??
                          meta.crm_type ??
                          "—"}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">{meta.assigned_to ?? "—"}</td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {new Date(company.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Load more */}
      {hasMore && (
        <div className="flex justify-center">
          <button
            onClick={() => void loadCompanies(nextCursor)}
            disabled={isLoading}
            data-testid="company-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
          >
            {isLoading ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
