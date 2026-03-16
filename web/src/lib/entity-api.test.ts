import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock api-client — keep real buildUrl/buildHeaders/parseResponse, mock clientMutate.
const mockClientMutate = vi.fn();
vi.mock("./api-client", async () => {
  const actual = await vi.importActual<typeof import("./api-client")>("./api-client");
  return {
    ...actual,
    clientMutate: (...args: unknown[]) => mockClientMutate(...args),
  };
});

// Mock fetch globally for non-clientMutate functions.
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import {
  createOrg,
  updateOrg,
  deleteOrg,
  createSpace,
  updateSpace,
  deleteSpace,
  createBoard,
  updateBoard,
  deleteBoard,
  createThread,
  createMessage,
  fetchThreadRevisions,
  fetchThreadUploads,
  uploadFile,
  deleteUpload,
  saveNotificationPreferences,
  saveDigestSchedule,
  createWebhook,
  deleteWebhook,
  toggleWebhook,
  replayWebhookDelivery,
  addMembership,
  changeMembershipRole,
  removeMembership,
  toggleVote,
  createFlag,
  resolveFlag,
  dismissFlag,
} from "./entity-api";

describe("entity-api", () => {
  const token = "test-token";

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // --- Org ---

  describe("createOrg", () => {
    it("posts to /orgs with body and token", async () => {
      const org = { id: "o1", name: "Acme", slug: "acme" };
      mockClientMutate.mockResolvedValue(org);

      const result = await createOrg(token, {
        name: "Acme",
        description: "Corp",
        metadata: "{}",
      });

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs", {
        token,
        body: { name: "Acme", description: "Corp", metadata: "{}" },
      });
      expect(result).toEqual(org);
    });
  });

  describe("updateOrg", () => {
    it("patches /orgs/:slug with body and token", async () => {
      const org = { id: "o1", name: "Acme Inc", slug: "acme" };
      mockClientMutate.mockResolvedValue(org);

      const result = await updateOrg(token, "acme", {
        name: "Acme Inc",
        description: "",
        metadata: "{}",
      });

      expect(mockClientMutate).toHaveBeenCalledWith("PATCH", "/orgs/acme", {
        token,
        body: { name: "Acme Inc", description: "", metadata: "{}" },
      });
      expect(result).toEqual(org);
    });
  });

  describe("deleteOrg", () => {
    it("deletes /orgs/:slug with token", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await deleteOrg(token, "acme");

      expect(mockClientMutate).toHaveBeenCalledWith("DELETE", "/orgs/acme", {
        token,
      });
    });
  });

  // --- Space ---

  describe("createSpace", () => {
    it("posts to /orgs/:org/spaces with body and token", async () => {
      const space = { id: "s1", name: "Sales", slug: "sales" };
      mockClientMutate.mockResolvedValue(space);

      const result = await createSpace(token, "acme", {
        name: "Sales",
        description: "",
        metadata: "{}",
        type: "crm",
      });

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/acme/spaces", {
        token,
        body: { name: "Sales", description: "", metadata: "{}", type: "crm" },
      });
      expect(result).toEqual(space);
    });
  });

  describe("updateSpace", () => {
    it("patches /orgs/:org/spaces/:space with body and token", async () => {
      const space = { id: "s1", name: "Sales Team", slug: "sales" };
      mockClientMutate.mockResolvedValue(space);

      const result = await updateSpace(token, "acme", "sales", {
        name: "Sales Team",
        description: "Updated",
        metadata: "{}",
      });

      expect(mockClientMutate).toHaveBeenCalledWith("PATCH", "/orgs/acme/spaces/sales", {
        token,
        body: { name: "Sales Team", description: "Updated", metadata: "{}" },
      });
      expect(result).toEqual(space);
    });
  });

  describe("deleteSpace", () => {
    it("deletes /orgs/:org/spaces/:space with token", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await deleteSpace(token, "acme", "sales");

      expect(mockClientMutate).toHaveBeenCalledWith("DELETE", "/orgs/acme/spaces/sales", {
        token,
      });
    });
  });

  // --- Board ---

  describe("createBoard", () => {
    it("posts to /orgs/:org/spaces/:space/boards with body and token", async () => {
      const board = { id: "b1", name: "Pipeline", slug: "pipeline" };
      mockClientMutate.mockResolvedValue(board);

      const result = await createBoard(token, "acme", "sales", {
        name: "Pipeline",
        description: "",
        metadata: "{}",
      });

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/acme/spaces/sales/boards", {
        token,
        body: { name: "Pipeline", description: "", metadata: "{}" },
      });
      expect(result).toEqual(board);
    });
  });

  describe("updateBoard", () => {
    it("patches /orgs/:org/spaces/:space/boards/:board", async () => {
      const board = { id: "b1", name: "Pipeline v2", slug: "pipeline" };
      mockClientMutate.mockResolvedValue(board);

      const result = await updateBoard(token, "acme", "sales", "pipeline", {
        name: "Pipeline v2",
        description: "Updated",
        metadata: "{}",
      });

      expect(mockClientMutate).toHaveBeenCalledWith(
        "PATCH",
        "/orgs/acme/spaces/sales/boards/pipeline",
        { token, body: { name: "Pipeline v2", description: "Updated", metadata: "{}" } },
      );
      expect(result).toEqual(board);
    });
  });

  describe("deleteBoard", () => {
    it("deletes /orgs/:org/spaces/:space/boards/:board", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await deleteBoard(token, "acme", "sales", "pipeline");

      expect(mockClientMutate).toHaveBeenCalledWith(
        "DELETE",
        "/orgs/acme/spaces/sales/boards/pipeline",
        { token },
      );
    });
  });

  // --- Thread ---

  describe("createThread", () => {
    it("posts to /orgs/:org/spaces/:space/boards/:board/threads with body and token", async () => {
      const thread = { id: "t1", title: "New Lead", slug: "new-lead" };
      mockClientMutate.mockResolvedValue(thread);

      const result = await createThread(token, "acme", "sales", "pipeline", {
        title: "New Lead",
        body: "Details here",
        metadata: "{}",
      });

      expect(mockClientMutate).toHaveBeenCalledWith(
        "POST",
        "/orgs/acme/spaces/sales/boards/pipeline/threads",
        {
          token,
          body: { title: "New Lead", body: "Details here", metadata: "{}" },
        },
      );
      expect(result).toEqual(thread);
    });
  });

  // --- Revisions ---

  describe("fetchThreadRevisions", () => {
    it("fetches revisions via GET /revisions/{entityType}/{entityId}", async () => {
      const response = {
        data: [{ id: "r1", version: 1 }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(response),
      });

      const result = await fetchThreadRevisions(token, "thread", "t1-uuid");

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/revisions/thread/t1-uuid"),
        expect.objectContaining({ method: "GET" }),
      );
      expect(result).toEqual(response);
    });
  });

  // --- Uploads ---

  describe("fetchThreadUploads", () => {
    it("returns empty list without making a fetch call (no list endpoint in backend)", async () => {
      const result = await fetchThreadUploads(token, "acme", "sales", "pipeline", "lead-a");

      expect(mockFetch).not.toHaveBeenCalled();
      expect(result).toEqual({ data: [], page_info: { has_more: false } });
    });
  });

  describe("uploadFile", () => {
    it("posts multipart form data to POST /uploads", async () => {
      const upload = { id: "u1", filename: "test.png" };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(upload),
      });
      const file = new File(["content"], "test.png", { type: "image/png" });

      const result = await uploadFile(token, "acme", "sales", "pipeline", "lead-a", file);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/uploads"),
        expect.objectContaining({
          method: "POST",
          headers: { Authorization: `Bearer ${token}` },
        }),
      );
      expect(result).toEqual(upload);
    });
  });

  describe("deleteUpload", () => {
    it("deletes /uploads/:id with token", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await deleteUpload(token, "u1");

      expect(mockClientMutate).toHaveBeenCalledWith("DELETE", "/uploads/u1", {
        token,
      });
    });
  });

  // --- Notification Preferences ---

  describe("saveNotificationPreferences", () => {
    it("puts preferences to /notifications/preferences", async () => {
      mockClientMutate.mockResolvedValue(undefined);
      const prefs = [
        { notificationType: "message" as const, channel: "in_app" as const, enabled: true },
        { notificationType: "message" as const, channel: "email" as const, enabled: false },
      ];

      await saveNotificationPreferences(token, prefs);

      expect(mockClientMutate).toHaveBeenCalledWith("PUT", "/notifications/preferences", {
        token,
        body: { preferences: prefs },
      });
    });
  });

  describe("saveDigestSchedule", () => {
    it("puts digest schedule to /notifications/digest", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await saveDigestSchedule(token, "weekly");

      expect(mockClientMutate).toHaveBeenCalledWith("PUT", "/notifications/digest", {
        token,
        body: { frequency: "weekly" },
      });
    });
  });

  // --- Message ---

  describe("createMessage", () => {
    it("posts to .../threads/:thread/messages with body and token", async () => {
      const message = { id: "m1", body: "Hello", type: "comment" };
      mockClientMutate.mockResolvedValue(message);

      const result = await createMessage(token, "acme", "sales", "pipeline", "new-lead", {
        body: "Hello",
        type: "comment",
      });

      expect(mockClientMutate).toHaveBeenCalledWith(
        "POST",
        "/orgs/acme/spaces/sales/boards/pipeline/threads/new-lead/messages",
        {
          token,
          body: { body: "Hello", type: "comment" },
        },
      );
      expect(result).toEqual(message);
    });

    it("defaults type to comment when not specified", async () => {
      const message = { id: "m2", body: "Hi", type: "comment" };
      mockClientMutate.mockResolvedValue(message);

      await createMessage(token, "acme", "sales", "pipeline", "new-lead", {
        body: "Hi",
      });

      expect(mockClientMutate).toHaveBeenCalledWith(
        "POST",
        "/orgs/acme/spaces/sales/boards/pipeline/threads/new-lead/messages",
        {
          token,
          body: { body: "Hi", type: "comment" },
        },
      );
    });
  });

  // --- Webhook mutations ---

  describe("createWebhook", () => {
    it("posts to /orgs/default/webhooks with url and event_filter", async () => {
      const webhook = { id: "ws1", url: "https://example.com/hook" };
      mockClientMutate.mockResolvedValue(webhook);

      const result = await createWebhook(token, "https://example.com/hook", "message.created");

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/default/webhooks", {
        token,
        body: { url: "https://example.com/hook", event_filter: "message.created" },
      });
      expect(result).toEqual(webhook);
    });
  });

  describe("deleteWebhook", () => {
    it("deletes /orgs/default/webhooks/:id with token", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await deleteWebhook(token, "ws1");

      expect(mockClientMutate).toHaveBeenCalledWith("DELETE", "/orgs/default/webhooks/ws1", {
        token,
      });
    });
  });

  describe("toggleWebhook", () => {
    it("patches /orgs/default/webhooks/:id with token (no toggle endpoint in backend)", async () => {
      const updated = { id: "ws1", is_active: false };
      mockClientMutate.mockResolvedValue(updated);

      const result = await toggleWebhook(token, "ws1");

      expect(mockClientMutate).toHaveBeenCalledWith("PATCH", "/orgs/default/webhooks/ws1", {
        token,
      });
      expect(result).toEqual(updated);
    });
  });

  describe("replayWebhookDelivery", () => {
    it("posts to /orgs/default/webhooks/:webhookId/deliveries/:deliveryId/replay", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await replayWebhookDelivery(token, "ws1", "d1");

      expect(mockClientMutate).toHaveBeenCalledWith(
        "POST",
        "/orgs/default/webhooks/ws1/deliveries/d1/replay",
        { token },
      );
    });
  });

  // --- Membership mutations ---

  describe("addMembership", () => {
    it("posts to /orgs/default/members with user_id and role", async () => {
      const membership = { id: "m1", user_id: "u1", role: "admin" };
      mockClientMutate.mockResolvedValue(membership);

      const result = await addMembership(token, "u1", "admin");

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/default/members", {
        token,
        body: { user_id: "u1", role: "admin" },
      });
      expect(result).toEqual(membership);
    });
  });

  describe("changeMembershipRole", () => {
    it("patches /orgs/default/members/:userId with new role", async () => {
      const updated = { id: "m1", user_id: "u1", role: "moderator" };
      mockClientMutate.mockResolvedValue(updated);

      const result = await changeMembershipRole(token, "u1", "moderator");

      expect(mockClientMutate).toHaveBeenCalledWith("PATCH", "/orgs/default/members/u1", {
        token,
        body: { role: "moderator" },
      });
      expect(result).toEqual(updated);
    });
  });

  describe("removeMembership", () => {
    it("deletes /orgs/default/members/:userId with token", async () => {
      mockClientMutate.mockResolvedValue(undefined);

      await removeMembership(token, "u1");

      expect(mockClientMutate).toHaveBeenCalledWith("DELETE", "/orgs/default/members/u1", {
        token,
      });
    });
  });

  // --- Vote mutations ---

  describe("toggleVote", () => {
    it("posts to .../threads/:thread/vote with token", async () => {
      const vote = { id: "v1", thread_id: "t1", user_id: "u1", weight: 1 };
      mockClientMutate.mockResolvedValue(vote);

      const result = await toggleVote(token, "acme", "sales", "pipeline", "new-lead");

      expect(mockClientMutate).toHaveBeenCalledWith(
        "POST",
        "/orgs/acme/spaces/sales/boards/pipeline/threads/new-lead/vote",
        { token },
      );
      expect(result).toEqual(vote);
    });
  });

  // --- Flag mutations ---

  describe("createFlag", () => {
    it("posts to /orgs/default/flags with thread_id and reason", async () => {
      const flag = { id: "f1", thread_id: "t1", reason: "Spam" };
      mockClientMutate.mockResolvedValue(flag);

      const result = await createFlag(token, "t1", "Spam");

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/default/flags", {
        token,
        body: { thread_id: "t1", reason: "Spam" },
      });
      expect(result).toEqual(flag);
    });
  });

  describe("resolveFlag", () => {
    it("posts to /orgs/default/flags/:id/resolve with note", async () => {
      const flag = { id: "f1", status: "resolved" };
      mockClientMutate.mockResolvedValue(flag);

      const result = await resolveFlag(token, "f1", "Addressed by moderator");

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/default/flags/f1/resolve", {
        token,
        body: { resolution_note: "Addressed by moderator" },
      });
      expect(result).toEqual(flag);
    });
  });

  describe("dismissFlag", () => {
    it("posts to /orgs/default/flags/:id/dismiss with token", async () => {
      const flag = { id: "f1", status: "dismissed" };
      mockClientMutate.mockResolvedValue(flag);

      const result = await dismissFlag(token, "f1");

      expect(mockClientMutate).toHaveBeenCalledWith("POST", "/orgs/default/flags/f1/dismiss", {
        token,
      });
      expect(result).toEqual(flag);
    });
  });
});
