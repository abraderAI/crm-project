import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock @clerk/nextjs/server before imports.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs/server", () => ({
  auth: vi.fn().mockResolvedValue({ getToken: () => mockGetToken() }),
}));

// Mock api-client functions.
const mockServerFetch = vi.fn();
const mockServerFetchPaginated = vi.fn();
vi.mock("./api-client", () => ({
  serverFetch: (...args: unknown[]) => mockServerFetch(...args),
  serverFetchPaginated: (...args: unknown[]) => mockServerFetchPaginated(...args),
}));

import {
  fetchAdminStats,
  fetchAdminSettings,
  fetchAdminUsers,
  fetchPlatformAdmins,
  fetchAuditLog,
  fetchFeatureFlags,
  fetchBillingInfo,
  fetchWebhookSubscriptions,
  fetchWebhookDeliveries,
  fetchMemberships,
  fetchFlags,
} from "./admin-api";

describe("admin-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  describe("fetchAdminStats", () => {
    it("fetches stats with auth token", async () => {
      const stats = { orgs: { total: 1, last_7d: 0, last_30d: 1 } };
      mockServerFetch.mockResolvedValue(stats);

      const result = await fetchAdminStats();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/stats", { token: "test-token" });
      expect(result).toEqual(stats);
    });

    it("throws when unauthenticated", async () => {
      mockGetToken.mockResolvedValue(null);

      await expect(fetchAdminStats()).rejects.toThrow("Unauthenticated");
    });
  });

  describe("fetchAdminSettings", () => {
    it("fetches settings with auth token", async () => {
      const settings = { webhook_retry_policy: { max_attempts: 5 } };
      mockServerFetch.mockResolvedValue(settings);

      const result = await fetchAdminSettings();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/settings", { token: "test-token" });
      expect(result).toEqual(settings);
    });
  });

  describe("fetchAdminUsers", () => {
    it("fetches paginated users", async () => {
      const response = { data: [{ clerk_user_id: "u1" }], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchAdminUsers({ email: "test" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/users",
        { email: "test" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });

    it("works without params", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchAdminUsers();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/admin/users", undefined, {
        token: "test-token",
      });
    });
  });

  describe("fetchPlatformAdmins", () => {
    it("fetches and unwraps data array", async () => {
      const admins = [{ user_id: "u1", granted_by: "bootstrap", is_active: true }];
      mockServerFetch.mockResolvedValue({ data: admins });

      const result = await fetchPlatformAdmins();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/platform-admins", {
        token: "test-token",
      });
      expect(result).toEqual(admins);
    });
  });

  describe("fetchAuditLog", () => {
    it("fetches paginated audit log", async () => {
      const response = { data: [{ id: "a1" }], page_info: { has_more: true } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchAuditLog({ action: "create" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/audit-log",
        { action: "create" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchFeatureFlags", () => {
    it("fetches and unwraps data array", async () => {
      const flags = [{ key: "community_voting", enabled: true }];
      mockServerFetch.mockResolvedValue({ data: flags });

      const result = await fetchFeatureFlags();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/feature-flags", { token: "test-token" });
      expect(result).toEqual(flags);
    });
  });

  describe("fetchBillingInfo", () => {
    it("fetches billing info with auth token", async () => {
      const billing = {
        org_id: "org1",
        tier: "pro",
        payment_status: "active",
        invoices: [],
      };
      mockServerFetch.mockResolvedValue(billing);

      const result = await fetchBillingInfo();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/billing", { token: "test-token" });
      expect(result).toEqual(billing);
    });
  });

  describe("fetchWebhookSubscriptions", () => {
    it("fetches paginated webhook subscriptions", async () => {
      const response = {
        data: [{ id: "ws1", url: "https://example.com/hook" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchWebhookSubscriptions();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/admin/webhooks", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });
  });

  describe("fetchWebhookDeliveries", () => {
    it("fetches paginated webhook deliveries", async () => {
      const response = {
        data: [{ id: "d1", event_type: "message.created" }],
        page_info: { has_more: true },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchWebhookDeliveries({ cursor: "abc" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/webhook-deliveries",
        { cursor: "abc" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchMemberships", () => {
    it("fetches paginated memberships", async () => {
      const response = {
        data: [{ id: "m1", user_id: "u1", role: "admin", org_id: "org1" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchMemberships();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/admin/memberships", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });
  });

  describe("fetchFlags", () => {
    it("fetches paginated flags", async () => {
      const response = {
        data: [{ id: "f1", thread_id: "t1", reason: "Spam", status: "pending" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchFlags();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/admin/flags", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });

    it("passes params through", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchFlags({ status: "pending" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/flags",
        { status: "pending" },
        { token: "test-token" },
      );
    });
  });
});
