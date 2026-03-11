import { renderHook, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useWebSocket } from "./use-websocket";
import type { WSMessage } from "@/lib/api-types";

/** Minimal mock WebSocket. */
class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  static instances: MockWebSocket[] = [];
  readyState = MockWebSocket.CONNECTING;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  url: string;
  send = vi.fn();
  close = vi.fn();

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  /** Simulate the connection opening. */
  simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  /** Simulate receiving a message. */
  simulateMessage(data: WSMessage): void {
    this.onmessage?.(new MessageEvent("message", { data: JSON.stringify(data) }));
  }

  /** Simulate a non-JSON message (e.g. pong). */
  simulateRawMessage(data: string): void {
    this.onmessage?.(new MessageEvent("message", { data }));
  }

  /** Simulate the connection closing. */
  simulateClose(): void {
    this.readyState = WebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }

  /** Simulate an error. */
  simulateError(): void {
    this.onerror?.(new Event("error"));
  }
}

describe("useWebSocket", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockWebSocket.instances = [];
    vi.stubGlobal("WebSocket", MockWebSocket);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("starts disconnected when token is null", () => {
    const { result } = renderHook(() => useWebSocket({ token: null }));
    expect(result.current.state).toBe("disconnected");
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("starts disconnected when enabled is false", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok", enabled: false }));
    expect(result.current.state).toBe("disconnected");
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("connects when token is provided and enabled", () => {
    const { result } = renderHook(() => useWebSocket({ token: "jwt-123" }));
    expect(result.current.state).toBe("connecting");
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0]?.url).toContain("/v1/ws?token=jwt-123");
  });

  it("transitions to connected on open", () => {
    const onStateChange = vi.fn();
    const { result } = renderHook(() => useWebSocket({ token: "tok", onStateChange }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    expect(result.current.state).toBe("connected");
    expect(onStateChange).toHaveBeenCalledWith("connected");
  });

  it("dispatches typed events to handlers", () => {
    const handler = vi.fn();
    const { result } = renderHook(() =>
      useWebSocket({
        token: "tok",
        onEvent: { "message.created": handler },
      }),
    );

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    const wsMsg: WSMessage = {
      type: "message.created",
      channel: "thread:t-1",
      payload: { id: "m-1", body: "Hello" },
      timestamp: new Date().toISOString(),
    };

    act(() => {
      MockWebSocket.instances[0]?.simulateMessage(wsMsg);
    });

    expect(handler).toHaveBeenCalledWith(wsMsg);
    expect(result.current.lastMessage).toEqual(wsMsg);
  });

  it("ignores non-JSON messages without error", () => {
    renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    // Should not throw.
    act(() => {
      MockWebSocket.instances[0]?.simulateRawMessage("pong");
    });
  });

  it("calls onError when WS error occurs", () => {
    const onError = vi.fn();
    renderHook(() => useWebSocket({ token: "tok", onError }));

    act(() => {
      MockWebSocket.instances[0]?.simulateError();
    });

    expect(onError).toHaveBeenCalledOnce();
  });

  it("send calls ws.send when connected", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    act(() => {
      result.current.send({ action: "subscribe", channel: "thread:t-1" });
    });

    expect(MockWebSocket.instances[0]?.send).toHaveBeenCalledWith(
      JSON.stringify({ action: "subscribe", channel: "thread:t-1" }),
    );
  });

  it("send does not call ws.send when not connected", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      result.current.send({ action: "subscribe", channel: "thread:t-1" });
    });

    // WS is in CONNECTING state, so send should not be called.
    expect(MockWebSocket.instances[0]?.send).not.toHaveBeenCalled();
  });

  it("subscribe sends subscribe command and tracks channel", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    act(() => {
      result.current.subscribe("board:b-1");
    });

    expect(MockWebSocket.instances[0]?.send).toHaveBeenCalledWith(
      JSON.stringify({ action: "subscribe", channel: "board:b-1" }),
    );
  });

  it("unsubscribe sends unsubscribe command", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    act(() => {
      result.current.unsubscribe("board:b-1");
    });

    expect(MockWebSocket.instances[0]?.send).toHaveBeenCalledWith(
      JSON.stringify({ action: "unsubscribe", channel: "board:b-1" }),
    );
  });

  it("attempts reconnect on close with exponential backoff", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    act(() => {
      MockWebSocket.instances[0]?.simulateClose();
    });

    expect(result.current.state).toBe("reconnecting");

    // After 1s delay, reconnect should create a new WS.
    act(() => {
      vi.advanceTimersByTime(1100);
    });

    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it("cleans up on unmount", () => {
    const { unmount } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    const ws = MockWebSocket.instances[0];
    unmount();

    expect(ws?.close).toHaveBeenCalled();
  });

  it("starts ping interval on connect", () => {
    renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    // Advance past ping interval (30s).
    act(() => {
      vi.advanceTimersByTime(30001);
    });

    // Ping sends a JSON "ping" action.
    expect(MockWebSocket.instances[0]?.send).toHaveBeenCalledWith(
      JSON.stringify({ action: "ping" }),
    );
  });

  it("lastMessage is null initially", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));
    expect(result.current.lastMessage).toBeNull();
  });

  it("resubscribes to channels after reconnect", () => {
    const { result } = renderHook(() => useWebSocket({ token: "tok" }));

    act(() => {
      MockWebSocket.instances[0]?.simulateOpen();
    });

    // Subscribe to a channel.
    act(() => {
      result.current.subscribe("thread:t-1");
    });

    // Simulate disconnect/reconnect.
    act(() => {
      MockWebSocket.instances[0]?.simulateClose();
    });

    act(() => {
      vi.advanceTimersByTime(1100);
    });

    // Simulate new connection opening.
    act(() => {
      MockWebSocket.instances[1]?.simulateOpen();
    });

    // Should have resent the subscribe command.
    const ws2 = MockWebSocket.instances[1];
    const calls = ws2?.send.mock.calls.map((c: unknown[]) => c[0]) ?? [];
    expect(calls).toContain(JSON.stringify({ action: "subscribe", channel: "thread:t-1" }));
  });
});
