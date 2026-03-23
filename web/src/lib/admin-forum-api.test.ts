import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchAdminForumThreads,
  toggleForumThreadPin,
  toggleForumThreadHidden,
  toggleForumThreadLocked,
} from "./admin-forum-api";

const mockThread = {
  id: "t1",
  title: "Test",
  slug: "test",
  is_pinned: false,
  is_hidden: false,
  is_locked: false,
};

describe("admin-forum-api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("fetchAdminForumThreads calls the correct URL", async () => {
    const spy = vi.spyOn(global, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ data: [mockThread], page_info: { has_more: false } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const result = await fetchAdminForumThreads("token", { limit: 10 });
    expect(result.data).toHaveLength(1);
    expect(spy).toHaveBeenCalledTimes(1);
    const url = spy.mock.calls[0][0] as string;
    expect(url).toContain("global-forum/threads");
    expect(url).toContain("limit=10");
  });

  it("toggleForumThreadPin sends PATCH with is_pinned", async () => {
    const spy = vi.spyOn(global, "fetch").mockResolvedValue(
      new Response(JSON.stringify(mockThread), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await toggleForumThreadPin("token", "test", true);
    expect(spy).toHaveBeenCalledTimes(1);
    const [, opts] = spy.mock.calls[0] as [string, RequestInit];
    expect(opts.method).toBe("PATCH");
    expect(opts.body).toContain("is_pinned");
  });

  it("toggleForumThreadHidden sends PATCH with is_hidden", async () => {
    const spy = vi.spyOn(global, "fetch").mockResolvedValue(
      new Response(JSON.stringify(mockThread), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await toggleForumThreadHidden("token", "test", true);
    const [, opts] = spy.mock.calls[0] as [string, RequestInit];
    expect(opts.body).toContain("is_hidden");
  });

  it("toggleForumThreadLocked sends PATCH with is_locked", async () => {
    const spy = vi.spyOn(global, "fetch").mockResolvedValue(
      new Response(JSON.stringify(mockThread), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await toggleForumThreadLocked("token", "test", true);
    const [, opts] = spy.mock.calls[0] as [string, RequestInit];
    expect(opts.body).toContain("is_locked");
  });
});
