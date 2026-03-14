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
    it("fetches revisions via GET with auth header", async () => {
      const response = {
        data: [{ id: "r1", version: 1 }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(response),
      });

      const result = await fetchThreadRevisions(token, "acme", "sales", "pipeline", "lead-a");

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/threads/lead-a/revisions"),
        expect.objectContaining({ method: "GET" }),
      );
      expect(result).toEqual(response);
    });
  });

  // --- Uploads ---

  describe("fetchThreadUploads", () => {
    it("fetches uploads via GET with auth header", async () => {
      const response = {
        data: [{ id: "u1", filename: "doc.pdf" }],
        page_info: { has_more: false },
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(response),
      });

      const result = await fetchThreadUploads(token, "acme", "sales", "pipeline", "lead-a");

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/threads/lead-a/uploads"),
        expect.objectContaining({ method: "GET" }),
      );
      expect(result).toEqual(response);
    });
  });

  describe("uploadFile", () => {
    it("posts multipart form data with file", async () => {
      const upload = { id: "u1", filename: "test.png" };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(upload),
      });
      const file = new File(["content"], "test.png", { type: "image/png" });

      const result = await uploadFile(token, "acme", "sales", "pipeline", "lead-a", file);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/threads/lead-a/uploads"),
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
});
