import type {
  Board,
  DigestFrequency,
  Flag,
  Message,
  MessageType,
  Org,
  OrgMembership,
  PaginatedResponse,
  Revision,
  Role,
  Space,
  Thread,
  Upload,
  Vote,
  WebhookSubscription,
} from "./api-types";
import type { PreferenceSetting } from "@/components/realtime/notification-preferences";
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

// --- Notification Preferences ---

/** Save notification preference toggles. */
export async function saveNotificationPreferences(
  token: string,
  preferences: PreferenceSetting[],
): Promise<void> {
  await clientMutate<void>("PUT", "/notifications/preferences", {
    token,
    body: { preferences },
  });
}

/** Save digest schedule frequency. */
export async function saveDigestSchedule(token: string, frequency: DigestFrequency): Promise<void> {
  await clientMutate<void>("PUT", "/notifications/digest", {
    token,
    body: { frequency },
  });
}

// --- Webhook mutations ---

/** Create a new webhook subscription. */
export async function createWebhook(
  token: string,
  url: string,
  eventFilter: string,
): Promise<WebhookSubscription> {
  return clientMutate<WebhookSubscription>("POST", "/admin/webhooks", {
    token,
    body: { url, event_filter: eventFilter },
  });
}

/** Delete a webhook subscription by ID. */
export async function deleteWebhook(token: string, subscriptionId: string): Promise<void> {
  await clientMutate<void>("DELETE", `/admin/webhooks/${subscriptionId}`, { token });
}

/** Toggle a webhook subscription active/inactive. */
export async function toggleWebhook(
  token: string,
  subscriptionId: string,
): Promise<WebhookSubscription> {
  return clientMutate<WebhookSubscription>("PATCH", `/admin/webhooks/${subscriptionId}/toggle`, {
    token,
  });
}

/** Replay a webhook delivery. */
export async function replayWebhookDelivery(token: string, deliveryId: string): Promise<void> {
  await clientMutate<void>("POST", `/admin/webhook-deliveries/${deliveryId}/replay`, { token });
}

// --- Membership mutations ---

/** Add a membership. */
export async function addMembership(
  token: string,
  userId: string,
  role: Role,
): Promise<OrgMembership> {
  return clientMutate<OrgMembership>("POST", "/admin/memberships", {
    token,
    body: { user_id: userId, role },
  });
}

/** Change a membership's role. */
export async function changeMembershipRole(
  token: string,
  membershipId: string,
  newRole: Role,
): Promise<OrgMembership> {
  return clientMutate<OrgMembership>("PATCH", `/admin/memberships/${membershipId}`, {
    token,
    body: { role: newRole },
  });
}

/** Remove a membership. */
export async function removeMembership(token: string, membershipId: string): Promise<void> {
  await clientMutate<void>("DELETE", `/admin/memberships/${membershipId}`, { token });
}

// --- Vote mutations ---

/** Toggle the current user's vote on a thread. */
export async function toggleVote(
  token: string,
  orgSlug: string,
  spaceSlug: string,
  boardSlug: string,
  threadSlug: string,
): Promise<Vote> {
  return clientMutate<Vote>(
    "POST",
    `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${boardSlug}/threads/${threadSlug}/vote`,
    { token },
  );
}

// --- Flag mutations ---

/** Create a content flag for moderation review. */
export async function createFlag(token: string, threadId: string, reason: string): Promise<Flag> {
  return clientMutate<Flag>("POST", "/admin/flags", {
    token,
    body: { thread_id: threadId, reason },
  });
}

/** Resolve a pending flag with an optional note. */
export async function resolveFlag(token: string, flagId: string, note: string): Promise<Flag> {
  return clientMutate<Flag>("PATCH", `/admin/flags/${flagId}/resolve`, {
    token,
    body: { resolution_note: note },
  });
}

/** Dismiss a pending flag. */
export async function dismissFlag(token: string, flagId: string): Promise<Flag> {
  return clientMutate<Flag>("PATCH", `/admin/flags/${flagId}/dismiss`, {
    token,
  });
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
