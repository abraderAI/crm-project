import { describe, expect, it, vi, beforeEach } from "vitest";

import { fetchApiKeys, createApiKey, revokeApiKey } from "./settings-api";

// Mock global fetch.
const mockFetch = vi.fn();
global.fetch = mockFetch;

beforeEach(() => {
  vi.clearAllMocks();
});

describe("fetchApiKeys", () => {
  it("calls GET /v1/orgs/default/api-keys with auth header", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: [
          {
            id: "key-1",
            name: "Test Key",
            prefix: "deft_live_abc",
            created_at: "2026-01-01T00:00:00Z",
            last_used_at: null,
          },
        ],
      }),
    });

    const keys = await fetchApiKeys("test-token");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/v1/orgs/default/api-keys");
    expect(opts.method).toBe("GET");
    expect(opts.headers).toHaveProperty("Authorization", "Bearer test-token");
    expect(keys).toHaveLength(1);
    expect(keys[0]?.name).toBe("Test Key");
  });

  it("returns empty array when no keys exist", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: [] }),
    });

    const keys = await fetchApiKeys("test-token");
    expect(keys).toEqual([]);
  });

  it("throws on API error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({
        type: "about:blank",
        title: "Unauthorized",
        status: 401,
      }),
    });

    await expect(fetchApiKeys("bad-token")).rejects.toThrow();
  });
});

describe("createApiKey", () => {
  it("calls POST /v1/orgs/default/api-keys with name in body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        id: "key-2",
        name: "My Key",
        prefix: "deft_live_xyz",
        key: "deft_live_xyz_full_secret_key",
        created_at: "2026-01-01T00:00:00Z",
      }),
    });

    const result = await createApiKey("test-token", "My Key");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/v1/orgs/default/api-keys");
    expect(opts.method).toBe("POST");
    expect(opts.body).toBe(JSON.stringify({ name: "My Key" }));
    expect(result.key).toBe("deft_live_xyz_full_secret_key");
    expect(result.name).toBe("My Key");
  });

  it("throws on validation error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({
        type: "about:blank",
        title: "Bad Request",
        status: 400,
        detail: "Name is required",
      }),
    });

    await expect(createApiKey("test-token", "")).rejects.toThrow("Name is required");
  });
});

describe("revokeApiKey", () => {
  it("calls DELETE /v1/orgs/default/api-keys/{keyId}", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    await revokeApiKey("test-token", "key-1");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/v1/orgs/default/api-keys/key-1");
    expect(opts.method).toBe("DELETE");
    expect(opts.headers).toHaveProperty("Authorization", "Bearer test-token");
  });

  it("throws on 404 error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: async () => ({
        type: "about:blank",
        title: "Not Found",
        status: 404,
        detail: "API key not found",
      }),
    });

    await expect(revokeApiKey("test-token", "nonexistent")).rejects.toThrow("API key not found");
  });
});
