import type { DeftMember, SupportEntry, SupportEntryType } from "./api-types";
import { buildHeaders, buildUrl, parseResponse, clientMutate } from "./api-client";

/**
 * Fetch all visible entries for a support ticket.
 * The server filters entries based on the caller's DEFT membership.
 * Requires authentication.
 */
export async function fetchTicketEntries(
  token: string,
  ticketSlug: string,
): Promise<SupportEntry[]> {
  const url = buildUrl(`/support/tickets/${encodeURIComponent(ticketSlug)}/entries`);
  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const data = await parseResponse<{ data: SupportEntry[] }>(res);
  return data.data;
}

/** Values accepted when creating a new ticket entry. */
export interface CreateEntryValues {
  type: SupportEntryType;
  body: string;
  is_deft_only?: boolean;
}

/**
 * Create a new entry on a support ticket.
 * Non-DEFT users may only use type "customer".
 * Requires authentication.
 */
export async function createTicketEntry(
  token: string,
  ticketSlug: string,
  values: CreateEntryValues,
): Promise<SupportEntry> {
  return clientMutate<SupportEntry>(
    "POST",
    `/support/tickets/${encodeURIComponent(ticketSlug)}/entries`,
    { token, body: values },
  );
}

/**
 * Update the body of a mutable (draft) entry.
 * Returns 403 if the entry is immutable.
 * Requires authentication.
 */
export async function updateTicketEntry(
  token: string,
  ticketSlug: string,
  entryId: string,
  body: string,
): Promise<SupportEntry> {
  return clientMutate<SupportEntry>(
    "PATCH",
    `/support/tickets/${encodeURIComponent(ticketSlug)}/entries/${encodeURIComponent(entryId)}`,
    { token, body: { body } },
  );
}

/**
 * Publish a draft entry, making it visible to the customer as an agent_reply.
 * Only callable by DEFT members.
 * Requires authentication.
 */
export async function publishTicketEntry(
  token: string,
  ticketSlug: string,
  entryId: string,
): Promise<SupportEntry> {
  return clientMutate<SupportEntry>(
    "POST",
    `/support/tickets/${encodeURIComponent(ticketSlug)}/entries/${encodeURIComponent(entryId)}/publish`,
    { token },
  );
}

/**
 * Toggle the DEFT-only visibility flag on an entry.
 * Only callable by DEFT members.
 * Requires authentication.
 */
export async function setEntryDeftVisibility(
  token: string,
  ticketSlug: string,
  entryId: string,
  isDeftOnly: boolean,
): Promise<SupportEntry> {
  return clientMutate<SupportEntry>(
    "PATCH",
    `/support/tickets/${encodeURIComponent(ticketSlug)}/entries/${encodeURIComponent(entryId)}/deft-visibility`,
    { token, body: { is_deft_only: isDeftOnly } },
  );
}

/**
 * Fetch all active DEFT org members for the assignee picker.
 * Only callable by DEFT members (tier 4+).
 * Requires authentication.
 */
export async function fetchDeftMembers(token: string): Promise<DeftMember[]> {
  const url = buildUrl("/support/deft-members");
  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const data = await parseResponse<{ data: DeftMember[] }>(res);
  return data.data;
}

/**
 * Update the notification detail level for a support ticket.
 * level: "full" — include agent reply body in notification emails.
 * level: "privacy" — send link-only notification emails.
 * Requires authentication.
 */
export async function setTicketNotificationPref(
  token: string,
  ticketSlug: string,
  level: "full" | "privacy",
): Promise<void> {
  const url = buildUrl(`/support/tickets/${encodeURIComponent(ticketSlug)}/notifications`);
  const res = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify({ notification_detail_level: level }),
  });
  if (!res.ok && res.status !== 204) {
    await parseResponse<void>(res);
  }
}
