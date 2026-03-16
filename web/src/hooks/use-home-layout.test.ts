import { renderHook, waitFor, act } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { useHomeLayout } from "./use-home-layout";
import { WIDGET_IDS } from "@/lib/default-layouts";

// Mock tier-api.
const mockFetchHomePreferences = vi.fn();
const mockSaveHomePreferences = vi.fn();
vi.mock("@/lib/tier-api", () => ({
  fetchHomePreferences: (...args: unknown[]) => mockFetchHomePreferences(...args),
  saveHomePreferences: (...args: unknown[]) => mockSaveHomePreferences(...args),
}));

describe("useHomeLayout", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns default layout for tier 1 anonymous user (no token)", async () => {
    const { result } = renderHook(() => useHomeLayout(1, null));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.layout).toHaveLength(3);
    expect(result.current.layout[0]?.widget_id).toBe(WIDGET_IDS.DOCS_HIGHLIGHTS);
    expect(result.current.isCustomized).toBe(false);
    expect(mockFetchHomePreferences).not.toHaveBeenCalled();
  });

  it("returns default layout for fresh user (404 preferences)", async () => {
    mockFetchHomePreferences.mockResolvedValue(null);
    const { result } = renderHook(() => useHomeLayout(2, "token"));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.layout).toHaveLength(4);
    expect(result.current.layout[0]?.widget_id).toBe(WIDGET_IDS.MY_PROFILE);
    expect(result.current.isCustomized).toBe(false);
  });

  it("returns saved layout for returning user", async () => {
    const savedLayout = [
      { widget_id: WIDGET_IDS.MY_FORUM_ACTIVITY, visible: true },
      { widget_id: WIDGET_IDS.MY_PROFILE, visible: true },
    ];
    mockFetchHomePreferences.mockResolvedValue({
      user_id: "u1",
      tier: 2,
      layout: savedLayout,
    });
    const { result } = renderHook(() => useHomeLayout(2, "token"));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.layout).toEqual(savedLayout);
    expect(result.current.isCustomized).toBe(true);
  });

  it("falls back to default on fetch error", async () => {
    mockFetchHomePreferences.mockRejectedValue(new Error("Network error"));
    const { result } = renderHook(() => useHomeLayout(2, "token"));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.layout[0]?.widget_id).toBe(WIDGET_IDS.MY_PROFILE);
    expect(result.current.isCustomized).toBe(false);
  });

  it("uses department-specific default for tier 4", async () => {
    mockFetchHomePreferences.mockResolvedValue(null);
    const { result } = renderHook(() => useHomeLayout(4, "token", "support"));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.layout[0]?.widget_id).toBe(WIDGET_IDS.TICKET_QUEUE);
  });

  it("updateLayout persists to server", async () => {
    mockFetchHomePreferences.mockResolvedValue(null);
    mockSaveHomePreferences.mockResolvedValue({});
    const { result } = renderHook(() => useHomeLayout(2, "token"));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const newLayout = [{ widget_id: WIDGET_IDS.UPGRADE_CTA, visible: true }];
    await act(async () => {
      await result.current.updateLayout(newLayout);
    });

    expect(result.current.layout).toEqual(newLayout);
    expect(result.current.isCustomized).toBe(true);
    expect(mockSaveHomePreferences).toHaveBeenCalledWith("token", newLayout);
  });

  it("updateLayout does not persist when no token", async () => {
    const { result } = renderHook(() => useHomeLayout(1, null));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const newLayout = [{ widget_id: WIDGET_IDS.GET_STARTED, visible: false }];
    await act(async () => {
      await result.current.updateLayout(newLayout);
    });

    expect(result.current.layout).toEqual(newLayout);
    expect(mockSaveHomePreferences).not.toHaveBeenCalled();
  });

  it("resetToDefault restores tier default and persists", async () => {
    const savedLayout = [{ widget_id: WIDGET_IDS.UPGRADE_CTA, visible: true }];
    mockFetchHomePreferences.mockResolvedValue({
      user_id: "u1",
      tier: 2,
      layout: savedLayout,
    });
    mockSaveHomePreferences.mockResolvedValue({});

    const { result } = renderHook(() => useHomeLayout(2, "token"));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.isCustomized).toBe(true);

    await act(async () => {
      await result.current.resetToDefault();
    });

    expect(result.current.layout[0]?.widget_id).toBe(WIDGET_IDS.MY_PROFILE);
    expect(result.current.isCustomized).toBe(false);
    expect(mockSaveHomePreferences).toHaveBeenCalled();
  });

  it("shows loading state initially", () => {
    mockFetchHomePreferences.mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useHomeLayout(2, "token"));
    expect(result.current.isLoading).toBe(true);
  });
});
