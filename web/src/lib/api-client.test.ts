import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  buildHeaders,
  buildUrl,
  clientMutate,
  parseResponse,
  serverFetch,
  serverFetchPaginated,
} from "./api-client";

describe("buildHeaders", () => {
  it("includes content-type and accept by default", () => {
    const headers = buildHeaders();
    expect(headers["Content-Type"]).toBe("application/json");
    expect(headers["Accept"]).toBe("application/json");
  });

  it("adds authorization header when token provided", () => {
    const headers = buildHeaders("test-token");
    expect(headers["Authorization"]).toBe("Bearer test-token");
  });

  it("does not add authorization when token is null", () => {
    const headers = buildHeaders(null);
    expect(headers["Authorization"]).toBeUndefined();
  });

  it("does not add authorization when token is undefined", () => {
    const headers = buildHeaders(undefined);
    expect(headers["Authorization"]).toBeUndefined();
  });

  it("merges extra headers", () => {
    const headers = buildHeaders(null, { "X-Custom": "value" });
    expect(headers["X-Custom"]).toBe("value");
    expect(headers["Content-Type"]).toBe("application/json");
  });
});

describe("buildUrl", () => {
  it("builds URL with v1 prefix", () => {
    const url = buildUrl("/orgs");
    expect(url).toBe("http://localhost:8080/v1/orgs");
  });

  it("adds query params", () => {
    const url = buildUrl("/orgs", { limit: "10", cursor: "abc" });
    expect(url).toContain("limit=10");
    expect(url).toContain("cursor=abc");
  });

  it("skips empty string params", () => {
    const url = buildUrl("/orgs", { limit: "10", cursor: "" });
    expect(url).toContain("limit=10");
    expect(url).not.toContain("cursor");
  });

  it("handles path with nested resources", () => {
    const url = buildUrl("/orgs/my-org/spaces");
    expect(url).toBe("http://localhost:8080/v1/orgs/my-org/spaces");
  });
});

describe("parseResponse", () => {
  it("returns parsed JSON for OK response", async () => {
    const data = { id: "123", name: "Test" };
    const response = new Response(JSON.stringify(data), { status: 200 });
    const result = await parseResponse<typeof data>(response);
    expect(result).toEqual(data);
  });

  it("throws ApiError with problem details for error response", async () => {
    const problem = {
      type: "https://api.deft.dev/errors/not-found",
      title: "Not Found",
      status: 404,
      detail: "Org not found",
    };
    const response = new Response(JSON.stringify(problem), {
      status: 404,
      headers: { "Content-Type": "application/problem+json" },
    });

    try {
      await parseResponse(response);
      expect.fail("Should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      const err = e as ApiError;
      expect(err.status).toBe(404);
      expect(err.problem.title).toBe("Not Found");
      expect(err.problem.detail).toBe("Org not found");
      expect(err.name).toBe("ApiError");
    }
  });

  it("handles non-JSON error response gracefully", async () => {
    const response = new Response("Internal Server Error", {
      status: 500,
      statusText: "Internal Server Error",
    });
    try {
      await parseResponse(response);
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      const err = e as ApiError;
      expect(err.status).toBe(500);
      expect(err.problem.title).toBe("Internal Server Error");
      expect(err.problem.detail).toBe("HTTP 500");
    }
  });

  it("handles empty statusText", async () => {
    const response = new Response("error", { status: 502, statusText: "" });
    try {
      await parseResponse(response);
    } catch (e) {
      const err = e as ApiError;
      expect(err.problem.title).toBe("Request failed");
    }
  });
});

describe("ApiError", () => {
  it("uses detail as message when available", () => {
    const err = new ApiError(400, {
      type: "about:blank",
      title: "Bad Request",
      status: 400,
      detail: "Invalid input",
    });
    expect(err.message).toBe("Invalid input");
    expect(err.name).toBe("ApiError");
    expect(err.status).toBe(400);
  });

  it("falls back to title when detail is undefined", () => {
    const err = new ApiError(403, {
      type: "about:blank",
      title: "Forbidden",
      status: 403,
    });
    expect(err.message).toBe("Forbidden");
  });
});

describe("serverFetch", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "1" }), { status: 200 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("calls fetch with correct URL and headers", async () => {
    await serverFetch("/orgs", { token: "jwt-token" });
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/orgs",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({
          Authorization: "Bearer jwt-token",
        }),
        cache: "no-store",
      }),
    );
  });

  it("returns parsed data", async () => {
    const result = await serverFetch<{ id: string }>("/orgs");
    expect(result).toEqual({ id: "1" });
  });
});

describe("serverFetchPaginated", () => {
  beforeEach(() => {
    const paginatedData = {
      data: [{ id: "1" }],
      page_info: { has_more: false },
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify(paginatedData), { status: 200 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("includes query params", async () => {
    await serverFetchPaginated("/orgs", { limit: "10" });
    const callUrl = (fetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[0] as string;
    expect(callUrl).toContain("limit=10");
  });

  it("returns paginated response", async () => {
    const result = await serverFetchPaginated<{ id: string }>("/orgs");
    expect(result.data).toHaveLength(1);
    expect(result.page_info.has_more).toBe(false);
  });
});

describe("clientMutate", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "new" }), { status: 201 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends POST with JSON body", async () => {
    const body = { name: "Test Org" };
    await clientMutate("POST", "/orgs", { token: "tok", body });
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/orgs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify(body),
        headers: expect.objectContaining({
          Authorization: "Bearer tok",
          "Content-Type": "application/json",
        }),
      }),
    );
  });

  it("sends DELETE without body", async () => {
    await clientMutate("DELETE", "/orgs/123");
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/orgs/123",
      expect.objectContaining({
        method: "DELETE",
        body: undefined,
      }),
    );
  });

  it("returns parsed response", async () => {
    const result = await clientMutate<{ id: string }>("POST", "/orgs", { body: { name: "x" } });
    expect(result).toEqual({ id: "new" });
  });
});
