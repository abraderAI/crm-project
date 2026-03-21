import type {
  AdminOrgDetail,
  Org,
  OrgMembership,
  PaginatedResponse,
  Role,
  Space,
  Thread,
} from "./api-types";
import { buildHeaders, buildUrl, clientMutate, parseResponse } from "./api-client";

/** Org overview data for the org overview widget. */
export interface OrgOverview {
  name: string;
  slug: string;
  member_count: number;
  plan_status: string;
  billing_tier: string;
}

/** Org support ticket stats for the dashboard widget. */
export interface OrgSupportStats {
  open: number;
  pending: number;
  resolved: number;
  total: number;
}

/** Space-level RBAC override. */
export interface SpaceRoleOverride {
  space_id: string;
  space_name: string;
  role: Role | null;
}

/**
 * Fetch org overview information (name, member count, plan status).
 * Returns stub data when real aggregate endpoint is unavailable.
 */
export async function fetchOrgOverview(token: string, orgId: string): Promise<OrgOverview> {
  const url = buildUrl(`/orgs/${orgId}`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const org = await parseResponse<Org>(response);

  return {
    name: org.name,
    slug: org.slug,
    member_count: org.spaces?.length ?? 0,
    plan_status: org.payment_status ?? "active",
    billing_tier: org.billing_tier ?? "pro",
  };
}

/**
 * Fetch support tickets filtered by org_id.
 * Calls the global-support space threads endpoint with org_id filter.
 */
export async function fetchOrgSupportTickets(
  token: string,
  orgId: string,
  params?: { limit?: number; cursor?: string },
): Promise<PaginatedResponse<Thread>> {
  const queryParams: Record<string, string> = { org_id: orgId };
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;

  const url = buildUrl("/global-spaces/global-support/threads", queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Thread>>(response);
}

/**
 * Fetch org support ticket statistics.
 * Aggregates from org-filtered support tickets. Returns stubs for now.
 */
export async function fetchOrgSupportStats(token: string, orgId: string): Promise<OrgSupportStats> {
  const url = buildUrl("/global-spaces/global-support/threads", {
    org_id: orgId,
    limit: "100",
  });
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const result = await parseResponse<PaginatedResponse<Thread>>(response);

  let open = 0;
  let pending = 0;
  let resolved = 0;
  for (const ticket of result.data) {
    const status = ticket.status ?? "open";
    if (status === "open") open++;
    else if (status === "pending") pending++;
    else if (status === "resolved" || status === "closed") resolved++;
    else open++;
  }

  return { open, pending, resolved, total: result.data.length };
}

/**
 * Fetch org members with role information.
 */
export async function fetchOrgMembers(
  token: string,
  orgId: string,
  params?: { limit?: number; cursor?: string },
): Promise<PaginatedResponse<OrgMembership>> {
  const queryParams: Record<string, string> = {};
  if (params?.limit) queryParams["limit"] = String(params.limit);
  if (params?.cursor) queryParams["cursor"] = params.cursor;

  const url = buildUrl(`/orgs/${orgId}/members`, queryParams);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<OrgMembership>>(response);
}

/**
 * Update a member's role within an org.
 * Cannot elevate above the caller's own role (enforced server-side).
 */
export async function updateMemberRole(
  token: string,
  orgId: string,
  memberId: string,
  role: Role,
): Promise<OrgMembership> {
  return clientMutate<OrgMembership>("PATCH", `/orgs/${orgId}/members/${memberId}`, {
    token,
    body: { role },
  });
}

/**
 * Remove a member from an org.
 */
export async function removeMember(token: string, orgId: string, memberId: string): Promise<void> {
  await clientMutate<void>("DELETE", `/orgs/${orgId}/members/${memberId}`, { token });
}

/**
 * Fetch spaces for an org (used by RBAC editor to list available spaces).
 */
export async function fetchOrgSpaces(
  token: string,
  orgId: string,
): Promise<PaginatedResponse<Space>> {
  const url = buildUrl(`/orgs/${orgId}/spaces`);
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  return parseResponse<PaginatedResponse<Space>>(response);
}

/**
 * Fetch all orgs from the admin endpoint (client-side).
 * Used by the "Add to Org" dialog to list available orgs.
 */
export async function fetchOrgsClient(token: string): Promise<AdminOrgDetail[]> {
  const url = buildUrl("/admin/orgs");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const result = await parseResponse<PaginatedResponse<AdminOrgDetail>>(response);
  return result.data;
}

/**
 * Create a new org (client-side).
 * Returns the newly created org detail.
 */
export async function createOrgClient(
  token: string,
  name: string,
  description?: string,
): Promise<AdminOrgDetail> {
  return clientMutate<AdminOrgDetail>("POST", "/orgs", {
    token,
    body: { name, description: description || undefined },
  });
}

/**
 * Set a space-level role override for a member.
 * Pass role=null to remove the override.
 */
export async function updateSpaceRoleOverride(
  token: string,
  orgId: string,
  memberId: string,
  spaceId: string,
  role: Role | null,
): Promise<void> {
  await clientMutate<void>("PUT", `/orgs/${orgId}/members/${memberId}/space-roles/${spaceId}`, {
    token,
    body: { role },
  });
}
