import { describe, expect, it, vi, beforeEach } from "vitest";
import { upgradeToCustomer, type UpgradeResponse } from "./upgrade-api";
import { ApiError } from "./api-client";

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

describe("upgradeToCustomer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls POST /v1/me/upgrade with auth header", async () => {
    const body: UpgradeResponse = { tier: 3, message: "Upgraded to Customer" };
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(body),
    });

    const result = await upgradeToCustomer("test-token");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/v1/me/upgrade");
    expect(options.method).toBe("POST");
    expect(options.headers).toEqual(
      expect.objectContaining({
        Authorization: "Bearer test-token",
        "Content-Type": "application/json",
      }),
    );
    expect(result).toEqual(body);
  });

  it("returns the upgrade response on success", async () => {
    const body: UpgradeResponse = { tier: 3, message: "Success" };
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(body),
    });

    const result = await upgradeToCustomer("my-token");
    expect(result.tier).toBe(3);
    expect(result.message).toBe("Success");
  });

  it("throws ApiError on non-OK response", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      json: () =>
        Promise.resolve({
          type: "about:blank",
          title: "Forbidden",
          status: 403,
          detail: "Already upgraded",
        }),
    });

    await expect(upgradeToCustomer("bad-token")).rejects.toThrow(ApiError);
  });

  it("throws ApiError with correct status on server error", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: () =>
        Promise.resolve({
          type: "about:blank",
          title: "Internal Server Error",
          status: 500,
          detail: "Something went wrong",
        }),
    });

    try {
      await upgradeToCustomer("token");
      expect.fail("Should have thrown");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
    }
  });
});
