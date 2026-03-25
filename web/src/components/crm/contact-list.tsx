"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { Users, Plus, AlertTriangle, Search } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import { parseLeadData } from "@/lib/crm-types";
import { fetchContacts } from "@/lib/crm-api";
import { useTier } from "@/hooks/use-tier";

/** Contact list view with filterable table and cursor pagination. */
export function ContactList(): React.ReactNode {
  const { tier, orgId, isLoading: tierLoading } = useTier();
  const { getToken } = useAuth();

  const [contacts, setContacts] = useState<Thread[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [search, setSearch] = useState("");
  const mountedRef = useRef(true);

  const hasAccess = tier >= 4;

  const loadContacts = useCallback(
    async (cursor?: string): Promise<void> => {
      if (!orgId) return;
      setIsLoading(true);
      setError(null);
      try {
        const token = await getToken();
        if (!token) return;

        const params: Record<string, string> = { limit: "50" };
        if (cursor) params.cursor = cursor;
        if (search) params.search = search;

        const result = await fetchContacts(token, orgId, params);
        if (!mountedRef.current) return;

        if (cursor) {
          setContacts((prev) => [...prev, ...result.data]);
        } else {
          setContacts(result.data);
        }
        setHasMore(result.page_info.has_more);
        setNextCursor(result.page_info.next_cursor);
      } catch (err) {
        if (mountedRef.current) {
          setError(err instanceof Error ? err.message : "Failed to load contacts");
        }
      } finally {
        if (mountedRef.current) setIsLoading(false);
      }
    },
    [getToken, orgId, search],
  );

  useEffect(() => {
    mountedRef.current = true;
    if (!tierLoading && hasAccess) {
      void loadContacts();
    }
    return () => {
      mountedRef.current = false;
    };
  }, [tierLoading, hasAccess, loadContacts]);

  if (tierLoading) {
    return (
      <div
        data-testid="contact-list-loading"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        Loading...
      </div>
    );
  }

  if (!hasAccess) {
    return (
      <div
        data-testid="contact-list-denied"
        className="flex flex-col items-center gap-3 py-12 text-center"
      >
        <AlertTriangle className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm font-medium text-foreground">Access Denied</p>
      </div>
    );
  }

  return (
    <div data-testid="contact-list" className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Users className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold text-foreground">Contacts</h2>
          {!isLoading && (
            <span
              className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
              data-testid="contact-count"
            >
              {contacts.length}
            </span>
          )}
        </div>
        <Link
          href="/crm/contacts/new"
          className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          data-testid="contact-create-btn"
        >
          <Plus className="h-4 w-4" />
          New Contact
        </Link>
      </div>

      {error && (
        <div
          data-testid="contact-list-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-3" data-testid="contact-filters">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search contacts..."
            data-testid="contact-search-input"
            className="h-8 w-56 rounded-md border border-border bg-background pl-8 pr-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      {isLoading && contacts.length === 0 ? (
        <div
          data-testid="contact-table-loading"
          className="py-8 text-center text-sm text-muted-foreground"
        >
          Loading contacts...
        </div>
      ) : contacts.length === 0 ? (
        <div
          data-testid="contact-table-empty"
          className="py-8 text-center text-sm text-muted-foreground"
        >
          No contacts found.
        </div>
      ) : (
        <div
          className="overflow-x-auto rounded-lg border border-border"
          data-testid="contact-table"
        >
          <table className="w-full text-sm">
            <thead className="border-b border-border bg-muted/50">
              <tr>
                <th className="px-4 py-2 text-left font-medium text-foreground">Name</th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Email</th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Company</th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Owner</th>
                <th className="px-4 py-2 text-left font-medium text-foreground">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {contacts.map((contact) => {
                const meta = parseLeadData(contact.metadata);
                return (
                  <tr
                    key={contact.id}
                    className="transition-colors hover:bg-accent/50"
                    data-testid={`contact-row-${contact.id}`}
                  >
                    <td className="px-4 py-2">
                      <Link
                        href={`/crm/contacts/${contact.id}`}
                        className="font-medium text-primary hover:underline"
                      >
                        {contact.title}
                      </Link>
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">{meta.contact_email ?? "—"}</td>
                    <td className="px-4 py-2 text-muted-foreground">{meta.company ?? "—"}</td>
                    <td className="px-4 py-2 text-muted-foreground">{meta.assigned_to ?? "—"}</td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {new Date(contact.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {hasMore && (
        <div className="flex justify-center">
          <button
            onClick={() => void loadContacts(nextCursor)}
            disabled={isLoading}
            data-testid="contact-load-more"
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent disabled:opacity-50"
          >
            {isLoading ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
