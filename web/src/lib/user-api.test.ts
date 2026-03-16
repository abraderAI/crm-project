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
  fetchOrgs,
  fetchOrg,
  fetchSpaces,
  fetchSpace,
  fetchBoards,
  fetchBoard,
  fetchThreads,
  fetchThread,
  fetchMessages,
  fetchRevisions,
  fetchNotifications,
  fetchNotificationPreferences,
  fetchDigestSchedule,
  fetchSearch,
  fetchUserVote,
} from "./user-api";

describe("user-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  describe("fetchOrgs", () => {
    it("fetches paginated orgs with auth token", async () => {
      const response = {
        data: [{ id: "o1", name: "Org1", slug: "org1" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchOrgs();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/orgs", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });

    it("passes params through", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchOrgs({ cursor: "abc" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs",
        { cursor: "abc" },
        { token: "test-token" },
      );
    });

    it("throws when unauthenticated", async () => {
      mockGetToken.mockResolvedValue(null);

      await expect(fetchOrgs()).rejects.toThrow("Unauthenticated");
    });
  });

  describe("fetchOrg", () => {
    it("fetches a single org by slug", async () => {
      const org = { id: "o1", name: "Org1", slug: "org1" };
      mockServerFetch.mockResolvedValue(org);

      const result = await fetchOrg("org1");

      expect(mockServerFetch).toHaveBeenCalledWith("/orgs/org1", { token: "test-token" });
      expect(result).toEqual(org);
    });
  });

  describe("fetchSpaces", () => {
    it("fetches paginated spaces for an org", async () => {
      const response = { data: [{ id: "s1", name: "Space1" }], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchSpaces("org1");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/orgs/org1/spaces", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });

    it("passes params through", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchSpaces("org1", { type: "crm" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/spaces",
        { type: "crm" },
        { token: "test-token" },
      );
    });
  });

  describe("fetchSpace", () => {
    it("fetches a single space by slug", async () => {
      const space = { id: "s1", name: "Sales", slug: "sales" };
      mockServerFetch.mockResolvedValue(space);

      const result = await fetchSpace("org1", "sales");

      expect(mockServerFetch).toHaveBeenCalledWith("/orgs/org1/spaces/sales", {
        token: "test-token",
      });
      expect(result).toEqual(space);
    });
  });

  describe("fetchBoards", () => {
    it("fetches paginated boards for a space", async () => {
      const response = { data: [{ id: "b1", name: "Board1" }], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchBoards("org1", "sales");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards",
        undefined,
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchBoard", () => {
    it("fetches a single board by slug", async () => {
      const board = { id: "b1", name: "Pipeline", slug: "pipeline" };
      mockServerFetch.mockResolvedValue(board);

      const result = await fetchBoard("org1", "sales", "pipeline");

      expect(mockServerFetch).toHaveBeenCalledWith("/orgs/org1/spaces/sales/boards/pipeline", {
        token: "test-token",
      });
      expect(result).toEqual(board);
    });
  });

  describe("fetchThreads", () => {
    it("fetches paginated threads for a board", async () => {
      const response = { data: [{ id: "t1", title: "Lead" }], page_info: { has_more: true } };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchThreads("org1", "sales", "pipeline");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards/pipeline/threads",
        undefined,
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });

    it("passes params through", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchThreads("org1", "sales", "pipeline", { limit: "20" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards/pipeline/threads",
        { limit: "20" },
        { token: "test-token" },
      );
    });
  });

  describe("fetchThread", () => {
    it("fetches a single thread by slug", async () => {
      const thread = { id: "t1", title: "Lead A", slug: "lead-a" };
      mockServerFetch.mockResolvedValue(thread);

      const result = await fetchThread("org1", "sales", "pipeline", "lead-a");

      expect(mockServerFetch).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards/pipeline/threads/lead-a",
        { token: "test-token" },
      );
      expect(result).toEqual(thread);
    });
  });

  describe("fetchMessages", () => {
    it("fetches paginated messages for a thread", async () => {
      const response = {
        data: [{ id: "m1", body: "Hello" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchMessages("org1", "sales", "pipeline", "lead-a");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards/pipeline/threads/lead-a/messages",
        undefined,
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });

    it("passes params through", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchMessages("org1", "sales", "pipeline", "lead-a", { limit: "10" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards/pipeline/threads/lead-a/messages",
        { limit: "10" },
        { token: "test-token" },
      );
    });
  });

  describe("fetchNotifications", () => {
    it("fetches paginated notifications", async () => {
      const response = {
        data: [{ id: "n1", title: "New message" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchNotifications();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith("/notifications", undefined, {
        token: "test-token",
      });
      expect(result).toEqual(response);
    });
  });

  describe("fetchRevisions", () => {
    it("fetches paginated revisions via /revisions/{entityType}/{entityId}", async () => {
      const response = {
        data: [{ id: "r1", version: 1, editor_id: "u1" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchRevisions("thread", "t1-uuid");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/revisions/thread/t1-uuid",
        undefined,
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchNotificationPreferences", () => {
    it("fetches paginated notification preferences", async () => {
      const response = {
        data: [{ notification_type: "message", channel: "in_app", enabled: true }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchNotificationPreferences();

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/notifications/preferences",
        undefined,
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });
  });

  describe("fetchDigestSchedule", () => {
    it("returns a default schedule without making an API call (no backend endpoint)", async () => {
      const result = await fetchDigestSchedule();

      expect(mockServerFetch).not.toHaveBeenCalled();
      expect(result).toMatchObject({ frequency: "none" });
    });
  });

  describe("fetchSearch", () => {
    it("fetches search results with query", async () => {
      const response = {
        data: [{ entity_type: "thread", title: "Test" }],
        page_info: { has_more: false },
      };
      mockServerFetchPaginated.mockResolvedValue(response);

      const result = await fetchSearch("test query");

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/search",
        { q: "test query" },
        { token: "test-token" },
      );
      expect(result).toEqual(response);
    });

    it("passes additional params through", async () => {
      const response = { data: [], page_info: { has_more: false } };
      mockServerFetchPaginated.mockResolvedValue(response);

      await fetchSearch("test", { type: "thread" });

      expect(mockServerFetchPaginated).toHaveBeenCalledWith(
        "/search",
        { q: "test", type: "thread" },
        { token: "test-token" },
      );
    });
  });

  describe("fetchUserVote", () => {
    it("attempts to fetch vote status and returns the result when successful", async () => {
      const status = { voted: true };
      mockServerFetch.mockResolvedValue(status);

      const result = await fetchUserVote("org1", "sales", "pipeline", "lead-a");

      expect(mockServerFetch).toHaveBeenCalledWith(
        "/orgs/org1/spaces/sales/boards/pipeline/threads/lead-a/vote",
        { token: "test-token" },
      );
      expect(result).toEqual(status);
    });

    it("returns {voted:false} as fallback when endpoint errors (no GET in backend)", async () => {
      mockServerFetch.mockRejectedValue(new Error("405 Method Not Allowed"));

      const result = await fetchUserVote("org1", "sales", "pipeline", "lead-a");

      expect(result).toEqual({ voted: false });
    });

    it("returns voted false when user has not voted", async () => {
      const status = { voted: false };
      mockServerFetch.mockResolvedValue(status);

      const result = await fetchUserVote("org1", "sales", "pipeline", "lead-a");

      expect(result).toEqual({ voted: false });
    });
  });
});
