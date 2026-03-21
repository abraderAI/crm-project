import { describe, expect, it, vi, beforeEach } from "vitest";
import {
  fetchOrgOverview,
  fetchOrgSupportTickets,
  fetchOrgSupportStats,
  fetchOrgMembers,
  updateMemberRole,
  removeMember,
  fetchOrgSpaces,
  updateSpaceRoleOverride,
  fetchOrgsClient,
  createOrgClient,
} from "./org-api";

const mockFetch = vi.fn();
global.fetch = mockFetch;

const mockClientMutate = vi.fn();
vi.mock("./api-client", () => ({
  buildHeaders: (token?: string | null) => ({
    "Content-Type": "application/json",
    Accept: "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }),
  buildUrl: (path: string, params?: Record<string, string>) => {
    const url = new URL(`http://localhost:8080/v1${path}`);
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v !== undefined && v !== "") url.searchParams.set(k, v);
      }
    }
    return url.toString();
  },
  parseResponse: async <T>(response: Response): Promise<T> => {
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return (await response.json()) as T;
  },
  clientMutate: (...args: unknown[]) => mockClientMutate(...args),
}));

function jsonResponse(data: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => data,
  } as Response;
}

describe("org-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("fetchOrgOverview", () => {
    it("returns org overview from org endpoint", async () => {
      mockFetch.mockResolvedValue(
        jsonResponse({
          name: "Acme Corp",
          slug: "acme",
          spaces: [{ id: "s1" }, { id: "s2" }],
          payment_status: "active",
          billing_tier: "enterprise",
        }),
      );

      const result = await fetchOrgOverview("token", "acme");
      expect(result.name).toBe("Acme Corp");
      expect(result.slug).toBe("acme");
      expect(result.member_count).toBe(2);
      expect(result.plan_status).toBe("active");
      expect(result.billing_tier).toBe("enterprise");
    });

    it("defaults plan status and billing tier when missing", async () => {
      mockFetch.mockResolvedValue(jsonResponse({ name: "Test", slug: "test" }));

      const result = await fetchOrgOverview("token", "test");
      expect(result.plan_status).toBe("active");
      expect(result.billing_tier).toBe("pro");
      expect(result.member_count).toBe(0);
    });

    it("throws on non-OK response", async () => {
      mockFetch.mockResolvedValue(jsonResponse({}, 404));
      await expect(fetchOrgOverview("token", "missing")).rejects.toThrow("HTTP 404");
    });
  });

  describe("fetchOrgSupportTickets", () => {
    it("fetches tickets filtered by org_id", async () => {
      const tickets = { data: [{ id: "t1" }], page_info: { has_more: false } };
      mockFetch.mockResolvedValue(jsonResponse(tickets));

      const result = await fetchOrgSupportTickets("token", "org-1", { limit: 5 });
      expect(result.data).toHaveLength(1);

      const calledUrl = mockFetch.mock.calls[0]![0] as string;
      expect(calledUrl).toContain("org_id=org-1");
      expect(calledUrl).toContain("limit=5");
    });

    it("passes cursor when provided", async () => {
      mockFetch.mockResolvedValue(jsonResponse({ data: [], page_info: { has_more: false } }));

      await fetchOrgSupportTickets("token", "org-1", { cursor: "abc" });
      const calledUrl = mockFetch.mock.calls[0]![0] as string;
      expect(calledUrl).toContain("cursor=abc");
    });
  });

  describe("fetchOrgSupportStats", () => {
    it("aggregates ticket counts by status", async () => {
      const tickets = {
        data: [
          { id: "1", status: "open" },
          { id: "2", status: "open" },
          { id: "3", status: "pending" },
          { id: "4", status: "resolved" },
          { id: "5", status: "closed" },
        ],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue(jsonResponse(tickets));

      const stats = await fetchOrgSupportStats("token", "org-1");
      expect(stats.open).toBe(2);
      expect(stats.pending).toBe(1);
      expect(stats.resolved).toBe(2);
      expect(stats.total).toBe(5);
    });

    it("defaults unknown statuses to open", async () => {
      const tickets = {
        data: [{ id: "1" }, { id: "2", status: "unknown" }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue(jsonResponse(tickets));

      const stats = await fetchOrgSupportStats("token", "org-1");
      expect(stats.open).toBe(2);
    });

    it("returns zeros for empty ticket list", async () => {
      mockFetch.mockResolvedValue(jsonResponse({ data: [], page_info: { has_more: false } }));

      const stats = await fetchOrgSupportStats("token", "org-1");
      expect(stats).toEqual({ open: 0, pending: 0, resolved: 0, total: 0 });
    });
  });

  describe("fetchOrgMembers", () => {
    it("fetches members for an org", async () => {
      const members = {
        data: [{ id: "m1", user_id: "u1", role: "admin" }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue(jsonResponse(members));

      const result = await fetchOrgMembers("token", "org-1");
      expect(result.data).toHaveLength(1);

      const calledUrl = mockFetch.mock.calls[0]![0] as string;
      expect(calledUrl).toContain("/orgs/org-1/members");
    });

    it("passes pagination params", async () => {
      mockFetch.mockResolvedValue(jsonResponse({ data: [], page_info: { has_more: false } }));

      await fetchOrgMembers("token", "org-1", { limit: 10, cursor: "xyz" });
      const calledUrl = mockFetch.mock.calls[0]![0] as string;
      expect(calledUrl).toContain("limit=10");
      expect(calledUrl).toContain("cursor=xyz");
    });
  });

  describe("updateMemberRole", () => {
    it("calls clientMutate with correct params", async () => {
      mockClientMutate.mockResolvedValue({ id: "m1", role: "moderator" });

      const result = await updateMemberRole("token", "org-1", "m1", "moderator");
      expect(result.role).toBe("moderator");
      expect(mockClientMutate).toHaveBeenCalledWith("PATCH", "/orgs/org-1/members/m1", {
        token: "token",
        body: { role: "moderator" },
      });
    });
  });

  describe("removeMember", () => {
    it("calls clientMutate with DELETE", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await removeMember("token", "org-1", "m1");
      expect(mockClientMutate).toHaveBeenCalledWith("DELETE", "/orgs/org-1/members/m1", {
        token: "token",
      });
    });
  });

  describe("fetchOrgSpaces", () => {
    it("fetches spaces for an org", async () => {
      const spaces = {
        data: [{ id: "s1", name: "General" }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue(jsonResponse(spaces));

      const result = await fetchOrgSpaces("token", "org-1");
      expect(result.data).toHaveLength(1);
    });
  });

  describe("updateSpaceRoleOverride", () => {
    it("calls clientMutate with PUT", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await updateSpaceRoleOverride("token", "org-1", "m1", "s1", "admin");
      expect(mockClientMutate).toHaveBeenCalledWith(
        "PUT",
        "/orgs/org-1/members/m1/space-roles/s1",
        { token: "token", body: { role: "admin" } },
      );
    });

    it("passes null role to remove override", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await updateSpaceRoleOverride("token", "org-1", "m1", "s1", null);
      expect(mockClientMutate).toHaveBeenCalledWith(
        "PUT",
        "/orgs/org-1/members/m1/space-roles/s1",
        { token: "token", body: { role: null } },
      );
    });
  });

  describe("fetchOrgsClient", () => {
    it("fetches orgs from admin endpoint and returns data array", async () => {
      const orgs = {
        data: [{ id: "org-1", name: "Acme", slug: "acme" }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue(jsonResponse(orgs));

      const result = await fetchOrgsClient("token");
      expect(result).toHaveLength(1);
      expect(result[0]!.name).toBe("Acme");

      const calledUrl = mockFetch.mock.calls[0]![0] as string;
      expect(calledUrl).toContain("/admin/orgs");
    });
  });

  describe("createOrgClient", () => {
    it("calls clientMutate with POST /orgs", async () => {
      const created = { id: "org-new", name: "New Org", slug: "new-org" };
      mockClientMutate.mockResolvedValue(created);

      const result = await createOrgClient("token", "New Org", "A description");
      expect(result.name).toBe("New Org");
      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs", {
        token: "token",
        body: { name: "New Org", description: "A description" },
      });
    });

    it("omits description when empty", async () => {
      mockClientMutate.mockResolvedValue({ id: "org-new", name: "X", slug: "x" });

      await createOrgClient("token", "X", "");
      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs", {
        token: "token",
        body: { name: "X", description: undefined },
      });
    });
  });
});
