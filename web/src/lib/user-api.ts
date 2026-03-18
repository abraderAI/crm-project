import { auth } from "@clerk/nextjs/server";

import type {
  Board,
  DigestSchedule,
  Message,
  Notification,
  NotificationPreference,
  Org,
  PaginatedResponse,
  Revision,
  SearchResult,
  Space,
  Thread,
  UserVoteStatus,
} from "./api-types";
import { serverFetch, serverFetchPaginated } from "./api-client";

/** Get a Clerk JWT token for server-side requests. Throws if unauthenticated. */
async function getToken(): Promise<string> {
  const { getToken: clerkGetToken } = await auth();
  const token = await clerkGetToken();
  if (!token) {
    throw new Error("Unauthenticated");
  }
  return token;
}

/** Fetch paginated list of the user's organizations. */
export async function fetchOrgs(params?: Record<string, string>): Promise<PaginatedResponse<Org>> {
  const token = await getToken();
  return serverFetchPaginated<Org>("/orgs", params, { token });
}

/** Fetch a single organization by slug. */
export async function fetchOrg(slug: string): Promise<Org> {
  const token = await getToken();
  return serverFetch<Org>(`/orgs/${slug}`, { token });
}

/** Fetch paginated spaces within an org. */
export async function fetchSpaces(
  orgSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Space>> {
  const token = await getToken();
  return serverFetchPaginated<Space>(`/orgs/${orgSlug}/spaces`, params, { token });
}

/** Fetch a single space by slug. */
export async function fetchSpace(orgSlug: string, spaceSlug: string): Promise<Space> {
  const token = await getToken();
  return serverFetch<Space>(`/orgs/${orgSlug}/spaces/${spaceSlug}`, { token });
}

/** Fetch paginated boards within a space. */
export async function fetchBoards(
  orgSlug: string,
  spaceSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Board>> {
  const token = await getToken();
  return serverFetchPaginated<Board>(`/orgs/${orgSlug}/spaces/${spaceSlug}/boards`, params, {
    token,
  });
}

/** Fetch a single board by slug. */
export async function fetchBoard(
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
): Promise<Board> {
  const token = await getToken();
  return serverFetch<Board>(`/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}`, { token });
}

/** Fetch paginated threads within a board. */
export async function fetchThreads(
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Thread>> {
  const token = await getToken();
  return serverFetchPaginated<Thread>(
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads`,
    params,
    { token },
  );
}

/** Fetch a single thread by slug. */
export async function fetchThread(
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
): Promise<Thread> {
  const token = await getToken();
  return serverFetch<Thread>(
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}`,
    { token },
  );
}

/** Fetch paginated messages for a thread. */
export async function fetchMessages(
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Message>> {
  const token = await getToken();
  return serverFetchPaginated<Message>(
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/messages`,
    params,
    { token },
  );
}

/** Fetch paginated revisions for an entity.
 * entityType is e.g. "thread", entityId is the entity UUID.
 */
export async function fetchRevisions(
  entityType: string,
  entityId: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Revision>> {
  const token = await getToken();
  return serverFetchPaginated<Revision>(`/revisions/${entityType}/${entityId}`, params, { token });
}

/** Fetch paginated notifications for the current user. */
export async function fetchNotifications(
  params?: Record<string, string>,
): Promise<PaginatedResponse<Notification>> {
  const token = await getToken();
  return serverFetchPaginated<Notification>("/notifications", params, { token });
}

/** Fetch notification preferences for the current user. */
export async function fetchNotificationPreferences(
  params?: Record<string, string>,
): Promise<PaginatedResponse<NotificationPreference>> {
  const token = await getToken();
  return serverFetchPaginated<NotificationPreference>("/notifications/preferences", params, {
    token,
  });
}

/** Fetch the digest schedule for the current user.
 * NOTE: No backend endpoint exists for this. Returns a default schedule.
 */
export async function fetchDigestSchedule(): Promise<DigestSchedule> {
  return {
    id: "",
    user_id: "",
    frequency: "none",
    created_at: "",
    updated_at: "",
  };
}

/** Fetch whether the current user has voted on a thread.
 * NOTE: No GET endpoint exists in the backend (only POST toggle).
 * Returns voted:false as a safe fallback on any error.
 */
export async function fetchUserVote(
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
): Promise<UserVoteStatus> {
  try {
    const token = await getToken();
    return await serverFetch<UserVoteStatus>(
      `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/vote`,
      { token },
    );
  } catch {
    return { voted: false };
  }
}

/** Fetch paginated support tickets for the current user from global-support. */
export async function fetchSupportTickets(
  params?: Record<string, string>,
): Promise<PaginatedResponse<Thread>> {
  const token = await getToken();
  return serverFetchPaginated<Thread>(
    "/global-spaces/global-support/threads",
    { mine: "true", ...params },
    { token },
  );
}

/**
 * Fetch paginated leads from global-leads space.
 * Callers pass mine:"true" to scope results to own + assigned leads (tier 4),
 * or omit it to fetch all leads (tier 5+).
 */
export async function fetchLeads(
  params?: Record<string, string>,
): Promise<PaginatedResponse<Thread>> {
  const token = await getToken();
  return serverFetchPaginated<Thread>("/global-spaces/global-leads/threads", params, { token });
}

/** Fetch a single lead thread from global-leads space by slug. */
export async function fetchGlobalLeadThread(threadSlug: string): Promise<Thread> {
  const token = await getToken();
  return serverFetch<Thread>(`/global-spaces/global-leads/threads/${threadSlug}`, { token });
}

/** Fetch search results. */
export async function fetchSearch(
  query: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<SearchResult>> {
  const token = await getToken();
  return serverFetchPaginated<SearchResult>("/search", { q: query, ...params }, { token });
}
