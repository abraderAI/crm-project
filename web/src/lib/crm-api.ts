import type { PaginatedResponse, Thread, Message } from "./api-types";
import { buildHeaders, buildUrl, parseResponse } from "./api-client";

// ---------------------------------------------------------------------------
// Company API
// ---------------------------------------------------------------------------

/** Fetch paginated companies for an org. */
export async function fetchCompanies(
  token: string,
  orgSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Thread>> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/companies`, params);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/** Fetch a single company by ID. */
export async function fetchCompany(
  token: string,
  orgSlug: string,
  companyId: string,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/companies/${companyId}`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<Thread>(response);
}

/** Create a new company. */
export async function createCompany(
  token: string,
  orgSlug: string,
  data: Record<string, unknown>,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/companies`);
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Update an existing company. */
export async function updateCompany(
  token: string,
  orgSlug: string,
  companyId: string,
  data: Record<string, unknown>,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/companies/${companyId}`);
  const response = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Check for duplicate company name. */
export async function checkDuplicateCompany(
  token: string,
  orgSlug: string,
  name: string,
): Promise<Thread[]> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/companies/check-duplicate`, { name });
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const result = await parseResponse<{ data: Thread[] }>(response);
  return result.data;
}

// ---------------------------------------------------------------------------
// Contact API
// ---------------------------------------------------------------------------

/** Fetch paginated contacts for an org. */
export async function fetchContacts(
  token: string,
  orgSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Thread>> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/contacts`, params);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/** Fetch a single contact by ID. */
export async function fetchContact(
  token: string,
  orgSlug: string,
  contactId: string,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/contacts/${contactId}`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<Thread>(response);
}

/** Create a new contact. */
export async function createContact(
  token: string,
  orgSlug: string,
  data: Record<string, unknown>,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/contacts`);
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Update an existing contact. */
export async function updateContact(
  token: string,
  orgSlug: string,
  contactId: string,
  data: Record<string, unknown>,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/contacts/${contactId}`);
  const response = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Check for duplicate contact email. */
export async function checkDuplicateContact(
  token: string,
  orgSlug: string,
  email: string,
): Promise<Thread[]> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/contacts/check-duplicate`, { email });
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const result = await parseResponse<{ data: Thread[] }>(response);
  return result.data;
}

// ---------------------------------------------------------------------------
// Opportunity API
// ---------------------------------------------------------------------------

/** Fetch paginated opportunities for an org. */
export async function fetchOpportunities(
  token: string,
  orgSlug: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Thread>> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/opportunities`, params);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/** Fetch a single opportunity by ID. */
export async function fetchOpportunity(
  token: string,
  orgSlug: string,
  opportunityId: string,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/opportunities/${opportunityId}`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<Thread>(response);
}

/** Create a new opportunity. */
export async function createOpportunity(
  token: string,
  orgSlug: string,
  data: Record<string, unknown>,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/opportunities`);
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Update an existing opportunity. */
export async function updateOpportunity(
  token: string,
  orgSlug: string,
  opportunityId: string,
  data: Record<string, unknown>,
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/opportunities/${opportunityId}`);
  const response = await fetch(url, {
    method: "PATCH",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Transition an opportunity to a new stage. */
export async function transitionOpportunity(
  token: string,
  orgSlug: string,
  opportunityId: string,
  data: { stage: string; comment?: string; reason?: string; close_reason?: string },
): Promise<Thread> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/opportunities/${opportunityId}/transition`);
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
    body: JSON.stringify(data),
  });
  return parseResponse<Thread>(response);
}

/** Reassign an entity to a new owner. */
export async function reassignEntity(
  token: string,
  orgSlug: string,
  entityType: string,
  entityId: string,
  newOwnerId: string,
): Promise<void> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/${entityType}/${entityId}/reassign`);
  const response = await fetch(url, {
    method: "POST",
    headers: buildHeaders(token),
    body: JSON.stringify({ new_owner_id: newOwnerId }),
  });
  await parseResponse<void>(response);
}

// ---------------------------------------------------------------------------
// CRM Links & Messages
// ---------------------------------------------------------------------------

/** Fetch messages (activity timeline) for a CRM entity thread. */
export async function fetchEntityMessages(
  token: string,
  orgSlug: string,
  entityType: string,
  entityId: string,
  params?: Record<string, string>,
): Promise<PaginatedResponse<Message>> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/${entityType}/${entityId}/messages`, params);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Message>>(response);
}

/** Fetch linked contacts for a company. */
export async function fetchLinkedContacts(
  token: string,
  orgSlug: string,
  companyId: string,
): Promise<PaginatedResponse<Thread>> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/companies/${companyId}/contacts`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/** Fetch linked opportunities for a company or contact. */
export async function fetchLinkedOpportunities(
  token: string,
  orgSlug: string,
  entityType: string,
  entityId: string,
): Promise<PaginatedResponse<Thread>> {
  const url = buildUrl(`/orgs/${orgSlug}/crm/${entityType}/${entityId}/opportunities`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}
