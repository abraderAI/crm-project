import { auth } from "@clerk/nextjs/server";

import type {
  AuditEntry,
  BillingInfo,
  FeatureFlag,
  OrgMembership,
  PaginatedResponse,
  PlatformAdmin,
  PlatformStats,
  UserShadow,
  WebhookDelivery,
  WebhookSubscription,
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

/** Fetch platform-wide statistics. */
export async function fetchAdminStats(): Promise<PlatformStats> {
  const token = await getToken();
  return serverFetch<PlatformStats>("/admin/stats", { token });
}

/** Fetch paginated list of users. */
export async function fetchAdminUsers(
  params?: Record<string, string>,
): Promise<PaginatedResponse<UserShadow>> {
  const token = await getToken();
  return serverFetchPaginated<UserShadow>("/admin/users", params, { token });
}

/** Fetch list of platform admins. */
export async function fetchPlatformAdmins(): Promise<PlatformAdmin[]> {
  const token = await getToken();
  const res = await serverFetch<{ data: PlatformAdmin[] }>("/admin/platform-admins", { token });
  return res.data;
}

/** Fetch paginated audit log. */
export async function fetchAuditLog(
  params?: Record<string, string>,
): Promise<PaginatedResponse<AuditEntry>> {
  const token = await getToken();
  return serverFetchPaginated<AuditEntry>("/admin/audit-log", params, { token });
}

/** Fetch all feature flags. */
export async function fetchFeatureFlags(): Promise<FeatureFlag[]> {
  const token = await getToken();
  const res = await serverFetch<{ data: FeatureFlag[] }>("/admin/feature-flags", { token });
  return res.data;
}

/** Fetch billing information for the current org. */
export async function fetchBillingInfo(): Promise<BillingInfo> {
  const token = await getToken();
  return serverFetch<BillingInfo>("/admin/billing", { token });
}

/** Fetch paginated webhook subscriptions. */
export async function fetchWebhookSubscriptions(
  params?: Record<string, string>,
): Promise<PaginatedResponse<WebhookSubscription>> {
  const token = await getToken();
  return serverFetchPaginated<WebhookSubscription>("/admin/webhooks", params, { token });
}

/** Fetch paginated webhook delivery log. */
export async function fetchWebhookDeliveries(
  params?: Record<string, string>,
): Promise<PaginatedResponse<WebhookDelivery>> {
  const token = await getToken();
  return serverFetchPaginated<WebhookDelivery>("/admin/webhook-deliveries", params, { token });
}

/** Fetch paginated org memberships. */
export async function fetchMemberships(
  params?: Record<string, string>,
): Promise<PaginatedResponse<OrgMembership>> {
  const token = await getToken();
  return serverFetchPaginated<OrgMembership>("/admin/memberships", params, { token });
}
