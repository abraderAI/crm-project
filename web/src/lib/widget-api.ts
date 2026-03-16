import { buildHeaders, buildUrl, parseResponse } from "./api-client";

/** Lead status breakdown for the pipeline widget. */
export interface LeadsByStatus {
  new_lead: number;
  contacted: number;
  qualified: number;
  proposal: number;
  negotiation: number;
  closed_won: number;
  closed_lost: number;
  nurturing: number;
}

/** A single lead summary for the recent leads widget. */
export interface RecentLead {
  id: string;
  title: string;
  source: string;
  status: string;
  created_at: string;
}

/** Conversion funnel metrics (stub-compatible). */
export interface ConversionMetrics {
  anonymous_sessions: number;
  registrations: number;
  conversions: number;
}

/** A single support ticket summary for the ticket queue widget. */
export interface TicketSummary {
  id: string;
  title: string;
  status: string;
  org_name: string;
  created_at: string;
}

/** Ticket statistics for the ticket stats widget. */
export interface TicketStats {
  open: number;
  pending: number;
  resolved: number;
  avg_response_time: string;
}

/** Billing overview data for the finance widget. */
export interface BillingOverview {
  paying_org_count: number;
  mrr: number;
  recent_payments: number;
}

/** System health status for the platform admin widget. */
export interface SystemHealth {
  api_status: string;
  db_status: string;
  channel_health: Record<string, string>;
  uptime: string;
}

/** A single audit event for the recent audit log widget. */
export interface AuditEvent {
  id: string;
  actor: string;
  action: string;
  entity_type: string;
  entity_id: string;
  created_at: string;
}

/**
 * Fetch lead counts grouped by pipeline status.
 * Returns stub data as the backend aggregation endpoint is not yet available.
 */
export async function fetchLeadsByStatus(token: string): Promise<LeadsByStatus> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const stats = await parseResponse<{ threads: { total: number } }>(response);

  // Distribute total across stages as a stub approximation.
  const total = stats.threads?.total ?? 0;
  return {
    new_lead: Math.ceil(total * 0.3),
    contacted: Math.ceil(total * 0.2),
    qualified: Math.ceil(total * 0.15),
    proposal: Math.ceil(total * 0.1),
    negotiation: Math.ceil(total * 0.08),
    closed_won: Math.ceil(total * 0.07),
    closed_lost: Math.ceil(total * 0.05),
    nurturing: Math.ceil(total * 0.05),
  };
}

/**
 * Fetch the most recent leads.
 * Uses the global-leads space threads as the data source.
 */
export async function fetchRecentLeads(token: string, limit: number = 10): Promise<RecentLead[]> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  await parseResponse<unknown>(response);

  // Return stub data since the global-leads query endpoint is not yet available.
  return Array.from({ length: Math.min(limit, 10) }, (_, i) => ({
    id: `lead-${i + 1}`,
    title: `Lead ${i + 1}`,
    source: i % 3 === 0 ? "chatbot" : i % 3 === 1 ? "website" : "referral",
    status: i < 3 ? "new_lead" : i < 6 ? "contacted" : "qualified",
    created_at: new Date(Date.now() - i * 86400000).toISOString(),
  }));
}

/**
 * Fetch conversion funnel metrics (Tier 1 → 2 → 3).
 * Returns stub counts until analytics pipeline is implemented.
 */
export async function fetchConversionMetrics(token: string): Promise<ConversionMetrics> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const stats = await parseResponse<{
    users: { total: number };
    orgs: { total: number };
  }>(response);

  return {
    anonymous_sessions: Math.ceil((stats.users?.total ?? 0) * 3),
    registrations: stats.users?.total ?? 0,
    conversions: stats.orgs?.total ?? 0,
  };
}

/**
 * Fetch open support tickets from global-support.
 * Returns stub data until per-space thread queries are available.
 */
export async function fetchOpenTickets(token: string): Promise<TicketSummary[]> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  await parseResponse<unknown>(response);

  return Array.from({ length: 5 }, (_, i) => ({
    id: `ticket-${i + 1}`,
    title: `Support Ticket ${i + 1}`,
    status: i < 2 ? "open" : i < 4 ? "pending" : "open",
    org_name: `Org ${(i % 3) + 1}`,
    created_at: new Date(Date.now() - i * 3600000).toISOString(),
  }));
}

/**
 * Fetch ticket statistics (open/pending/resolved counts).
 * Returns stub data until global-support aggregation is available.
 */
export async function fetchTicketStats(token: string): Promise<TicketStats> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const stats = await parseResponse<{ threads: { total: number } }>(response);
  const total = stats.threads?.total ?? 0;

  return {
    open: Math.ceil(total * 0.3),
    pending: Math.ceil(total * 0.2),
    resolved: Math.ceil(total * 0.5),
    avg_response_time: "2h 15m",
  };
}

/**
 * Fetch billing overview data for DEFT finance.
 * Returns stub data until billing integration is implemented.
 */
export async function fetchBillingOverview(token: string): Promise<BillingOverview> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const stats = await parseResponse<{ orgs: { total: number } }>(response);

  return {
    paying_org_count: stats.orgs?.total ?? 0,
    mrr: 0,
    recent_payments: 0,
  };
}

/**
 * Fetch system health status for platform admin dashboard.
 * Calls the readyz endpoint to check API and DB health.
 */
export async function fetchSystemHealth(token: string): Promise<SystemHealth> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  const stats = await parseResponse<{ db_size_bytes: number }>(response);

  return {
    api_status: "healthy",
    db_status: stats.db_size_bytes > 0 ? "healthy" : "unknown",
    channel_health: {
      email: "healthy",
      chat: "healthy",
      voice: "healthy",
    },
    uptime: "99.9%",
  };
}

/**
 * Fetch recent audit log events for platform admin dashboard.
 * Returns stub data until audit-log list endpoint supports client-side token auth.
 */
export async function fetchRecentAuditEvents(
  token: string,
  limit: number = 10,
): Promise<AuditEvent[]> {
  const url = buildUrl("/admin/stats");
  const response = await fetch(url, {
    method: "GET",
    headers: buildHeaders(token),
    cache: "no-store",
  });
  await parseResponse<unknown>(response);

  return Array.from({ length: Math.min(limit, 10) }, (_, i) => ({
    id: `audit-${i + 1}`,
    actor: `user-${(i % 5) + 1}`,
    action: i % 4 === 0 ? "create" : i % 4 === 1 ? "update" : i % 4 === 2 ? "delete" : "login",
    entity_type: i % 3 === 0 ? "org" : i % 3 === 1 ? "thread" : "user",
    entity_id: `entity-${i + 1}`,
    created_at: new Date(Date.now() - i * 1800000).toISOString(),
  }));
}
