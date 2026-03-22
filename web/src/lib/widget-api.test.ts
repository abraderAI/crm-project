import { describe, expect, it } from "vitest";

import type {
  LeadsByStatus,
  RecentLead,
  ConversionMetrics,
  TicketSummary,
  TicketStats,
  BillingOverview,
  SystemHealth,
  AuditEvent,
} from "./widget-api";

describe("widget-api types", () => {
  it("exports LeadsByStatus interface", () => {
    const val: LeadsByStatus = {
      new_lead: 0, contacted: 0, qualified: 0, proposal: 0,
      negotiation: 0, closed_won: 0, closed_lost: 0, nurturing: 0,
    };
    expect(val.new_lead).toBe(0);
  });

  it("exports RecentLead interface", () => {
    const val: RecentLead = { id: "1", title: "t", source: "s", status: "new", created_at: "" };
    expect(val.id).toBe("1");
  });

  it("exports ConversionMetrics interface", () => {
    const val: ConversionMetrics = { anonymous_sessions: 0, registrations: 0, conversions: 0 };
    expect(val.conversions).toBe(0);
  });

  it("exports TicketSummary interface", () => {
    const val: TicketSummary = { id: "1", title: "t", status: "open", org_name: "o", created_at: "" };
    expect(val.status).toBe("open");
  });

  it("exports TicketStats interface", () => {
    const val: TicketStats = { open: 0, pending: 0, resolved: 0, avg_response_time: "0" };
    expect(val.open).toBe(0);
  });

  it("exports BillingOverview interface", () => {
    const val: BillingOverview = { paying_org_count: 0, mrr: 0, recent_payments: 0 };
    expect(val.mrr).toBe(0);
  });

  it("exports SystemHealth interface", () => {
    const val: SystemHealth = { api_status: "ok", db_status: "ok", channel_health: {}, uptime: "" };
    expect(val.api_status).toBe("ok");
  });

  it("exports AuditEvent interface", () => {
    const val: AuditEvent = { id: "1", actor: "a", action: "create", entity_type: "org", entity_id: "e", created_at: "" };
    expect(val.action).toBe("create");
  });
});
