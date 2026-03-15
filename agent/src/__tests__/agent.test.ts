import { describe, it, expect, vi } from "vitest";
import { AgentSession } from "../agent.js";
import { loadConfig, validateConfig, DEFAULT_STT_MODEL, DEFAULT_TTS_MODEL, DEFAULT_SYSTEM_PROMPT } from "../config.js";
import { createTools, type FetchFn } from "../tools.js";
import type { AgentConfig } from "../config.js";

// --- Config tests ---

describe("config", () => {
  it("loads defaults when env is empty", () => {
    const cfg = loadConfig({});
    expect(cfg.crmBaseUrl).toBe("http://localhost:8080");
    expect(cfg.defaultSttModel).toBe(DEFAULT_STT_MODEL);
    expect(cfg.defaultTtsModel).toBe(DEFAULT_TTS_MODEL);
    expect(cfg.systemPrompt).toBe(DEFAULT_SYSTEM_PROMPT);
    expect(cfg.internalApiKey).toBe("");
    expect(cfg.orgId).toBe("");
  });

  it("loads values from env", () => {
    const cfg = loadConfig({
      CRM_BASE_URL: "https://api.example.com",
      INTERNAL_API_KEY: "test-key",
      LIVEKIT_URL: "wss://lk.example.com",
      LIVEKIT_API_KEY: "api-key",
      LIVEKIT_API_SECRET: "api-secret",
      DEFAULT_STT_MODEL: "whisper-v3",
      DEFAULT_TTS_MODEL: "piper-v1",
      SYSTEM_PROMPT: "Custom prompt",
      ORG_ID: "org-123",
    });
    expect(cfg.crmBaseUrl).toBe("https://api.example.com");
    expect(cfg.internalApiKey).toBe("test-key");
    expect(cfg.livekitUrl).toBe("wss://lk.example.com");
    expect(cfg.livekitApiKey).toBe("api-key");
    expect(cfg.livekitApiSecret).toBe("api-secret");
    expect(cfg.defaultSttModel).toBe("whisper-v3");
    expect(cfg.defaultTtsModel).toBe("piper-v1");
    expect(cfg.systemPrompt).toBe("Custom prompt");
    expect(cfg.orgId).toBe("org-123");
  });

  it("validates required fields", () => {
    const cfg = loadConfig({});
    const missing = validateConfig(cfg);
    expect(missing).toContain("livekitUrl");
    expect(missing).toContain("livekitApiKey");
    expect(missing).toContain("livekitApiSecret");
    expect(missing).not.toContain("crmBaseUrl"); // Has default.
  });

  it("returns no missing when all required present", () => {
    const cfg = loadConfig({
      CRM_BASE_URL: "http://localhost",
      LIVEKIT_URL: "wss://lk",
      LIVEKIT_API_KEY: "key",
      LIVEKIT_API_SECRET: "secret",
    });
    expect(validateConfig(cfg)).toEqual([]);
  });
});

// --- Tools tests ---

function makeConfig(overrides: Partial<AgentConfig> = {}): AgentConfig {
  return {
    crmBaseUrl: "http://localhost:8080",
    internalApiKey: "test-key",
    livekitUrl: "wss://lk",
    livekitApiKey: "key",
    livekitApiSecret: "secret",
    defaultSttModel: DEFAULT_STT_MODEL,
    defaultTtsModel: DEFAULT_TTS_MODEL,
    systemPrompt: "Test prompt",
    orgId: "org-1",
    ...overrides,
  };
}

describe("tools", () => {
  it("lookupContact sends correct request", async () => {
    const mockFetch: FetchFn = vi.fn(async () =>
      new Response(JSON.stringify({ contacts: [{ id: "t-1", title: "Test", metadata: "{}" }] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const tools = createTools(makeConfig(), mockFetch);
    const results = await tools.lookupContact({ email: "test@example.com" });

    expect(results).toHaveLength(1);
    expect(results[0]?.id).toBe("t-1");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/internal/contacts/lookup?email=test%40example.com"),
      expect.objectContaining({
        headers: expect.objectContaining({ "X-Internal-Key": "test-key" }),
      }),
    );
  });

  it("lookupContact by phone", async () => {
    const mockFetch: FetchFn = vi.fn(async () =>
      new Response(JSON.stringify({ contacts: [] }), { status: 200 }),
    );
    const tools = createTools(makeConfig(), mockFetch);
    const results = await tools.lookupContact({ phone: "+15551234567" });

    expect(results).toEqual([]);
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("phone=%2B15551234567"),
      expect.any(Object),
    );
  });

  it("lookupContact throws on error", async () => {
    const mockFetch: FetchFn = vi.fn(async () => new Response("", { status: 500 }));
    const tools = createTools(makeConfig(), mockFetch);
    await expect(tools.lookupContact({ email: "x@y.com" })).rejects.toThrow("Contact lookup failed: 500");
  });

  it("getThreadSummary sends correct request", async () => {
    const summary = { id: "t-1", title: "Call", body: "Test", metadata: "{}", message_count: 3, created_at: "2024-01-01" };
    const mockFetch: FetchFn = vi.fn(async () =>
      new Response(JSON.stringify(summary), { status: 200 }),
    );
    const tools = createTools(makeConfig(), mockFetch);
    const result = await tools.getThreadSummary("t-1");

    expect(result.id).toBe("t-1");
    expect(result.message_count).toBe(3);
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/internal/threads/t-1/summary"),
      expect.any(Object),
    );
  });

  it("getThreadSummary throws on 404", async () => {
    const mockFetch: FetchFn = vi.fn(async () => new Response("", { status: 404 }));
    const tools = createTools(makeConfig(), mockFetch);
    await expect(tools.getThreadSummary("nonexistent")).rejects.toThrow("Thread summary failed: 404");
  });

  it("does not include X-Internal-Key when empty", async () => {
    const mockFetch: FetchFn = vi.fn(async () =>
      new Response(JSON.stringify({ contacts: [] }), { status: 200 }),
    );
    const tools = createTools(makeConfig({ internalApiKey: "" }), mockFetch);
    await tools.lookupContact({ email: "a@b.com" });

    const callArgs = (mockFetch as ReturnType<typeof vi.fn>).mock.calls[0] as [string, RequestInit | undefined];
    const headers = callArgs[1]?.headers as Record<string, string> | undefined;
    expect(headers?.["X-Internal-Key"]).toBeUndefined();
  });
});

// --- AgentSession tests ---

describe("AgentSession", () => {
  const validConfig = makeConfig();

  it("creates with defaults", () => {
    const session = new AgentSession({ config: validConfig });
    expect(session.active).toBe(false);
    expect(session.transcript).toEqual([]);
  });

  it("starts and stops", () => {
    const session = new AgentSession({ config: validConfig });
    session.start();
    expect(session.active).toBe(true);
    session.stop();
    expect(session.active).toBe(false);
  });

  it("throws on start with missing config", () => {
    const session = new AgentSession({ config: loadConfig({}) });
    expect(() => session.start()).toThrow("Missing required config");
  });

  it("processes utterance without LLM", async () => {
    const session = new AgentSession({ config: validConfig });
    session.start();
    const response = await session.processUtterance("Hello");
    expect(response).toContain("Hello");
    expect(session.transcript).toHaveLength(2);
    expect(session.transcript[0]?.speaker).toBe("caller");
    expect(session.transcript[1]?.speaker).toBe("agent");
  });

  it("processes utterance with mock LLM", async () => {
    const mockLLM = {
      chat: vi.fn(async (_system: string, user: string) => `LLM response to: ${user}`),
    };
    const session = new AgentSession({ config: validConfig, llm: mockLLM });
    session.start();
    const response = await session.processUtterance("What is my balance?");
    expect(response).toBe("LLM response to: What is my balance?");
    expect(mockLLM.chat).toHaveBeenCalledWith(validConfig.systemPrompt, "What is my balance?");
  });

  it("throws when processing utterance while inactive", async () => {
    const session = new AgentSession({ config: validConfig });
    await expect(session.processUtterance("Hi")).rejects.toThrow("Session is not active");
  });

  it("detects escalation intent", () => {
    const session = new AgentSession({ config: validConfig });
    expect(session.detectEscalationIntent("I want to speak to a human")).toBe(true);
    expect(session.detectEscalationIntent("Can you transfer me?")).toBe(true);
    expect(session.detectEscalationIntent("What is the weather?")).toBe(false);
    expect(session.detectEscalationIntent("I need a real person")).toBe(true);
    expect(session.detectEscalationIntent("talk to a person please")).toBe(true);
    expect(session.detectEscalationIntent("connect me to a supervisor")).toBe(true);
    expect(session.detectEscalationIntent("I want a manager")).toBe(true);
  });

  it("formats transcript", async () => {
    const session = new AgentSession({ config: validConfig });
    session.start();
    await session.processUtterance("Hello");
    await session.processUtterance("How are you?");
    const formatted = session.getFormattedTranscript();
    expect(formatted).toContain("[caller] Hello");
    expect(formatted).toContain("[agent]");
    expect(formatted).toContain("[caller] How are you?");
  });

  it("lookupContact delegates to tools", async () => {
    const mockFetch: FetchFn = vi.fn(async () =>
      new Response(JSON.stringify({ contacts: [{ id: "c-1", title: "Contact", metadata: "{}" }] }), {
        status: 200,
      }),
    );
    const session = new AgentSession({ config: validConfig, fetchFn: mockFetch });
    const contacts = await session.lookupContact({ email: "test@test.com" });
    expect(contacts).toHaveLength(1);
  });

  it("getThreadSummary delegates to tools", async () => {
    const summary = { id: "t-1", title: "Thread", body: "", metadata: "{}", message_count: 0, created_at: "" };
    const mockFetch: FetchFn = vi.fn(async () =>
      new Response(JSON.stringify(summary), { status: 200 }),
    );
    const session = new AgentSession({ config: validConfig, fetchFn: mockFetch });
    const result = await session.getThreadSummary("t-1");
    expect(result.id).toBe("t-1");
  });
});
