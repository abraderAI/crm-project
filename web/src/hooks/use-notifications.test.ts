import { renderHook, act, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useNotifications } from "./use-notifications";
import type { Notification, WSMessage } from "@/lib/api-types";

const makeNotif = (overrides: Partial<Notification> = {}): Notification => ({
  id: "n-1",
  user_id: "u-1",
  type: "message",
  title: "Test notification",
  is_read: false,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  ...overrides,
});

describe("useNotifications", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [makeNotif(), makeNotif({ id: "n-2", is_read: true })],
            page_info: { has_more: false },
          }),
          { status: 200 },
        ),
      ),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fetches notifications on mount when enabled and token provided", async () => {
    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.notifications).toHaveLength(2);
    });
    expect(result.current.loading).toBe(false);
    expect(fetch).toHaveBeenCalledOnce();
  });

  it("does not fetch when token is null", () => {
    renderHook(() => useNotifications({ token: null }));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("does not fetch when enabled is false", () => {
    renderHook(() => useNotifications({ token: "tok", enabled: false }));
    expect(fetch).not.toHaveBeenCalled();
  });

  it("computes unreadCount correctly", async () => {
    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.unreadCount).toBe(1);
    });
  });

  it("markRead updates notification optimistically", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              data: [makeNotif()],
              page_info: { has_more: false },
            }),
            { status: 200 },
          ),
        )
        .mockResolvedValueOnce(new Response("{}", { status: 200 })),
    );

    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.notifications).toHaveLength(1);
    });

    await act(async () => {
      await result.current.markRead("n-1");
    });

    expect(result.current.notifications[0]?.is_read).toBe(true);
    expect(result.current.unreadCount).toBe(0);
  });

  it("markAllRead marks all as read", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              data: [makeNotif({ id: "n-1" }), makeNotif({ id: "n-2" })],
              page_info: { has_more: false },
            }),
            { status: 200 },
          ),
        )
        .mockResolvedValueOnce(new Response("{}", { status: 200 })),
    );

    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.notifications).toHaveLength(2);
    });

    await act(async () => {
      await result.current.markAllRead();
    });

    expect(result.current.notifications.every((n) => n.is_read)).toBe(true);
    expect(result.current.unreadCount).toBe(0);
  });

  it("handleWSNotification adds a new notification", async () => {
    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.notifications).toHaveLength(2);
    });

    const newNotif = makeNotif({ id: "n-3", title: "Real-time push" });
    const wsMsg: WSMessage<Notification> = {
      type: "notification",
      channel: "user:u-1",
      payload: newNotif,
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleWSNotification(wsMsg);
    });

    expect(result.current.notifications).toHaveLength(3);
    expect(result.current.notifications[0]?.id).toBe("n-3");
  });

  it("handleWSNotification avoids duplicates", async () => {
    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.notifications).toHaveLength(2);
    });

    // Send duplicate of existing notification.
    const wsMsg: WSMessage<Notification> = {
      type: "notification",
      channel: "user:u-1",
      payload: makeNotif({ id: "n-1" }),
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleWSNotification(wsMsg);
    });

    expect(result.current.notifications).toHaveLength(2);
  });

  it("handles fetch error gracefully", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Network error")));

    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.error).toBe("Network error");
    });
    expect(result.current.loading).toBe(false);
    expect(result.current.notifications).toEqual([]);
  });

  it("refresh re-fetches notifications", async () => {
    const { result } = renderHook(() => useNotifications({ token: "tok" }));

    await waitFor(() => {
      expect(result.current.notifications).toHaveLength(2);
    });

    await act(async () => {
      await result.current.refresh();
    });

    // fetch called twice: initial + refresh.
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("markRead does nothing when token is null", async () => {
    const { result } = renderHook(() => useNotifications({ token: null }));

    await act(async () => {
      await result.current.markRead("n-1");
    });
    // Should not throw.
  });

  it("markAllRead does nothing when token is null", async () => {
    const { result } = renderHook(() => useNotifications({ token: null }));

    await act(async () => {
      await result.current.markAllRead();
    });
    // Should not throw.
  });
});
