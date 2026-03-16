import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock api-client functions.
const mockBuildHeaders = vi.fn().mockReturnValue({ Authorization: "Bearer tok" });
const mockBuildUrl = vi
  .fn()
  .mockImplementation((path: string) => `http://localhost:8080/v1${path}`);
const mockParseResponse = vi.fn();

vi.mock("./api-client", () => ({
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
}));

// Mock global fetch.
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import { fetchTierInfo, fetchHomePreferences, saveHomePreferences } from "./tier-api";

describe("tier-api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("fetchTierInfo", () => {
    it("returns tier 1 for anonymous (no token)", async () => {
      const result = await fetchTierInfo(null);
      expect(result).toEqual({ tier: 1, sub_type: null });
      expect(mockFetch).not.toHaveBeenCalled();
    });

    it("returns tier 1 for undefined token", async () => {
      const result = await fetchTierInfo(undefined);
      expect(result).toEqual({ tier: 1, sub_type: null });
    });

    it("fetches tier info from API with valid token", async () => {
      const tierInfo = { tier: 3, sub_type: "owner", org_id: "org-1" };
      const mockResponse = { status: 200 };
      mockFetch.mockResolvedValue(mockResponse);
      mockParseResponse.mockResolvedValue(tierInfo);

      const result = await fetchTierInfo("test-token");

      expect(mockBuildUrl).toHaveBeenCalledWith("/me/tier");
      expect(mockBuildHeaders).toHaveBeenCalledWith("test-token");
      expect(mockFetch).toHaveBeenCalledWith(
        "http://localhost:8080/v1/me/tier",
        expect.objectContaining({
          method: "GET",
          cache: "no-store",
        }),
      );
      expect(mockParseResponse).toHaveBeenCalledWith(mockResponse);
      expect(result).toEqual(tierInfo);
    });

    it("returns DEFT employee tier with department", async () => {
      const tierInfo = { tier: 4, sub_type: null, deft_department: "sales" };
      mockFetch.mockResolvedValue({ status: 200 });
      mockParseResponse.mockResolvedValue(tierInfo);

      const result = await fetchTierInfo("deft-token");
      expect(result).toEqual(tierInfo);
    });

    it("returns platform admin tier", async () => {
      const tierInfo = { tier: 6, sub_type: null };
      mockFetch.mockResolvedValue({ status: 200 });
      mockParseResponse.mockResolvedValue(tierInfo);

      const result = await fetchTierInfo("admin-token");
      expect(result.tier).toBe(6);
    });
  });

  describe("fetchHomePreferences", () => {
    it("returns null when no preferences saved (404)", async () => {
      mockFetch.mockResolvedValue({ status: 404 });

      const result = await fetchHomePreferences("test-token");

      expect(result).toBeNull();
      expect(mockParseResponse).not.toHaveBeenCalled();
    });

    it("returns preferences when saved", async () => {
      const prefs = {
        user_id: "u1",
        tier: 2,
        layout: [{ widget_id: "my-profile", visible: true }],
      };
      mockFetch.mockResolvedValue({ status: 200 });
      mockParseResponse.mockResolvedValue(prefs);

      const result = await fetchHomePreferences("test-token");

      expect(mockBuildUrl).toHaveBeenCalledWith("/me/home-preferences");
      expect(result).toEqual(prefs);
    });

    it("passes auth token in headers", async () => {
      mockFetch.mockResolvedValue({ status: 404 });

      await fetchHomePreferences("my-token");

      expect(mockBuildHeaders).toHaveBeenCalledWith("my-token");
    });
  });

  describe("saveHomePreferences", () => {
    it("sends PUT with layout body", async () => {
      const layout = [{ widget_id: "my-profile", visible: true }];
      const saved = { user_id: "u1", tier: 2, layout };
      mockFetch.mockResolvedValue({ status: 200 });
      mockParseResponse.mockResolvedValue(saved);

      const result = await saveHomePreferences("test-token", layout);

      expect(mockBuildUrl).toHaveBeenCalledWith("/me/home-preferences");
      expect(mockFetch).toHaveBeenCalledWith(
        "http://localhost:8080/v1/me/home-preferences",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({ layout }),
        }),
      );
      expect(result).toEqual(saved);
    });

    it("passes auth token in headers", async () => {
      mockFetch.mockResolvedValue({ status: 200 });
      mockParseResponse.mockResolvedValue({});

      await saveHomePreferences("my-token", []);

      expect(mockBuildHeaders).toHaveBeenCalledWith("my-token");
    });
  });
});
