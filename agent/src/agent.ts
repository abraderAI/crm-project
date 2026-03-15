/**
 * Voice agent sidecar entry point.
 * Provides AgentSession class with configurable STT, LLM, and TTS.
 */

import { type AgentConfig, loadConfig, validateConfig } from "./config.js";
import { createTools, type FetchFn } from "./tools.js";

/** STT (Speech-to-Text) provider interface. */
export interface STTProvider {
  model: string;
  transcribe(audio: Uint8Array): Promise<string>;
}

/** TTS (Text-to-Speech) provider interface. */
export interface TTSProvider {
  model: string;
  synthesize(text: string): Promise<Uint8Array>;
}

/** LLM provider interface — calls CRM's LLMProvider via internal API. */
export interface LLMProvider {
  chat(systemPrompt: string, userMessage: string): Promise<string>;
}

/** TranscriptEntry for tracking conversation. */
export interface TranscriptEntry {
  speaker: "agent" | "caller";
  text: string;
  timestamp: number;
}

/** AgentSessionOptions configures the agent session. */
export interface AgentSessionOptions {
  config?: AgentConfig;
  stt?: STTProvider;
  tts?: TTSProvider;
  llm?: LLMProvider;
  fetchFn?: FetchFn;
}

/**
 * AgentSession manages a voice call session with configurable
 * STT (LiveKit Inference), LLM (CRM's LLMProvider), and TTS (LiveKit Inference).
 */
export class AgentSession {
  readonly config: AgentConfig;
  readonly transcript: TranscriptEntry[] = [];
  private readonly tools;
  private readonly stt?: STTProvider;
  private readonly tts?: TTSProvider;
  private readonly llm?: LLMProvider;
  private _active = false;

  constructor(options: AgentSessionOptions = {}) {
    this.config = options.config ?? loadConfig();
    this.stt = options.stt;
    this.tts = options.tts;
    this.llm = options.llm;
    this.tools = createTools(this.config, options.fetchFn);
  }

  /** Returns whether the session is currently active. */
  get active(): boolean {
    return this._active;
  }

  /**
   * Starts the agent session. Validates configuration before proceeding.
   * Throws if required config fields are missing.
   */
  start(): void {
    const missing = validateConfig(this.config);
    if (missing.length > 0) {
      throw new Error(`Missing required config: ${missing.join(", ")}`);
    }
    this._active = true;
  }

  /** Stops the agent session. */
  stop(): void {
    this._active = false;
  }

  /**
   * Processes a caller utterance through the LLM and returns the agent response.
   * Also records both entries in the transcript.
   */
  async processUtterance(callerText: string): Promise<string> {
    if (!this._active) {
      throw new Error("Session is not active");
    }

    this.transcript.push({
      speaker: "caller",
      text: callerText,
      timestamp: Date.now(),
    });

    let response: string;
    if (this.llm) {
      response = await this.llm.chat(this.config.systemPrompt, callerText);
    } else {
      response = `I heard: "${callerText}". Let me help you with that.`;
    }

    this.transcript.push({
      speaker: "agent",
      text: response,
      timestamp: Date.now(),
    });

    return response;
  }

  /**
   * Checks if the caller's utterance indicates escalation intent.
   * Returns true for common escalation phrases.
   */
  detectEscalationIntent(text: string): boolean {
    const escalationPhrases = [
      "speak to a human",
      "talk to a person",
      "transfer me",
      "human agent",
      "real person",
      "speak to someone",
      "talk to someone",
      "connect me",
      "supervisor",
      "manager",
    ];
    const lower = text.toLowerCase();
    return escalationPhrases.some((phrase) => lower.includes(phrase));
  }

  /**
   * Looks up a contact using the CRM bridge API.
   */
  async lookupContact(params: { email?: string; phone?: string }) {
    return this.tools.lookupContact(params);
  }

  /**
   * Gets a thread summary using the CRM bridge API.
   */
  async getThreadSummary(threadId: string) {
    return this.tools.getThreadSummary(threadId);
  }

  /**
   * Returns the compiled transcript as a formatted string.
   */
  getFormattedTranscript(): string {
    return this.transcript
      .map((e) => `[${e.speaker}] ${e.text}`)
      .join("\n");
  }
}
