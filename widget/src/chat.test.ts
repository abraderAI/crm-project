import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  storeSession,
  loadSession,
  clearSession,
  createSession,
  sendMessage,
  connectWebSocket,
  type ChatSessionData,
  type ChatClientConfig,
} from "./chat";

const mockSession: ChatSessionData = {
  token: "test-token-123",
  session_id: "sess-abc",
  visitor_id: "vis-def",
  expires_at: Math.floor(Date.now() / 1000) + 86400, // 24h from now.
  returning: false,
  greeting: "Hello!",
};

describe("storeSession / loadSession / clearSession", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("stores and loads a session", () => {
    storeSession(mockSession);
    const loaded = loadSession();
    expect(loaded).not.toBeNull();
    expect(loaded?.token).toBe("test-token-123");
    expect(loaded?.session_id).toBe("sess-abc");
  });

  it("returns null when no session stored", () => {
    expect(loadSession()).toBeNull();
  });

  it("returns null for expired session", () => {
    const expired: ChatSessionData = {
      ...mockSession,
      expires_at: Math.floor(Date.now() / 1000) - 3600, // 1h ago.
    };
    storeSession(expired);
    expect(loadSession()).toBeNull();
  });

  it("returns null for session expiring within 5 minutes", () => {
    const expiringSoon: ChatSessionData = {
      ...mockSession,
      expires_at: Math.floor(Date.now() / 1000) + 60, // 1 minute from now.
    };
    storeSession(expiringSoon);
    expect(loadSession()).toBeNull();
  });

  it("clears session", () => {
    storeSession(mockSession);
    clearSession();
    expect(loadSession()).toBeNull();
  });

  it("handles localStorage exceptions gracefully", () => {
    const origSetItem = localStorage.setItem;
    localStorage.setItem = () => {
      throw new Error("quota exceeded");
    };
    // Should not throw.
    expect(() => storeSession(mockSession)).not.toThrow();
    localStorage.setItem = origSetItem;
  });

  it("handles corrupt stored data", () => {
    localStorage.setItem("crm_chat_session", "not-valid-json");
    expect(loadSession()).toBeNull();
  });
});

describe("createSession", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns existing session from localStorage", async () => {
    storeSession(mockSession);
    const session = await createSession("http://api.test", "key", "fp");
    expect(session.token).toBe("test-token-123");
  });

  it("calls fetch when no stored session", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockSession),
    });
    vi.stubGlobal("fetch", fetchMock);

    const session = await createSession("http://api.test", "embed-key", "fp-hash");
    expect(fetchMock).toHaveBeenCalledWith("http://api.test/v1/chat/session", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        embed_key: "embed-key",
        fingerprint_hash: "fp-hash",
      }),
    });
    expect(session.token).toBe("test-token-123");
  });

  it("stores session after successful creation", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );

    await createSession("http://api.test", "key", "fp");
    const loaded = loadSession();
    expect(loaded).not.toBeNull();
    expect(loaded?.session_id).toBe("sess-abc");
  });

  it("throws on non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () => Promise.resolve("invalid embed key"),
      }),
    );

    await expect(createSession("http://api.test", "bad-key", "fp")).rejects.toThrow(
      "Session creation failed (401)",
    );
  });
});

describe("sendMessage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("sends message with correct headers", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({ type: "ai_response", message: "Hello!", message_id: "msg-1" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const resp = await sendMessage("http://api.test", "token-123", "Hello");
    expect(fetchMock).toHaveBeenCalledWith("http://api.test/v1/chat/message", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer token-123",
      },
      body: JSON.stringify({ message: "Hello" }),
    });
    expect(resp.type).toBe("ai_response");
    expect(resp.message).toBe("Hello!");
  });

  it("handles escalation response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            type: "escalation",
            message: "Connecting to agent...",
          }),
      }),
    );

    const resp = await sendMessage("http://api.test", "token", "speak to human");
    expect(resp.type).toBe("escalation");
  });

  it("throws on non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: () => Promise.resolve("server error"),
      }),
    );

    await expect(sendMessage("http://api.test", "token", "hi")).rejects.toThrow(
      "Message send failed (500)",
    );
  });
});

describe("connectWebSocket", () => {
  it("returns null when WebSocket is undefined", () => {
    const origWS = globalThis.WebSocket;
    // @ts-expect-error - testing absence of WebSocket
    delete globalThis.WebSocket;
    const config: ChatClientConfig = {
      baseURL: "http://api.test",
      embedKey: "key",
    };
    const ws = connectWebSocket(config, "token");
    expect(ws).toBeNull();
    globalThis.WebSocket = origWS;
  });

  it("creates WebSocket with correct URL", () => {
    const instances: Array<{ url: string }> = [];
    class MockWS {
      url: string;
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(url: string) {
        this.url = url;
        instances.push(this);
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    const config: ChatClientConfig = {
      baseURL: "http://api.test",
      embedKey: "key",
    };
    connectWebSocket(config, "my-token");
    expect(instances).toHaveLength(1);
    expect(instances[0]?.url).toBe("ws://api.test/v1/ws?token=my-token");
    vi.restoreAllMocks();
  });

  it("converts https to wss", () => {
    const instances: Array<{ url: string }> = [];
    class MockWS {
      url: string;
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(url: string) {
        this.url = url;
        instances.push(this);
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    const config: ChatClientConfig = {
      baseURL: "https://api.test",
      embedKey: "key",
    };
    connectWebSocket(config, "token");
    expect(instances).toHaveLength(1);
    expect(instances[0]?.url).toBe("wss://api.test/v1/ws?token=token");
    vi.restoreAllMocks();
  });

  it("calls onConnectionChange(true) on open", () => {
    const onConnectionChange = vi.fn();
    let instance: { onopen: (() => void) | null } | null = null;

    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(_url: string) {
        instance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    connectWebSocket(
      { baseURL: "http://api.test", embedKey: "key", onConnectionChange },
      "token",
    );
    instance?.onopen?.();
    expect(onConnectionChange).toHaveBeenCalledWith(true);
    vi.restoreAllMocks();
  });

  it("calls onMessage on valid message", () => {
    const onMessage = vi.fn();
    let instance: { onmessage: ((e: MessageEvent) => void) | null } | null = null;

    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(_url: string) {
        instance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    connectWebSocket({ baseURL: "http://api.test", embedKey: "key", onMessage }, "token");
    instance?.onmessage?.(
      new MessageEvent("message", {
        data: JSON.stringify({ type: "chat.message", channel: "ch", payload: {} }),
      }),
    );
    expect(onMessage).toHaveBeenCalledWith({
      type: "chat.message",
      channel: "ch",
      payload: {},
    });
    vi.restoreAllMocks();
  });

  it("ignores malformed WebSocket messages", () => {
    const onMessage = vi.fn();
    let instance: { onmessage: ((e: MessageEvent) => void) | null } | null = null;

    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(_url: string) {
        instance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    connectWebSocket({ baseURL: "http://api.test", embedKey: "key", onMessage }, "token");
    // Should not throw.
    instance?.onmessage?.(new MessageEvent("message", { data: "not-json" }));
    expect(onMessage).not.toHaveBeenCalled();
    vi.restoreAllMocks();
  });

  it("calls onConnectionChange(false) on close", () => {
    const onConnectionChange = vi.fn();
    let instance: { onclose: (() => void) | null } | null = null;

    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(_url: string) {
        instance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    connectWebSocket(
      { baseURL: "http://api.test", embedKey: "key", onConnectionChange },
      "token",
    );
    instance?.onclose?.();
    expect(onConnectionChange).toHaveBeenCalledWith(false);
    vi.restoreAllMocks();
  });

  it("calls onConnectionChange(false) on error", () => {
    const onConnectionChange = vi.fn();
    let instance: { onerror: (() => void) | null } | null = null;

    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(_url: string) {
        instance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    connectWebSocket(
      { baseURL: "http://api.test", embedKey: "key", onConnectionChange },
      "token",
    );
    instance?.onerror?.();
    expect(onConnectionChange).toHaveBeenCalledWith(false);
    vi.restoreAllMocks();
  });
});
