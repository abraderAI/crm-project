/**
 * Configuration for the voice agent sidecar.
 * Loaded from environment variables and the CRM internal API.
 */

/** AgentConfig holds all configuration for an agent session. */
export interface AgentConfig {
  /** CRM backend base URL for internal bridge API calls. */
  crmBaseUrl: string;
  /** Internal API key for authenticating bridge API calls. */
  internalApiKey: string;
  /** LiveKit project URL. */
  livekitUrl: string;
  /** LiveKit API key. */
  livekitApiKey: string;
  /** LiveKit API secret. */
  livekitApiSecret: string;
  /** Default STT model identifier. */
  defaultSttModel: string;
  /** Default TTS model identifier. */
  defaultTtsModel: string;
  /** System prompt for the LLM. */
  systemPrompt: string;
  /** Organization ID this agent is serving. */
  orgId: string;
}

/** Default STT model when none is configured. */
export const DEFAULT_STT_MODEL = "deepgram-nova-2";

/** Default TTS model when none is configured. */
export const DEFAULT_TTS_MODEL = "eleven-turbo-v2";

/** Default system prompt when none is configured. */
export const DEFAULT_SYSTEM_PROMPT =
  "You are a helpful voice assistant for our CRM platform. " +
  "Help callers with their questions, look up their contact information, " +
  "and summarize relevant threads. If you cannot help, offer to transfer " +
  "to a human agent.";

/**
 * Loads agent configuration from environment variables.
 * Falls back to sensible defaults where possible.
 */
export function loadConfig(env: Record<string, string | undefined> = process.env): AgentConfig {
  return {
    crmBaseUrl: env["CRM_BASE_URL"] ?? "http://localhost:8080",
    internalApiKey: env["INTERNAL_API_KEY"] ?? "",
    livekitUrl: env["LIVEKIT_URL"] ?? "",
    livekitApiKey: env["LIVEKIT_API_KEY"] ?? "",
    livekitApiSecret: env["LIVEKIT_API_SECRET"] ?? "",
    defaultSttModel: env["DEFAULT_STT_MODEL"] ?? DEFAULT_STT_MODEL,
    defaultTtsModel: env["DEFAULT_TTS_MODEL"] ?? DEFAULT_TTS_MODEL,
    systemPrompt: env["SYSTEM_PROMPT"] ?? DEFAULT_SYSTEM_PROMPT,
    orgId: env["ORG_ID"] ?? "",
  };
}

/**
 * Validates that all required configuration fields are present.
 * Returns an array of missing field names, or an empty array if valid.
 */
export function validateConfig(config: AgentConfig): string[] {
  const missing: string[] = [];
  if (!config.crmBaseUrl) missing.push("crmBaseUrl");
  if (!config.livekitUrl) missing.push("livekitUrl");
  if (!config.livekitApiKey) missing.push("livekitApiKey");
  if (!config.livekitApiSecret) missing.push("livekitApiSecret");
  return missing;
}
