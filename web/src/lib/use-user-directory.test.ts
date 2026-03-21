import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import { useUserDirectory } from "./use-user-directory";

describe("useUserDirectory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  it("resolves a user after loading", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: [
          {
            clerk_user_id: "user-1",
            email: "alice@example.com",
            display_name: "Alice",
            primary_org_name: "DEFT",
          },
        ],
        page_info: { has_more: false },
      }),
    });

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const resolved = result.current.resolve("user-1");
    expect(resolved).toEqual({ display_name: "Alice", org_name: "DEFT" });
  });

  it("format returns 'Name (Org)' for resolved user with org", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: [
          {
            clerk_user_id: "user-1",
            email: "alice@example.com",
            display_name: "Alice",
            primary_org_name: "DEFT",
          },
        ],
        page_info: { has_more: false },
      }),
    });

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.format("user-1")).toBe("Alice (DEFT)");
  });

  it("format returns name only when no org", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        data: [
          {
            clerk_user_id: "user-2",
            email: "bob@example.com",
            display_name: "Bob",
          },
        ],
        page_info: { has_more: false },
      }),
    });

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.format("user-2")).toBe("Bob");
  });

  it("format truncates unknown user IDs", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: [], page_info: { has_more: false } }),
    });

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.format("user_very_long_id_123")).toBe("user_very_lo…");
  });

  it("returns undefined for unknown user in resolve", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: [], page_info: { has_more: false } }),
    });

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.resolve("unknown")).toBeUndefined();
  });

  it("handles fetch failure gracefully", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Should fall back to empty directory without throwing.
    expect(result.current.resolve("user-1")).toBeUndefined();
  });

  it("handles null token gracefully", async () => {
    mockGetToken.mockResolvedValue(null);

    const { result } = renderHook(() => useUserDirectory());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.resolve("user-1")).toBeUndefined();
  });
});
