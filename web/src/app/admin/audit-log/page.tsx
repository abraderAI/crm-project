"use client";
export const dynamic = "force-dynamic";

import { Suspense, useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";

import type { AuditEntry, PaginatedResponse } from "@/lib/api-types";
import { buildHeaders, buildUrl, parseResponse } from "@/lib/api-client";
import { DateRangePicker } from "@/components/reports/date-range-picker";
import { AuditLogViewerWithDirectory } from "@/components/admin/audit-log-viewer-wrapper";

/** Action filter options. */
const ACTION_OPTIONS = [
  { value: "", label: "All Actions" },
  { value: "create", label: "Create" },
  { value: "update", label: "Update" },
  { value: "delete", label: "Delete" },
] as const;

/** Entity type filter options. */
const ENTITY_TYPE_OPTIONS = [
  { value: "", label: "All Types" },
  { value: "org", label: "Organization" },
  { value: "user", label: "User" },
  { value: "thread", label: "Thread" },
  { value: "platform_admin", label: "Platform Admin" },
  { value: "feature_flag", label: "Feature Flag" },
  { value: "system_settings", label: "System Settings" },
  { value: "rbac_policy", label: "RBAC Policy" },
] as const;

/** Format a Date as YYYY-MM-DD. */
function toDateString(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

/** Parse a YYYY-MM-DD string into a Date, or return fallback. */
function parseDate(value: string | null, fallback: Date): Date {
  if (!value) return fallback;
  const d = new Date(value + "T00:00:00");
  return isNaN(d.getTime()) ? fallback : d;
}

/** Convert a Date to RFC3339 for the backend `after`/`before` params. */
function toRFC3339(date: Date): string {
  return date.toISOString();
}

/** End-of-day for a given date (23:59:59.999). */
function endOfDay(date: Date): Date {
  const d = new Date(date);
  d.setHours(23, 59, 59, 999);
  return d;
}

function AdminAuditLogInner(): React.ReactNode {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { getToken } = useAuth();

  const now = new Date();
  const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

  const [from, setFrom] = useState<Date>(parseDate(searchParams.get("from"), thirtyDaysAgo));
  const [to, setTo] = useState<Date>(parseDate(searchParams.get("to"), now));
  const [action, setAction] = useState(searchParams.get("action") ?? "");
  const [entityType, setEntityType] = useState(searchParams.get("entity_type") ?? "");
  const [userSearch, setUserSearch] = useState(searchParams.get("user") ?? "");

  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(true);

  /** Build query params from current filter state. */
  const buildParams = useCallback(
    (nextCursor?: string): Record<string, string> => {
      const params: Record<string, string> = {
        after: toRFC3339(from),
        before: toRFC3339(endOfDay(to)),
      };
      if (action) params["action"] = action;
      if (entityType) params["entity_type"] = entityType;
      if (userSearch.trim()) params["user"] = userSearch.trim();
      if (nextCursor) params["cursor"] = nextCursor;
      return params;
    },
    [from, to, action, entityType, userSearch],
  );

  /** Fetch entries from the API. */
  const fetchEntries = useCallback(
    async (append = false): Promise<void> => {
      setLoading(true);
      try {
        const token = await getToken();
        const params = buildParams(append ? cursor : undefined);
        const url = buildUrl("/admin/audit-log", params);
        const response = await fetch(url, {
          method: "GET",
          headers: buildHeaders(token),
          cache: "no-store",
        });
        const result = await parseResponse<PaginatedResponse<AuditEntry>>(response);
        setEntries((prev) => (append ? [...prev, ...result.data] : result.data));
        setHasMore(result.page_info.has_more);
        setCursor(result.page_info.next_cursor);
      } catch {
        // On error, clear entries to avoid stale data.
        if (!append) setEntries([]);
      } finally {
        setLoading(false);
      }
    },
    [getToken, buildParams, cursor],
  );

  /** Re-fetch on filter changes (not appending). */
  useEffect(() => {
    setCursor(undefined);
    void fetchEntries(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, to, action, entityType, userSearch]);

  /** Sync filter state to URL. */
  useEffect(() => {
    const params = new URLSearchParams();
    params.set("from", toDateString(from));
    params.set("to", toDateString(to));
    if (action) params.set("action", action);
    if (entityType) params.set("entity_type", entityType);
    if (userSearch.trim()) params.set("user", userSearch.trim());
    router.replace(`/admin/audit-log?${params.toString()}`, { scroll: false });
  }, [from, to, action, entityType, userSearch, router]);

  function handleDateChange(range: { from: Date; to: Date }): void {
    setFrom(range.from);
    setTo(range.to);
  }

  const selectClass =
    "rounded-md border border-border bg-background px-2 py-2 text-sm text-foreground";

  return (
    <div data-testid="admin-audit-log" className="flex flex-col gap-4">
      {/* Filter bar */}
      <div
        className="flex flex-wrap items-center gap-3"
        data-testid="audit-filter-bar"
      >
        <DateRangePicker from={from} to={to} onChange={handleDateChange} />

        <select
          value={action}
          onChange={(e) => setAction(e.target.value)}
          data-testid="audit-filter-action"
          className={selectClass}
        >
          {ACTION_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>

        <select
          value={entityType}
          onChange={(e) => setEntityType(e.target.value)}
          data-testid="audit-filter-entity-type"
          className={selectClass}
        >
          {ENTITY_TYPE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>

        <input
          type="text"
          value={userSearch}
          onChange={(e) => setUserSearch(e.target.value)}
          placeholder="Filter by user ID…"
          data-testid="audit-filter-user"
          className="rounded-md border border-border bg-background px-2 py-2 text-sm text-foreground placeholder:text-muted-foreground"
        />
      </div>

      {/* Results */}
      <AuditLogViewerWithDirectory
        entries={entries}
        loading={loading}
        hasMore={hasMore}
        onLoadMore={() => void fetchEntries(true)}
      />
    </div>
  );
}

export default function AdminAuditLogPage(): React.ReactNode {
  return (
    <Suspense>
      <AdminAuditLogInner />
    </Suspense>
  );
}
