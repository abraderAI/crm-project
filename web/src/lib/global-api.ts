import type {
  MessageWithAuthor,
  PaginatedResponse,
  Thread,
  ThreadWithAuthor,
  Upload,
} from "./api-types";
import { buildHeaders, buildUrl, parseResponse, clientMutate } from "./api-client";

/** Global space slugs. */
export const GLOBAL_SPACES = {
  DOCS: "global-docs",
  FORUM: "global-forum",
  SUPPORT: "global-support",
  LEADS: "global-leads",
} as const;

/** Parameters for listing global space threads. */
export interface GlobalThreadParams {
  limit?: number;
  cursor?: string;
  thread_type?: string;
}

/** Parameters for fetching support tickets from global-support space. */
export interface GlobalSupportParams {
  limit?: number;
  cursor?: string;
  /** When true, scopes results to threads authored by the current user. */
  mine?: boolean;
  /** When set, scopes results to threads belonging to the given org. */
  org_id?: string;
}

/** Parameters for fetching leads from global-leads space. */
export interface GlobalLeadsParams {
  limit?: number;
  cursor?: string;
  /** When true, scopes results to threads authored by or assigned to the current user. */
  mine?: boolean;
}

/**
 * Fetch recent public threads from a global space.
 * No auth required for public spaces (global-docs, global-forum).
 */
export async function fetchGlobalThreads(
  spaceSlug: string,
  params?: GlobalThreadParams,
  token?: string | null,
): Promise<PaginatedResponse<Thread>> {
  const queryParams: Record<string, string> = {};
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;
  if (params?.thread_type) queryParams["thread_type"] = params.thread_type;

  const url = buildUrl(`/global-spaces/${spaceSlug}/threads`, queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/**
 * Fetch threads authored or commented on by the current user in global-forum.
 * Requires authentication.
 */
export async function fetchUserForumActivity(
  token: string,
  params?: GlobalThreadParams,
): Promise<PaginatedResponse<Thread>> {
  const queryParams: Record<string, string> = { mine: "true" };
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;

  const url = buildUrl(`/global-spaces/${GLOBAL_SPACES.FORUM}/threads`, queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/**
 * Fetch the current user's support tickets from global-support.
 * Requires authentication. Returns tickets filtered to current user.
 */
export async function fetchUserSupportTickets(
  token: string,
  params?: GlobalThreadParams,
): Promise<PaginatedResponse<Thread>> {
  const queryParams: Record<string, string> = { mine: "true" };
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;

  const url = buildUrl(`/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads`, queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/**
 * Fetch support tickets from global-support space.
 * Pass mine=true to scope to the current user's own tickets.
 * Pass org_id to scope to an org's tickets.
 * Omit both to fetch all tickets (requires tier 4+ authorization).
 * Requires authentication.
 */
export async function fetchGlobalSupportTickets(
  token: string,
  params?: GlobalSupportParams,
): Promise<PaginatedResponse<ThreadWithAuthor>> {
  const queryParams: Record<string, string> = {};
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;
  if (params?.mine) queryParams["mine"] = "true";
  if (params?.org_id) queryParams["org_id"] = params.org_id;

  const url = buildUrl(`/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads`, queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/**
 * Fetch leads from global-leads space.
 * Pass mine=true to scope results to the current user's own and assigned leads.
 * Requires authentication.
 */
export async function fetchGlobalLeads(
  token: string,
  params?: GlobalLeadsParams,
): Promise<PaginatedResponse<Thread>> {
  const queryParams: Record<string, string> = {};
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;
  if (params?.mine) queryParams["mine"] = "true";

  const url = buildUrl(`/global-spaces/${GLOBAL_SPACES.LEADS}/threads`, queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/**
 * Fetch a single thread from a global space by slug.
 * Used for global lead detail view.
 */
export async function fetchGlobalThread(
  spaceSlug: string,
  threadSlug: string,
  token?: string | null,
): Promise<Thread> {
  const url = buildUrl(`/global-spaces/${spaceSlug}/threads/${threadSlug}`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<Thread>(response);
}

/** Values for creating a forum post. */
export interface CreateForumPostValues {
  title: string;
  body?: string;
}

/**
 * Create a new forum thread in global-forum. Tier 2+ only.
 * Backend enforces tier restriction.
 */
export async function createForumThread(
  token: string,
  values: CreateForumPostValues,
): Promise<Thread> {
  return clientMutate<Thread>("POST", `/global-spaces/${GLOBAL_SPACES.FORUM}/threads`, {
    token,
    body: values,
  });
}

/** Fetch messages (replies) for a forum thread by slug. No auth required. */
export async function fetchForumMessages(
  threadSlug: string,
  params?: { limit?: number; cursor?: string },
): Promise<PaginatedResponse<MessageWithAuthor>> {
  const queryParams: Record<string, string> = {};
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;

  const url = buildUrl(
    `/global-spaces/${GLOBAL_SPACES.FORUM}/threads/${encodeURIComponent(threadSlug)}/messages`,
    queryParams,
  );
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<MessageWithAuthor>>(response);
}

/** Create a reply on a forum thread. Requires authentication. */
export async function createForumReply(
  token: string,
  threadSlug: string,
  body: string,
): Promise<MessageWithAuthor> {
  return clientMutate<MessageWithAuthor>(
    "POST",
    `/global-spaces/${GLOBAL_SPACES.FORUM}/threads/${encodeURIComponent(threadSlug)}/messages`,
    { token, body: { body } },
  );
}

/**
 * Fetch a single support ticket from global-support by its slug.
 * Requires authentication.
 */
export async function fetchSupportTicket(token: string, slug: string): Promise<ThreadWithAuthor> {
  const url = buildUrl(
    `/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads/${encodeURIComponent(slug)}`,
  );
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<ThreadWithAuthor>(response);
}

/** Values for updating a support ticket (body, status, and/or assignee). */
export interface UpdateSupportTicketValues {
  body?: string;
  status?: string;
  assigned_to?: string;
}

/**
 * Update a support ticket's body and/or status.
 * Requires authentication.
 */
export async function updateSupportTicket(
  token: string,
  slug: string,
  values: UpdateSupportTicketValues,
): Promise<ThreadWithAuthor> {
  return clientMutate<ThreadWithAuthor>(
    "PATCH",
    `/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads/${encodeURIComponent(slug)}`,
    { token, body: values },
  );
}

/**
 * Fetch attachments for a support ticket thread.
 * Returns an array of Upload records attached to the thread.
 * Requires authentication.
 */
export async function fetchThreadAttachments(token: string, slug: string): Promise<Upload[]> {
  const url = buildUrl(
    `/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads/${encodeURIComponent(slug)}/attachments`,
  );
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<Upload[]>(response);
}

/**
 * Upload a file attachment to a support ticket thread via the dedicated
 * thread-scoped endpoint. The server resolves org_id automatically, so no
 * org context is required from the caller.
 * Requires authentication. The file is posted as multipart/form-data.
 */
export async function uploadThreadAttachment(
  token: string,
  threadSlug: string,
  file: File,
): Promise<Upload> {
  const formData = new FormData();
  formData.append("file", file);

  // Omit Content-Type so the browser sets the multipart/form-data boundary automatically.
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const url = buildUrl(
    `/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads/${encodeURIComponent(threadSlug)}/attachments`,
  );
  const response = await fetch(url, {
    method: "POST",
    headers,
    body: formData,
  });
  return parseResponse<Upload>(response);
}

/**
 * Download an uploaded file attachment by its upload record ID.
 * Fetches the file from the authenticated download endpoint and triggers a
 * browser file download. Requires a valid auth token.
 */
/* v8 ignore start -- browser-only DOM download; cannot be tested in jsdom */
export async function downloadUpload(
  token: string,
  uploadId: string,
  filename: string,
): Promise<void> {
  const url = buildUrl(`/uploads/${encodeURIComponent(uploadId)}/download`);
  const response = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}`, Accept: "*/*" },
  });
  if (!response.ok) {
    throw new Error(`Download failed: ${response.status}`);
  }
  const blob = await response.blob();
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = objectUrl;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(objectUrl);
}
/* v8 ignore stop */

/** Values for creating a support ticket. */
export interface CreateSupportTicketValues {
  title: string;
  body?: string;
  org_id?: string | null;
  /** Assign ticket to a customer by email (DEFT members only). */
  contact_email?: string;
}

/**
 * Create a new support ticket in global-support. Tier 2+ only.
 * If user belongs to an org, org_id is set for scoping.
 * Backend enforces tier restriction.
 */
export async function createSupportTicket(
  token: string,
  values: CreateSupportTicketValues,
): Promise<Thread> {
  return clientMutate<Thread>("POST", `/global-spaces/${GLOBAL_SPACES.SUPPORT}/threads`, {
    token,
    body: values,
  });
}
