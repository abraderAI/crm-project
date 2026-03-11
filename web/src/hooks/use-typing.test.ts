import { renderHook, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useTyping } from "./use-typing";
import type { TypingPayload, WSMessage } from "@/lib/api-types";

describe("useTyping", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts with empty typing users", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));
    expect(result.current.typingUsers).toEqual([]);
  });

  it("handleLocalTyping calls sendTyping with threadId", () => {
    const sendTyping = vi.fn();
    const { result } = renderHook(() => useTyping({ threadId: "t-1", sendTyping }));

    act(() => {
      result.current.handleLocalTyping();
    });

    expect(sendTyping).toHaveBeenCalledWith("t-1");
  });

  it("throttles handleLocalTyping calls", () => {
    const sendTyping = vi.fn();
    const { result } = renderHook(() => useTyping({ threadId: "t-1", sendTyping }));

    act(() => {
      result.current.handleLocalTyping();
    });
    expect(sendTyping).toHaveBeenCalledTimes(1);

    // Second call within throttle window should be ignored.
    act(() => {
      vi.advanceTimersByTime(500);
      result.current.handleLocalTyping();
    });
    expect(sendTyping).toHaveBeenCalledTimes(1);

    // After throttle period, should fire again.
    act(() => {
      vi.advanceTimersByTime(2000);
      result.current.handleLocalTyping();
    });
    expect(sendTyping).toHaveBeenCalledTimes(2);
  });

  it("handleRemoteTyping adds a typing user", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));

    const msg: WSMessage<TypingPayload> = {
      type: "typing",
      channel: "thread:t-1",
      payload: { user_id: "u-2", user_name: "Alice", thread_id: "t-1" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleRemoteTyping(msg);
    });

    expect(result.current.typingUsers).toHaveLength(1);
    expect(result.current.typingUsers[0]?.userName).toBe("Alice");
  });

  it("ignores typing events from current user", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));

    const msg: WSMessage<TypingPayload> = {
      type: "typing",
      channel: "thread:t-1",
      payload: { user_id: "me", user_name: "Me", thread_id: "t-1" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleRemoteTyping(msg);
    });

    expect(result.current.typingUsers).toHaveLength(0);
  });

  it("ignores typing events from other threads", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));

    const msg: WSMessage<TypingPayload> = {
      type: "typing",
      channel: "thread:t-2",
      payload: { user_id: "u-2", user_name: "Bob", thread_id: "t-2" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleRemoteTyping(msg);
    });

    expect(result.current.typingUsers).toHaveLength(0);
  });

  it("updates existing typing user expiry", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));

    const makeMsg = (): WSMessage<TypingPayload> => ({
      type: "typing",
      channel: "thread:t-1",
      payload: { user_id: "u-2", user_name: "Alice", thread_id: "t-1" },
      timestamp: new Date().toISOString(),
    });

    act(() => {
      result.current.handleRemoteTyping(makeMsg());
    });
    const firstExpiry = result.current.typingUsers[0]?.expiresAt ?? 0;

    act(() => {
      vi.advanceTimersByTime(1000);
      result.current.handleRemoteTyping(makeMsg());
    });
    const secondExpiry = result.current.typingUsers[0]?.expiresAt ?? 0;

    expect(secondExpiry).toBeGreaterThan(firstExpiry);
    expect(result.current.typingUsers).toHaveLength(1);
  });

  it("cleans up expired typing users", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));

    const msg: WSMessage<TypingPayload> = {
      type: "typing",
      channel: "thread:t-1",
      payload: { user_id: "u-2", user_name: "Alice", thread_id: "t-1" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleRemoteTyping(msg);
    });
    expect(result.current.typingUsers).toHaveLength(1);

    // Advance past timeout (3s) + cleanup interval (1s).
    act(() => {
      vi.advanceTimersByTime(4100);
    });

    expect(result.current.typingUsers).toHaveLength(0);
  });

  it("resets typing users when threadId changes", () => {
    const { result, rerender } = renderHook(
      ({ threadId }: { threadId: string }) => useTyping({ threadId, currentUserId: "me" }),
      { initialProps: { threadId: "t-1" } },
    );

    const msg: WSMessage<TypingPayload> = {
      type: "typing",
      channel: "thread:t-1",
      payload: { user_id: "u-2", user_name: "Alice", thread_id: "t-1" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleRemoteTyping(msg);
    });
    expect(result.current.typingUsers).toHaveLength(1);

    rerender({ threadId: "t-2" });
    expect(result.current.typingUsers).toHaveLength(0);
  });

  it("uses user_id as fallback when user_name is missing", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1", currentUserId: "me" }));

    const msg: WSMessage<TypingPayload> = {
      type: "typing",
      channel: "thread:t-1",
      payload: { user_id: "u-2", thread_id: "t-1" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      result.current.handleRemoteTyping(msg);
    });

    expect(result.current.typingUsers[0]?.userName).toBe("u-2");
  });

  it("works without sendTyping callback", () => {
    const { result } = renderHook(() => useTyping({ threadId: "t-1" }));

    // Should not throw.
    act(() => {
      result.current.handleLocalTyping();
    });
  });
});
