import { describe, expect, it, vi, beforeEach } from "vitest";

const mockBuildHeaders = vi.fn().mockReturnValue({ Authorization: "Bearer tok" });
const mockBuildUrl = vi
  .fn()
  .mockImplementation((path: string) => `http://localhost:8080/v1${path}`);
const mockParseResponse = vi.fn();

vi.mock("./api-client", () => ({
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
}));

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import {
  fetchLeadsByStatus,
  fetchRecentLeads,
  fetchConversionMetrics,
  fetchOpenTickets,
  fetchTicketStats,
  fetchBillingOverview,
  fetchSystemHealth,
  fetchRecentAuditEvents,
} from "./widget-api";

describe("widget-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetch.mockResolvedValue({ status: 200 });
  });

  describe("fetchLeadsByStatus", () => {
    it("returns lead counts grouped by status", async () => {
      mockParseResponse.mockResolvedValue({ threads: { total: 100 } });

      const result = await fetchLeadsByStatus("test-token");

      expect(mockBuildUrl).toHaveBeenCalledWith("/admin/stats");
      expect(mockBuildHeaders).toHaveBeenCalledWith("test-token");
      expect(result.new_lead).toBe(30);
      expect(result.contacted).toBe(20);
      expect(result.qualified).toBe(15);
      expect(result.proposal).toBe(10);
    });

    it("handles zero total threads", async () => {
      mockParseResponse.mockResolvedValue({ threads: { total: 0 } });

      const result = await fetchLeadsByStatus("token");

      expect(result.new_lead).toBe(0);
      expect(result.contacted).toBe(0);
    });

    it("handles missing threads field", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchLeadsByStatus("token");

      expect(result.new_lead).toBe(0);
    });
  });

  describe("fetchRecentLeads", () => {
    it("returns leads with default limit of 10", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentLeads("token");

      expect(result).toHaveLength(10);
      expect(result[0]?.id).toBe("lead-1");
      expect(result[0]?.source).toBeDefined();
      expect(result[0]?.status).toBeDefined();
    });

    it("respects custom limit", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentLeads("token", 5);

      expect(result).toHaveLength(5);
    });

    it("caps at 10 even with larger limit", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentLeads("token", 20);

      expect(result).toHaveLength(10);
    });

    it("includes source field on each lead", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentLeads("token");

      for (const lead of result) {
        expect(["chatbot", "website", "referral"]).toContain(lead.source);
      }
    });
  });

  describe("fetchConversionMetrics", () => {
    it("returns funnel metrics derived from stats", async () => {
      mockParseResponse.mockResolvedValue({
        users: { total: 100 },
        orgs: { total: 10 },
      });

      const result = await fetchConversionMetrics("token");

      expect(result.anonymous_sessions).toBe(300);
      expect(result.registrations).toBe(100);
      expect(result.conversions).toBe(10);
    });

    it("handles missing user/org stats", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchConversionMetrics("token");

      expect(result.anonymous_sessions).toBe(0);
      expect(result.registrations).toBe(0);
      expect(result.conversions).toBe(0);
    });
  });

  describe("fetchOpenTickets", () => {
    it("returns ticket summaries", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchOpenTickets("token");

      expect(result).toHaveLength(5);
      expect(result[0]?.id).toBe("ticket-1");
      expect(result[0]?.status).toBeDefined();
      expect(result[0]?.org_name).toBeDefined();
    });

    it("includes varied statuses", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchOpenTickets("token");
      const statuses = result.map((t) => t.status);

      expect(statuses).toContain("open");
      expect(statuses).toContain("pending");
    });
  });

  describe("fetchTicketStats", () => {
    it("returns ticket counts", async () => {
      mockParseResponse.mockResolvedValue({ threads: { total: 100 } });

      const result = await fetchTicketStats("token");

      expect(result.open).toBe(30);
      expect(result.pending).toBe(20);
      expect(result.resolved).toBe(50);
      expect(result.avg_response_time).toBe("2h 15m");
    });

    it("handles zero threads", async () => {
      mockParseResponse.mockResolvedValue({ threads: { total: 0 } });

      const result = await fetchTicketStats("token");

      expect(result.open).toBe(0);
      expect(result.pending).toBe(0);
      expect(result.resolved).toBe(0);
    });
  });

  describe("fetchBillingOverview", () => {
    it("returns billing data with org count", async () => {
      mockParseResponse.mockResolvedValue({ orgs: { total: 25 } });

      const result = await fetchBillingOverview("token");

      expect(result.paying_org_count).toBe(25);
      expect(result.mrr).toBe(0);
      expect(result.recent_payments).toBe(0);
    });

    it("handles missing org stats", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchBillingOverview("token");

      expect(result.paying_org_count).toBe(0);
    });
  });

  describe("fetchSystemHealth", () => {
    it("returns health status when DB has data", async () => {
      mockParseResponse.mockResolvedValue({ db_size_bytes: 1024 });

      const result = await fetchSystemHealth("token");

      expect(result.api_status).toBe("healthy");
      expect(result.db_status).toBe("healthy");
      expect(result.uptime).toBe("99.9%");
      expect(result.channel_health).toHaveProperty("email");
      expect(result.channel_health).toHaveProperty("chat");
      expect(result.channel_health).toHaveProperty("voice");
    });

    it("returns unknown DB status when size is 0", async () => {
      mockParseResponse.mockResolvedValue({ db_size_bytes: 0 });

      const result = await fetchSystemHealth("token");

      expect(result.db_status).toBe("unknown");
    });
  });

  describe("fetchRecentAuditEvents", () => {
    it("returns audit events with default limit", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentAuditEvents("token");

      expect(result).toHaveLength(10);
      expect(result[0]?.id).toBe("audit-1");
      expect(result[0]?.actor).toBeDefined();
      expect(result[0]?.action).toBeDefined();
    });

    it("respects custom limit", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentAuditEvents("token", 5);

      expect(result).toHaveLength(5);
    });

    it("includes varied action types", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentAuditEvents("token");
      const actions = result.map((e) => e.action);

      expect(actions).toContain("create");
      expect(actions).toContain("update");
    });

    it("includes varied entity types", async () => {
      mockParseResponse.mockResolvedValue({});

      const result = await fetchRecentAuditEvents("token");
      const types = result.map((e) => e.entity_type);

      expect(types).toContain("org");
      expect(types).toContain("thread");
      expect(types).toContain("user");
    });
  });
});
