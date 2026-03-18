import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import {
  GLOBAL_SPACES,
  fetchGlobalThreads,
  fetchGlobalLeads,
  fetchGlobalThread,
  fetchUserForumActivity,
  fetchUserSupportTickets,
  createForumThread,
  createSupportTicket,
} from "./global-api";

const mockFetch = vi.fn();

beforeEach(() => {
  mockFetch.mockReset();
  vi.stubGlobal("fetch", mockFetch);
});

afterEach(() => {
  vi.restoreAllMocks();
});

function mockOkResponse<T>(data: T): Response {
  return {
    ok: true,
    status: 200,
    json: () => Promise.resolve(data),
  } as unknown as Response;
}

function mockErrorResponse(status: number): Response {
  return {
    ok: false,
    status,
    statusText: "Error",
    json: () =>
      Promise.resolve({
        type: "about:blank",
        title: "Error",
        status,
        detail: `HTTP ${status}`,
      }),
  } as unknown as Response;
}

const THREAD_FIXTURE = {
  id: "t1",
  board_id: "b1",
  title: "Test Thread",
  slug: "test-thread",
  metadata: "{}",
  author_id: "u1",
  is_pinned: false,
  is_locked: false,
  is_hidden: false,
  vote_score: 0,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("GLOBAL_SPACES", () => {
  it("defines all four global space slugs", () => {
    expect(GLOBAL_SPACES.DOCS).toBe("global-docs");
    expect(GLOBAL_SPACES.FORUM).toBe("global-forum");
    expect(GLOBAL_SPACES.SUPPORT).toBe("global-support");
    expect(GLOBAL_SPACES.LEADS).toBe("global-leads");
  });
});

describe("fetchGlobalThreads", () => {
  it("fetches threads from a global space without auth", async () => {
    const body = { data: [THREAD_FIXTURE], page_info: { has_more: false } };
    mockFetch.mockResolvedValue(mockOkResponse(body));

    const result = await fetchGlobalThreads("global-docs");

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/global-spaces/global-docs/threads");
    expect(options.method).toBe("GET");
    expect(result.data).toHaveLength(1);
    expect(result.data[0]?.title).toBe("Test Thread");
  });

  it("passes query params (limit, cursor, thread_type)", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchGlobalThreads("global-docs", { limit: 5, cursor: "abc", thread_type: "wiki" });

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("limit=5");
    expect(url).toContain("cursor=abc");
    expect(url).toContain("thread_type=wiki");
  });

  it("passes auth token when provided", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchGlobalThreads("global-docs", undefined, "my-token");

    const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    const headers = options.headers as Record<string, string>;
    expect(headers["Authorization"]).toBe("Bearer my-token");
  });

  it("throws on API error", async () => {
    mockFetch.mockResolvedValue(mockErrorResponse(500));
    await expect(fetchGlobalThreads("global-docs")).rejects.toThrow();
  });
});

describe("fetchUserForumActivity", () => {
  it("fetches user's forum threads with mine=true", async () => {
    const body = { data: [THREAD_FIXTURE], page_info: { has_more: false } };
    mockFetch.mockResolvedValue(mockOkResponse(body));

    const result = await fetchUserForumActivity("token");

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("/global-spaces/global-forum/threads");
    expect(url).toContain("mine=true");
    expect(result.data).toHaveLength(1);
  });

  it("passes limit and cursor params", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchUserForumActivity("token", { limit: 10, cursor: "xyz" });

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("limit=10");
    expect(url).toContain("cursor=xyz");
  });

  it("includes auth header", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchUserForumActivity("test-token");

    const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    const headers = options.headers as Record<string, string>;
    expect(headers["Authorization"]).toBe("Bearer test-token");
  });
});

describe("fetchUserSupportTickets", () => {
  it("fetches user's support tickets with mine=true", async () => {
    const body = { data: [THREAD_FIXTURE], page_info: { has_more: false } };
    mockFetch.mockResolvedValue(mockOkResponse(body));

    const result = await fetchUserSupportTickets("token");

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("/global-spaces/global-support/threads");
    expect(url).toContain("mine=true");
    expect(result.data).toHaveLength(1);
  });

  it("passes limit and cursor params", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchUserSupportTickets("token", { limit: 3, cursor: "c1" });

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("limit=3");
    expect(url).toContain("cursor=c1");
  });
});

describe("createForumThread", () => {
  it("creates a forum thread via POST", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    const result = await createForumThread("token", {
      title: "My Post",
      body: "Post body",
    });

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/global-spaces/global-forum/threads");
    expect(options.method).toBe("POST");
    const body = JSON.parse(options.body as string) as Record<string, unknown>;
    expect(body["title"]).toBe("My Post");
    expect(body["body"]).toBe("Post body");
    expect(result.title).toBe("Test Thread");
  });

  it("throws on 403 (tier enforcement)", async () => {
    mockFetch.mockResolvedValue(mockErrorResponse(403));
    await expect(createForumThread("token", { title: "x" })).rejects.toThrow();
  });
});

describe("createSupportTicket", () => {
  it("creates a support ticket via POST", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    const result = await createSupportTicket("token", {
      title: "Help me",
      body: "I need help",
      org_id: "org-1",
    });

    const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/global-spaces/global-support/threads");
    expect(options.method).toBe("POST");
    const body = JSON.parse(options.body as string) as Record<string, unknown>;
    expect(body["title"]).toBe("Help me");
    expect(body["org_id"]).toBe("org-1");
    expect(result.id).toBe("t1");
  });

  it("creates a ticket without org_id", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    await createSupportTicket("token", { title: "No org" });

    const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    const body = JSON.parse(options.body as string) as Record<string, unknown>;
    expect(body["title"]).toBe("No org");
    expect(body).not.toHaveProperty("org_id");
  });

  it("throws on API error", async () => {
    mockFetch.mockResolvedValue(mockErrorResponse(500));
    await expect(createSupportTicket("token", { title: "x" })).rejects.toThrow();
  });
});

describe("fetchGlobalLeads", () => {
  it("fetches leads from global-leads space", async () => {
    const body = { data: [THREAD_FIXTURE], page_info: { has_more: false } };
    mockFetch.mockResolvedValue(mockOkResponse(body));

    const result = await fetchGlobalLeads("token");

    const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/global-spaces/global-leads/threads");
    expect(options.method).toBe("GET");
    expect(result.data).toHaveLength(1);
    expect(result.data[0]?.id).toBe("t1");
  });

  it("does not append mine param when mine is false", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchGlobalLeads("token", { mine: false });

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).not.toContain("mine=");
  });

  it("appends mine=true when mine is true", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchGlobalLeads("token", { mine: true });

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("mine=true");
  });

  it("passes limit and cursor params", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchGlobalLeads("token", { limit: 20, cursor: "next-page" });

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("limit=20");
    expect(url).toContain("cursor=next-page");
  });

  it("includes auth header", async () => {
    mockFetch.mockResolvedValue(mockOkResponse({ data: [], page_info: { has_more: false } }));

    await fetchGlobalLeads("lead-token");

    const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    const headers = options.headers as Record<string, string>;
    expect(headers["Authorization"]).toBe("Bearer lead-token");
  });

  it("throws on API error", async () => {
    mockFetch.mockResolvedValue(mockErrorResponse(403));
    await expect(fetchGlobalLeads("token")).rejects.toThrow();
  });

  it("returns has_more and next_cursor from page_info", async () => {
    const body = {
      data: [THREAD_FIXTURE],
      page_info: { has_more: true, next_cursor: "abc123" },
    };
    mockFetch.mockResolvedValue(mockOkResponse(body));

    const result = await fetchGlobalLeads("token");

    expect(result.page_info.has_more).toBe(true);
    expect(result.page_info.next_cursor).toBe("abc123");
  });
});

describe("fetchGlobalThread", () => {
  it("fetches a single thread by slug from a global space", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    const result = await fetchGlobalThread("global-leads", "test-thread");

    const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/global-spaces/global-leads/threads/test-thread");
    expect(options.method).toBe("GET");
    expect(result.id).toBe("t1");
    expect(result.title).toBe("Test Thread");
  });

  it("works for any global space slug", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    await fetchGlobalThread("global-support", "support-slug");

    const [url] = mockFetch.mock.calls[0] as [string];
    expect(url).toContain("/global-spaces/global-support/threads/support-slug");
  });

  it("passes auth token when provided", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    await fetchGlobalThread("global-leads", "lead-slug", "auth-token");

    const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    const headers = options.headers as Record<string, string>;
    expect(headers["Authorization"]).toBe("Bearer auth-token");
  });

  it("fetches without auth token when not provided", async () => {
    mockFetch.mockResolvedValue(mockOkResponse(THREAD_FIXTURE));

    await fetchGlobalThread("global-leads", "lead-slug");

    const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    const headers = options.headers as Record<string, string>;
    expect(headers["Authorization"]).toBeUndefined();
  });

  it("throws on 404 when thread is not found", async () => {
    mockFetch.mockResolvedValue(mockErrorResponse(404));
    await expect(fetchGlobalThread("global-leads", "missing")).rejects.toThrow();
  });

  it("throws on API error", async () => {
    mockFetch.mockResolvedValue(mockErrorResponse(500));
    await expect(fetchGlobalThread("global-leads", "slug")).rejects.toThrow();
  });
});
