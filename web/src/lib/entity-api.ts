import type {
  Board,
  Message,
  MessageType,
  Org,
  PaginatedResponse,
  Revision,
  Space,
  Thread,
  Upload,
} from "./api-types";
import type { EntityFormValues } from "@/components/entities/entity-form";
import { clientMutate, buildHeaders, buildUrl, parseResponse } from "./api-client";

// --- Org mutations ---

/** Create a new organization. */
export async function createOrg(token: string, values: EntityFormValues): Promise<Org> {
  return clientMutate<Org>("POST", "/orgs", { token, body: values });
}

/** Update an existing organization by slug. */
export async function updateOrg(
  token: string,
  slug: string,
  values: EntityFormValues,
): Promise<Org> {
  return clientMutate<Org>("PATCH", `/orgs/${slug}`, { token, body: values });
}

/** Soft-delete an organization by slug. */
export async function deleteOrg(token: string, slug: string): Promise<void> {
  await clientMutate<void>("DELETE", `/orgs/${slug}`, { token });
}

// --- Space mutations ---

/** Create a new space within an org. */
export async function createSpace(
  token: string,
  orgSlug: string,
  values: EntityFormValues,
): Promise<Space> {
  return clientMutate<Space>("POST", `/orgs/${orgSlug}/spaces`, { token, body: values });
}

/** Update an existing space by slug. */
export async function updateSpace(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  values: EntityFormValues,
): Promise<Space> {
  return clientMutate<Space>("PATCH", `/orgs/${orgSlug}/spaces/${spaceSlug}`, {
    token,
    body: values,
  });
}

/** Soft-delete a space by slug. */
export async function deleteSpace(
  token: string,
  orgSlug: string,
  spaceSlug: string,
): Promise<void> {
  await clientMutate<void>("DELETE", `/orgs/${orgSlug}/spaces/${spaceSlug}`, { token });
}

// --- Board mutations ---

/** Create a new board within a space. */
export async function createBoard(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  values: EntityFormValues,
): Promise<Board> {
  return clientMutate<Board>("POST", `/orgs/${orgSlug}/spaces/${spaceSlug}/boards`, {
    token,
    body: values,
  });
}

/** Update an existing board by slug. */
export async function updateBoard(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  values: EntityFormValues,
): Promise<Board> {
  return clientMutate<Board>("PATCH", `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}`, {
    token,
    body: values,
  });
}

/** Soft-delete a board by slug. */
export async function deleteBoard(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
): Promise<void> {
  await clientMutate<void>("DELETE", `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}`, {
    token,
  });
}

// --- Thread mutations ---

/** Values accepted when creating a thread. */
export interface CreateThreadValues {
  title: string;
  body?: string;
  metadata?: string;
}

/** Create a new thread within a board. */
export async function createThread(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  values: CreateThreadValues,
): Promise<Thread> {
  return clientMutate<Thread>(
    "POST",
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads`,
    { token, body: values },
  );
}

// --- Message mutations ---

/** Values accepted when creating a message. */
export interface CreateMessageValues {
  body: string;
  type?: MessageType;
}

/** Fetch revisions for a thread (client-side). */
export async function fetchThreadRevisions(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
): Promise<PaginatedResponse<Revision>> {
  const url = buildUrl(
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/revisions`,
  );
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
  });
  return parseResponse<PaginatedResponse<Revision>>(response);
}

/** Fetch uploads for a thread (client-side). */
export async function fetchThreadUploads(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
): Promise<PaginatedResponse<Upload>> {
  const url = buildUrl(
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/uploads`,
  );
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
  });
  return parseResponse<PaginatedResponse<Upload>>(response);
}

/** Upload a file to a thread. */
export async function uploadFile(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
  file: File,
): Promise<Upload> {
  const url = buildUrl(
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/uploads`,
  );
  const formData = new FormData();
  formData.append("file", file);
  const response = await fetch(url, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: formData,
  });
  return parseResponse<Upload>(response);
}

/** Delete an upload by ID. */
export async function deleteUpload(token: string, uploadId: string): Promise<void> {
  await clientMutate<void>("DELETE", `/uploads/${uploadId}`, { token });
}

/** Create a new message within a thread. Defaults type to "comment". */
export async function createMessage(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
  values: CreateMessageValues,
): Promise<Message> {
  return clientMutate<Message>(
    "POST",
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/messages`,
    { token, body: { ...values, type: values.type ?? "comment" } },
  );
}
