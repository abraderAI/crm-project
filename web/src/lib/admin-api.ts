import { auth } from "@clerk/nextjs/server";

import type {
  AuditEntry,
  BillingInfo,
  ChannelConfig,
  ChannelHealth,
  ChannelType,
  DeadLetterEvent,
  EffectivePolicy,
  FeatureFlag,
  Flag,
  OrgMembership,
  PaginatedResponse,
  PlatformAdmin,
  PlatformStats,
  SecurityLogEntry,
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

/** Fetch all system settings as a key-value map. */
export async function fetchAdminSettings(): Promise<Record<string, unknown>> {
  const token = await getToken();
  return serverFetch<Record<string, unknown>>("/admin/settings", { token });
}

/** Fetch paginated list of users. */
export async function fetchAdminUsers(
  params?: Record<string, string>,
): Promise<PaginatedResponse<UserShadow>> {
  const token = await getToken();
  return serverFetchPaginated<UserShadow>("/admin/users", params, { token });
}

/** Fetch a single user by ID. */
export async function fetchAdminUser(userId: string): Promise<UserShadow> {
  const token = await getToken();
  return serverFetch<UserShadow>(`/admin/users/${userId}`, { token });
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

/** Fetch paginated moderation flags. */
export async function fetchFlags(
  params?: Record<string, string>,
): Promise<PaginatedResponse<Flag>> {
  const token = await getToken();
  return serverFetchPaginated<Flag>("/admin/flags", params, { token });
}

/** Fetch paginated org memberships. */
export async function fetchMemberships(
  params?: Record<string, string>,
): Promise<PaginatedResponse<OrgMembership>> {
  const token = await getToken();
  return serverFetchPaginated<OrgMembership>("/admin/memberships", params, { token });
}

/** Fetch paginated recent login events. */
export async function fetchRecentLogins(
  params?: Record<string, string>,
): Promise<PaginatedResponse<SecurityLogEntry>> {
  const token = await getToken();
  return serverFetchPaginated<SecurityLogEntry>("/admin/security/recent-logins", params, {
    token,
  });
}

/** Fetch paginated failed authentication events. */
export async function fetchFailedAuths(
  params?: Record<string, string>,
): Promise<PaginatedResponse<SecurityLogEntry>> {
  const token = await getToken();
  return serverFetchPaginated<SecurityLogEntry>("/admin/security/failed-auths", params, {
    token,
  });
}

/** Fetch effective RBAC policy. */
export async function fetchRBACPolicy(): Promise<EffectivePolicy> {
  const token = await getToken();
  return serverFetch<EffectivePolicy>("/admin/rbac-policy", { token });
}

// --- IO Channel API functions ---

/** Fetch channel configuration for a specific channel type. */
export async function fetchChannelConfig(
  org: string,
  channelType: ChannelType,
): Promise<ChannelConfig> {
  const token = await getToken();
  return serverFetch<ChannelConfig>(`/orgs/${org}/channels/${channelType}`, { token });
}

/** Update channel configuration. */
export async function putChannelConfig(
  org: string,
  channelType: ChannelType,
  body: { settings: string; enabled: boolean },
): Promise<ChannelConfig> {
  const token = await getToken();
  const url = `/orgs/${org}/channels/${channelType}`;
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/v1${url}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
      cache: "no-store",
    },
  );
  if (!response.ok) {
    throw new Error(`Failed to update channel config: ${response.status}`);
  }
  return (await response.json()) as ChannelConfig;
}

/** Fetch channel health status. */
export async function fetchChannelHealth(
  org: string,
  channelType: ChannelType,
): Promise<ChannelHealth> {
  const token = await getToken();
  return serverFetch<ChannelHealth>(`/orgs/${org}/channels/${channelType}/health`, { token });
}

/** Fetch dead-letter queue events for a channel. */
export async function fetchDLQEvents(
  org: string,
  channelType: ChannelType,
  params?: Record<string, string>,
): Promise<PaginatedResponse<DeadLetterEvent>> {
  const token = await getToken();
  return serverFetchPaginated<DeadLetterEvent>(`/orgs/${org}/channels/${channelType}/dlq`, params, {
    token,
  });
}

/** Retry a dead-letter queue event. */
export async function retryDLQEvent(
  org: string,
  channelType: ChannelType,
  eventId: string,
): Promise<void> {
  const token = await getToken();
  const url = `/orgs/${org}/channels/${channelType}/dlq/${eventId}/retry`;
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/v1${url}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    },
  );
  if (!response.ok) {
    throw new Error(`Failed to retry DLQ event: ${response.status}`);
  }
}

/** Dismiss a dead-letter queue event. */
export async function dismissDLQEvent(
  org: string,
  channelType: ChannelType,
  eventId: string,
): Promise<void> {
  const token = await getToken();
  const url = `/orgs/${org}/channels/${channelType}/dlq/${eventId}/dismiss`;
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/v1${url}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    },
  );
  if (!response.ok) {
    throw new Error(`Failed to dismiss DLQ event: ${response.status}`);
  }
}
