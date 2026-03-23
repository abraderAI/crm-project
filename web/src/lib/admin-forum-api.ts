import type { PaginatedResponse, Thread } from "./api-types";
import { buildHeaders, buildUrl, parseResponse } from "./api-client";

/**
 * Fetch forum threads for admin management.
 * Uses the global-spaces endpoint with an admin token.
 */
export async function fetchAdminForumThreads(
  token: string,
  params?: { limit?: number; cursor?: string },
): Promise<PaginatedResponse<Thread>> {
  const queryParams: Record<string, string> = {};
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;

  const url = buildUrl("/global-spaces/global-forum/threads", queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/** Toggle pin status on a forum thread. */
export async function toggleForumThreadPin(
  token: string,
  slug: string,
  isPinned: boolean,
): Promise<Thread> {
  const url = buildUrl(`/global-spaces/global-forum/threads/${encodeURIComponent(slug)}`);
  const response = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify({ is_pinned: isPinned }),
  });
  return parseResponse<Thread>(response);
}

/** Toggle hidden status on a forum thread. */
export async function toggleForumThreadHidden(
  token: string,
  slug: string,
  isHidden: boolean,
): Promise<Thread> {
  const url = buildUrl(`/global-spaces/global-forum/threads/${encodeURIComponent(slug)}`);
  const response = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify({ is_hidden: isHidden }),
  });
  return parseResponse<Thread>(response);
}

/** Toggle locked status on a forum thread. */
export async function toggleForumThreadLocked(
  token: string,
  slug: string,
  isLocked: boolean,
): Promise<Thread> {
  const url = buildUrl(`/global-spaces/global-forum/threads/${encodeURIComponent(slug)}`);
  const response = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify({ is_locked: isLocked }),
  });
  return parseResponse<Thread>(response);
}
