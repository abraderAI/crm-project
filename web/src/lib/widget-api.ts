/**
 * Widget API types.
 *
 * Stub fetch functions have been removed — these types remain for widget
 * components that will be wired to real backend endpoints in the future.
 */

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

/** Conversion funnel metrics. */
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
