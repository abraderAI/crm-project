import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock @clerk/nextjs/server before imports.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs/server", () => ({
  auth: vi.fn().mockResolvedValue({ getToken: () => mockGetToken() }),
}));

// Mock api-client functions (all exported symbols).
const mockServerFetch = vi.fn();
const mockServerFetchPaginated = vi.fn();
const mockBuildUrl = vi.fn().mockReturnValue("http://localhost:8080/v1/test");
const mockBuildHeaders = vi.fn().mockReturnValue({ Authorization: "Bearer test-token" });
const mockParseResponse = vi.fn();
vi.mock("./api-client", () => ({
  serverFetch: (...args: unknown[]) => mockServerFetch(...args),
  serverFetchPaginated: (...args: unknown[]) => mockServerFetchPaginated(...args),
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
}));

// Mock global fetch for raw-fetch functions.
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import {
  createEmailInbox,
  deleteEmailInbox,
  dismissDLQEvent,
  fetchAdminOrg,
  fetchAdminOrgs,
  fetchAdminSettings,
  fetchAdminStats,
  fetchAdminUser,
  fetchAdminUsers,
  fetchApiUsage,
  fetchAuditLog,
  fetchBillingInfo,
  fetchChannelConfig,
  fetchChannelHealth,
  fetchDLQEvents,
  fetchEmailInboxes,
  fetchExports,
  fetchFailedAuths,
  fetchFeatureFlags,
  fetchFirstOrgId,
  fetchFlags,
  fetchLlmUsage,
  fetchMemberships,
  fetchPlatformAdmins,
  fetchRBACPolicy,
  fetchRecentLogins,
  fetchWebhookDeliveries,
  fetchWebhookSubscriptions,
  patchFeatureFlag,
  putChannelConfig,
  retryDLQEvent,
  updateEmailInbox,
} from "./admin-api";

describe("admin-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  describe("fetchAdminOrgs", () => {
    it("fetches paginated orgs", async () => {
      const response = { data: [{ id: "org_1", name: "Acme" }], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchAdminOrgs({ cursor: "abc" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/orgs",
        { cursor: "abc" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });

    it("works without params", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchAdminOrgs();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/admin/orgs", undefined, {
        token: "test-token",
      });
    });

    it("throws when unauthenticated", async () => {
      mockGetToken.mockResolvedValue(null);
      await expect(fetchAdminOrgs()).rejects.toThrow("Unauthenticated");
    });
  });

  describe("fetchAdminOrg", () => {
    it("fetches a single org by id", async () => {
      const org = { id: "org_1", name: "Acme", slug: "acme", member_count: 3 };
      mockServerFetch.mockResolvedValue(org);

      const result = await fetchAdminOrg("org_1");

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/orgs/org_1", { token: "test-token" });
      expect(result).toEqual(org);
    });

    it("throws when unauthenticated", async () => {
      mockGetToken.mockResolvedValue(null);
      await expect(fetchAdminOrg("org_1")).rejects.toThrow("Unauthenticated");
    });
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
    it("fetches billing info from org-scoped endpoint", async () => {
      const billing = {
        org_id: "org1",
        tier: "pro",
        payment_status: "active",
        invoices: [],
      };
      mockServerFetch.mockResolvedValue(billing);

      const result = await fetchBillingInfo();

      expect(mockServerFetch).toHaveBeenCalledWith("/orgs/default/billing", {
        token: "test-token",
      });
      expect(result).toEqual(billing);
    });
  });

  describe("fetchWebhookSubscriptions", () => {
    it("fetches paginated webhook subscriptions from org-scoped endpoint", async () => {
      const response = {
        data: [{ id: "ws1", url: "https://example.com/hook" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchWebhookSubscriptions();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/orgs/default/webhooks", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });
  });

  describe("fetchWebhookDeliveries", () => {
    it("fetches paginated webhook deliveries from platform-wide admin endpoint", async () => {
      const response = {
        data: [{ id: "d1", event_type: "message.created" }],
        page_info: { has_more: true },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchWebhookDeliveries({ cursor: "abc" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/webhooks/deliveries",
        { cursor: "abc" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchMemberships", () => {
    it("fetches paginated memberships from org-scoped endpoint", async () => {
      const response = {
        data: [{ id: "m1", user_id: "u1", role: "admin", org_id: "org1" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchMemberships();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/orgs/default/members", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });
  });

  describe("fetchAdminUser", () => {
    it("fetches a single user by ID", async () => {
      const user = { clerk_user_id: "u1", email: "u@example.com", display_name: "User" };
      mockServerFetch.mockResolvedValue(user);

      const result = await fetchAdminUser("u1");

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/users/u1", {
        token: "test-token",
      });
      expect(result).toEqual(user);
    });
  });

  describe("fetchRecentLogins", () => {
    it("fetches paginated recent logins", async () => {
      const response = { data: [{ id: "l1" }], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchRecentLogins({ cursor: "c" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/security/recent-logins",
        { cursor: "c" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchFailedAuths", () => {
    it("fetches paginated failed auths", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchFailedAuths();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/security/failed-auths",
        undefined,
        { token: "test-token" },
      );
    });
  });

  describe("fetchRBACPolicy", () => {
    it("fetches the effective RBAC policy", async () => {
      const policy = { rules: [] };
      mockServerFetch.mockResolvedValue(policy);

      const result = await fetchRBACPolicy();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/rbac-policy", { token: "test-token" });
      expect(result).toEqual(policy);
    });
  });

  describe("fetchLlmUsage", () => {
    it("fetches LLM usage", async () => {
      const usage = { entries: [] };
      mockServerFetch.mockResolvedValue(usage);

      const result = await fetchLlmUsage();

      expect(mockServerFetch).toHaveBeenCalledWith("/admin/llm-usage", { token: "test-token" });
      expect(result).toEqual(usage);
    });
  });

  describe("fetchExports", () => {
    it("fetches paginated exports", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchExports({ status: "done" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/exports",
        { status: "done" },
        { token: "test-token" },
      );
    });
  });

  describe("fetchApiUsage", () => {
    it("builds url and delegates to parseResponse", async () => {
      const data = { periods: [] };
      mockFetch.mockResolvedValue({ ok: true });
      mockParseResponse.mockResolvedValue(data);

      const result = await fetchApiUsage("7d");

      expect(mockBuildUrl).toHaveBeenCalledWith("/admin/api-usage", { period: "7d" });
      expect(mockBuildHeaders).toHaveBeenCalledWith("test-token");
      expect(result).toEqual(data);
    });

    it("uses 24h as default period", async () => {
      mockFetch.mockResolvedValue({ ok: true });
      mockParseResponse.mockResolvedValue({});

      await fetchApiUsage();

      expect(mockBuildUrl).toHaveBeenCalledWith("/admin/api-usage", { period: "24h" });
    });
  });

  describe("fetchFirstOrgId", () => {
    it("returns first org id when orgs exist", async () => {
      mockServerFetchPaginated.mockResolvedValue({
        data: [{ id: "org_real" }],
        page_info: { has_more: false },
      });

      const id = await fetchFirstOrgId();

      expect(id).toBe("org_real");
      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/admin/orgs",
        { limit: "1" },
        { token: "test-token" },
      );
    });

    it('falls back to "default" when data is empty', async () => {
      mockServerFetchPaginated.mockResolvedValue({
        data: [],
        page_info: { has_more: false },
      });

      const id = await fetchFirstOrgId();

      expect(id).toBe("default");
    });
  });

  describe("fetchChannelConfig", () => {
    it("fetches channel config via serverFetch", async () => {
      const config = { id: "c1", channel_type: "email", settings: "{}", enabled: true };
      mockServerFetch.mockResolvedValue(config);

      const result = await fetchChannelConfig("org1", "email");

      expect(mockServerFetch).toHaveBeenCalledWith("/orgs/org1/channels/email", {
        token: "test-token",
      });
      expect(result).toEqual(config);
    });
  });

  describe("putChannelConfig", () => {
    it("PUTs channel config and returns updated config", async () => {
      const updated = { id: "c1", channel_type: "email", settings: "{}", enabled: true };
      mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(updated) });

      const result = await putChannelConfig("org1", "email", { settings: "{}", enabled: true });

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/v1/orgs/org1/channels/email"),
        expect.objectContaining({ method: "PUT" }),
      );
      expect(result).toEqual(updated);
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 403 });

      await expect(putChannelConfig("org1", "email", { settings: "{}", enabled: false })).rejects.toThrow(
        "Failed to update channel config: 403",
      );
    });
  });

  describe("fetchChannelHealth", () => {
    it("returns matching channel health", async () => {
      const emailHealth = { channel_type: "email", status: "healthy", enabled: true };
      mockServerFetch.mockResolvedValue({
        channels: [emailHealth, { channel_type: "voice", status: "down", enabled: false }],
      });

      const result = await fetchChannelHealth("org1", "email");

      expect(result).toEqual(emailHealth);
    });

    it("returns null when channel type not found", async () => {
      mockServerFetch.mockResolvedValue({ channels: [] });

      const result = await fetchChannelHealth("org1", "email");

      expect(result).toBeNull();
    });
  });

  describe("fetchDLQEvents", () => {
    it("fetches DLQ events with channel_type param", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchDLQEvents("org1", "email");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/channels/dlq",
        { channel_type: "email" },
        { token: "test-token" },
      );
    });

    it("merges extra params", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchDLQEvents("org1", "voice", { cursor: "x" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/channels/dlq",
        { channel_type: "voice", cursor: "x" },
        { token: "test-token" },
      );
    });
  });

  describe("retryDLQEvent", () => {
    it("POSTs to retry endpoint", async () => {
      mockFetch.mockResolvedValue({ ok: true });

      await retryDLQEvent("org1", "email", "evt-1");

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/channels/dlq/evt-1/retry"),
        expect.objectContaining({ method: "POST" }),
      );
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 404 });

      await expect(retryDLQEvent("org1", "email", "evt-1")).rejects.toThrow(
        "Failed to retry DLQ event: 404",
      );
    });
  });

  describe("dismissDLQEvent", () => {
    it("POSTs to dismiss endpoint", async () => {
      mockFetch.mockResolvedValue({ ok: true });

      await dismissDLQEvent("org1", "email", "evt-1");

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/channels/dlq/evt-1/dismiss"),
        expect.objectContaining({ method: "POST" }),
      );
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 500 });

      await expect(dismissDLQEvent("org1", "email", "evt-1")).rejects.toThrow(
        "Failed to dismiss DLQ event: 500",
      );
    });
  });

  describe("patchFeatureFlag", () => {
    it("PATCHes a feature flag and returns updated flag", async () => {
      const flag = { key: "beta", enabled: true };
      mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(flag) });

      const result = await patchFeatureFlag("beta", true);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/admin/feature-flags/beta"),
        expect.objectContaining({ method: "PATCH" }),
      );
      expect(result).toEqual(flag);
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 422 });

      await expect(patchFeatureFlag("beta", false)).rejects.toThrow(
        "Failed to update feature flag: 422",
      );
    });
  });

  describe("fetchEmailInboxes", () => {
    it("fetches and unwraps inbox data array", async () => {
      const inboxes = [{ id: "i1", name: "Support", routing_action: "support_ticket" }];
      mockServerFetch.mockResolvedValue({ data: inboxes });

      const result = await fetchEmailInboxes("org1");

      expect(mockServerFetch).toHaveBeenCalledWith("/orgs/org1/channels/email/inboxes", {
        token: "test-token",
      });
      expect(result).toEqual(inboxes);
    });
  });

  describe("createEmailInbox", () => {
    const input = {
      name: "Support",
      imap_host: "imap.gmail.com",
      imap_port: 993,
      username: "support@acme.com",
      password: "app-pass",
      enabled: true,
    };

    it("POSTs and returns created inbox", async () => {
      const created = { id: "i1", ...input, routing_action: "support_ticket" };
      mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(created) });

      const result = await createEmailInbox("org1", input);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/orgs/org1/channels/email/inboxes"),
        expect.objectContaining({ method: "POST" }),
      );
      expect(result).toEqual(created);
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 400 });

      await expect(createEmailInbox("org1", input)).rejects.toThrow(
        "Failed to create email inbox: 400",
      );
    });
  });

  describe("updateEmailInbox", () => {
    const input = {
      name: "Support Renamed",
      imap_host: "imap.gmail.com",
      imap_port: 993,
      username: "support@acme.com",
      enabled: true,
    };

    it("PUTs and returns updated inbox", async () => {
      const updated = { id: "i1", ...input, routing_action: "support_ticket" };
      mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(updated) });

      const result = await updateEmailInbox("org1", "i1", input);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/orgs/org1/channels/email/inboxes/i1"),
        expect.objectContaining({ method: "PUT" }),
      );
      expect(result).toEqual(updated);
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 404 });

      await expect(updateEmailInbox("org1", "i1", input)).rejects.toThrow(
        "Failed to update email inbox: 404",
      );
    });
  });

  describe("deleteEmailInbox", () => {
    it("DELETEs the inbox", async () => {
      mockFetch.mockResolvedValue({ ok: true });

      await deleteEmailInbox("org1", "i1");

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/orgs/org1/channels/email/inboxes/i1"),
        expect.objectContaining({ method: "DELETE" }),
      );
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 403 });

      await expect(deleteEmailInbox("org1", "i1")).rejects.toThrow(
        "Failed to delete email inbox: 403",
      );
    });
  });

  describe("fetchFlags", () => {
    it("fetches paginated flags from org-scoped endpoint", async () => {
      const response = {
        data: [{ id: "f1", thread_id: "t1", reason: "Spam", status: "pending" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchFlags();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/orgs/default/flags", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });

    it("passes params through with explicit org", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchFlags("default", { status: "pending" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/default/flags",
        { status: "pending" },
        { token: "test-token" },
      );
    });
  });
});
