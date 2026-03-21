"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import type { UserShadow, PaginatedResponse } from "./api-types";
import { buildHeaders, buildUrl, parseResponse } from "./api-client";

/** Resolved user info from the directory. */
export interface ResolvedUser {
  display_name: string;
  org_name: string;
}

/** Return type of useUserDirectory. */
export interface UserDirectory {
  /** Resolve a user ID to display name + org. Returns undefined if not found. */
  resolve: (userId: string) => ResolvedUser | undefined;
  /** Format a user ID for display: "Name (Org)" or "Name" or truncated ID. */
  format: (userId: string) => string;
  /** Whether the directory is still loading. */
  loading: boolean;
}

/**
 * Hook that fetches the admin user list and provides a user ID resolver.
 * Caches the directory for the lifetime of the component tree.
 */
export function useUserDirectory(): UserDirectory {
  const { getToken } = useAuth();
  const [directory, setDirectory] = useState<Map<string, ResolvedUser>>(new Map());
  const [loading, setLoading] = useState(true);
  const fetchedRef = useRef(false);

  useEffect(() => {
    if (fetchedRef.current) return;
    fetchedRef.current = true;

    void (async () => {
      try {
        const token = await getToken();
        if (!token) {
          setLoading(false);
          return;
        }
        const url = buildUrl("/admin/users", { limit: "500" });
        const response = await fetch(url, {
          method: "GET",
          headers: buildHeaders(token),
          cache: "no-store",
        });
        const result = await parseResponse<PaginatedResponse<UserShadow>>(response);

        const map = new Map<string, ResolvedUser>();
        for (const user of result.data) {
          map.set(user.clerk_user_id, {
            display_name: user.display_name || user.email,
            org_name: user.primary_org_name ?? "",
          });
        }
        setDirectory(map);
      } catch {
        // Silently fall back to empty directory.
      } finally {
        setLoading(false);
      }
    })();
  }, [getToken]);

  const resolve = useCallback(
    (userId: string): ResolvedUser | undefined => directory.get(userId),
    [directory],
  );

  const format = useCallback(
    (userId: string): string => {
      const user = directory.get(userId);
      if (!user) return userId.length > 12 ? `${userId.slice(0, 12)}…` : userId;
      if (user.org_name) return `${user.display_name} (${user.org_name})`;
      return user.display_name;
    },
    [directory],
  );

  return { resolve, format, loading };
}
