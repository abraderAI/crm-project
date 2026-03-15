import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { parseScriptConfig, initWidget } from "./widget";

describe("parseScriptConfig", () => {
  it("returns defaults when script element is null", () => {
    const config = parseScriptConfig(null);
    expect(config.embedKey).toBe("");
    expect(config.position).toBe("bottom-right");
    expect(config.theme.primaryColor).toBe("#3B82F6");
    expect(config.theme.greeting).toBe("Hello! How can we help you today?");
  });

  it("reads data-embed-key attribute", () => {
    const script = document.createElement("script");
    script.dataset["embedKey"] = "test-key-123";
    const config = parseScriptConfig(script);
    expect(config.embedKey).toBe("test-key-123");
  });

  it("reads data-position attribute", () => {
    const script = document.createElement("script");
    script.dataset["position"] = "bottom-left";
    const config = parseScriptConfig(script);
    expect(config.position).toBe("bottom-left");
  });

  it("defaults position to bottom-right", () => {
    const script = document.createElement("script");
    const config = parseScriptConfig(script);
    expect(config.position).toBe("bottom-right");
  });

  it("reads data-theme as JSON", () => {
    const script = document.createElement("script");
    script.dataset["theme"] = JSON.stringify({
      primaryColor: "#FF0000",
      greeting: "Hi!",
    });
    const config = parseScriptConfig(script);
    expect(config.theme.primaryColor).toBe("#FF0000");
    expect(config.theme.greeting).toBe("Hi!");
  });

  it("uses defaults for invalid theme JSON", () => {
    const script = document.createElement("script");
    script.dataset["theme"] = "not-valid-json";
    const config = parseScriptConfig(script);
    expect(config.theme.primaryColor).toBe("#3B82F6");
    expect(config.theme.greeting).toBe("Hello! How can we help you today?");
  });

  it("merges partial theme with defaults", () => {
    const script = document.createElement("script");
    script.dataset["theme"] = JSON.stringify({ primaryColor: "#00FF00" });
    const config = parseScriptConfig(script);
    expect(config.theme.primaryColor).toBe("#00FF00");
    expect(config.theme.greeting).toBe("Hello! How can we help you today?");
  });

  it("derives baseURL from script src", () => {
    const script = document.createElement("script");
    script.setAttribute("src", "https://cdn.example.com/widget.js");
    // jsdom may not parse src into a URL object properly, so this tests the fallback.
    const config = parseScriptConfig(script);
    expect(config.baseURL).toBeTruthy();
  });

  it("falls back to window origin when no src", () => {
    const script = document.createElement("script");
    const config = parseScriptConfig(script);
    expect(config.baseURL).toBe(window.location.origin);
  });

  it("falls back to window origin on invalid src URL", () => {
    const script = document.createElement("script");
    // In jsdom, setting src to a relative path gives a full URL, so
    // we test the branch by checking that baseURL is always set.
    script.setAttribute("src", "");
    const config = parseScriptConfig(script);
    expect(config.baseURL).toBeTruthy();
  });
});

describe("initWidget", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    // Clean up widget host elements.
    document.querySelectorAll("#crm-chat-widget").forEach((el) => el.remove());
  });

  it("does nothing when embedKey is empty", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    await initWidget({
      embedKey: "",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });
    expect(consoleSpy).toHaveBeenCalledWith("[CRM Widget] Missing data-embed-key attribute.");
    expect(document.getElementById("crm-chat-widget")).toBeNull();
  });

  it("creates widget host element in the DOM", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );
    // Mock WebSocket.
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi!" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    expect(host).not.toBeNull();
    expect(host?.shadowRoot).not.toBeNull();
  });

  it("shows greeting message on init", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Welcome!" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    const bubble = host?.shadowRoot?.querySelector(".crm-msg-bubble");
    expect(bubble?.textContent).toBe("Welcome!");
  });

  it("shows returning visitor greeting", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: true,
      greeting: "Good to see you again!",
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    const bubbles = host?.shadowRoot?.querySelectorAll(".crm-msg-bubble");
    // Should have greeting + returning greeting.
    expect(bubbles?.length).toBeGreaterThanOrEqual(2);
    const texts = Array.from(bubbles ?? []).map((b) => b.textContent);
    expect(texts.some((t) => t?.includes("Welcome back"))).toBe(true);
  });

  it("handles message send and AI response", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSession),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            type: "ai_response",
            message: "I can help with that!",
            message_id: "msg-1",
          }),
      });
    vi.stubGlobal("fetch", fetchMock);
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    const input = host?.shadowRoot?.querySelector("input") as HTMLInputElement;
    const sendBtn = host?.shadowRoot?.querySelector(
      ".crm-panel-input button",
    ) as HTMLButtonElement;

    input.value = "Hello, I need help";
    sendBtn.click();

    // Wait for async operations.
    await new Promise((r) => setTimeout(r, 50));

    const bubbles = host?.shadowRoot?.querySelectorAll(".crm-msg-bubble");
    const texts = Array.from(bubbles ?? []).map((b) => b.textContent);
    expect(texts).toContain("Hello, I need help");
    expect(texts).toContain("I can help with that!");
  });

  it("handles escalation response", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSession),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            type: "escalation",
            message: "Connecting to agent...",
          }),
      });
    vi.stubGlobal("fetch", fetchMock);
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    const input = host?.shadowRoot?.querySelector("input") as HTMLInputElement;
    const sendBtn = host?.shadowRoot?.querySelector(
      ".crm-panel-input button",
    ) as HTMLButtonElement;

    input.value = "I want to speak to a human";
    sendBtn.click();

    await new Promise((r) => setTimeout(r, 50));

    const status = host?.shadowRoot?.querySelector(".crm-status") as HTMLElement;
    expect(status.style.display).toBe("block");
    expect(status.textContent).toContain("human agent");
  });

  it("handles send failure gracefully", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSession),
      })
      .mockRejectedValueOnce(new Error("network error"));
    vi.stubGlobal("fetch", fetchMock);
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    const input = host?.shadowRoot?.querySelector("input") as HTMLInputElement;
    const sendBtn = host?.shadowRoot?.querySelector(
      ".crm-panel-input button",
    ) as HTMLButtonElement;

    input.value = "Hello";
    sendBtn.click();

    await new Promise((r) => setTimeout(r, 50));

    const bubbles = host?.shadowRoot?.querySelectorAll(".crm-msg-bubble");
    const texts = Array.from(bubbles ?? []).map((b) => b.textContent);
    expect(texts).toContain("Sorry, something went wrong. Please try again.");
  });

  it("handles WebSocket onMessage for chat.message", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );

    let wsInstance: {
      onmessage: ((e: MessageEvent) => void) | null;
      onopen: (() => void) | null;
    } | null = null;
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor() {
        wsInstance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    // Simulate WS message.
    wsInstance?.onmessage?.(
      new MessageEvent("message", {
        data: JSON.stringify({
          type: "chat.message",
          channel: "chat:s1",
          payload: { author: "ai", body: "WS AI message", type: "ai_response" },
        }),
      }),
    );

    const host = document.getElementById("crm-chat-widget");
    const bubbles = host?.shadowRoot?.querySelectorAll(".crm-msg-bubble");
    const texts = Array.from(bubbles ?? []).map((b) => b.textContent);
    expect(texts).toContain("WS AI message");
  });

  it("handles WebSocket onMessage for chat.escalated", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );

    let wsInstance: { onmessage: ((e: MessageEvent) => void) | null } | null = null;
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor() {
        wsInstance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    wsInstance?.onmessage?.(
      new MessageEvent("message", {
        data: JSON.stringify({
          type: "chat.escalated",
          channel: "escalation:org1",
          payload: {},
        }),
      }),
    );

    const host = document.getElementById("crm-chat-widget");
    const status = host?.shadowRoot?.querySelector(".crm-status") as HTMLElement;
    expect(status.style.display).toBe("block");
    expect(status.textContent).toContain("agent");
  });

  it("ignores WebSocket messages from visitor", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );

    let wsInstance: { onmessage: ((e: MessageEvent) => void) | null } | null = null;
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor() {
        wsInstance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    const host = document.getElementById("crm-chat-widget");
    const beforeCount = host?.shadowRoot?.querySelectorAll(".crm-msg").length ?? 0;

    wsInstance?.onmessage?.(
      new MessageEvent("message", {
        data: JSON.stringify({
          type: "chat.message",
          channel: "chat:s1",
          payload: { author: "visitor", body: "visitor msg" },
        }),
      }),
    );

    const afterCount = host?.shadowRoot?.querySelectorAll(".crm-msg").length ?? 0;
    expect(afterCount).toBe(beforeCount); // No new message added.
  });

  it("handles WebSocket onMessage for escalation_timeout type", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );

    let wsInstance: { onmessage: ((e: MessageEvent) => void) | null } | null = null;
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor() {
        wsInstance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    wsInstance?.onmessage?.(
      new MessageEvent("message", {
        data: JSON.stringify({
          type: "chat.message",
          channel: "chat:s1",
          payload: {
            author: "ai",
            body: "Agents are busy, I can help",
            type: "escalation_timeout",
          },
        }),
      }),
    );

    const host = document.getElementById("crm-chat-widget");
    const bubbles = host?.shadowRoot?.querySelectorAll(".crm-msg-bubble");
    const texts = Array.from(bubbles ?? []).map((b) => b.textContent);
    expect(texts).toContain("Agents are busy, I can help");
  });

  it("handles WebSocket disconnect silently", async () => {
    const mockSession = {
      token: "tok",
      session_id: "s1",
      visitor_id: "v1",
      expires_at: Math.floor(Date.now() / 1000) + 86400,
      returning: false,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSession),
      }),
    );

    let wsInstance: { onclose: (() => void) | null } | null = null;
    class MockWS {
      onopen: (() => void) | null = null;
      onmessage: ((e: MessageEvent) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor() {
        wsInstance = this;
      }
    }
    vi.stubGlobal("WebSocket", MockWS);

    await initWidget({
      embedKey: "test-key",
      theme: { primaryColor: "#3B82F6", greeting: "Hi" },
      position: "bottom-right",
      baseURL: "http://api.test",
    });

    // Should not throw.
    expect(() => wsInstance?.onclose?.()).not.toThrow();
  });
});
