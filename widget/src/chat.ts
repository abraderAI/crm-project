/**
 * Chat client module.
 * Handles HTTP session creation, message sending, and WebSocket
 * connection for real-time updates from the backend.
 */

/** Session data returned from the backend. */
export interface ChatSessionData {
  token: string;
  session_id: string;
  visitor_id: string;
  expires_at: number;
  returning: boolean;
  greeting?: string;
}

/** Response from the chat message endpoint. */
export interface ChatMessageResponse {
  type: "ai_response" | "escalation";
  message: string;
  message_id?: string;
}

/** WebSocket message from the server. */
export interface WSMessage {
  type: string;
  channel: string;
  payload: Record<string, unknown>;
}

/** Configuration for the chat client. */
export interface ChatClientConfig {
  baseURL: string;
  embedKey: string;
  onMessage?: (msg: WSMessage) => void;
  onConnectionChange?: (connected: boolean) => void;
}

const TOKEN_KEY = "crm_chat_token";
const SESSION_KEY = "crm_chat_session";

/** Store session data in localStorage. */
export function storeSession(data: ChatSessionData): void {
  try {
    localStorage.setItem(TOKEN_KEY, data.token);
    localStorage.setItem(SESSION_KEY, JSON.stringify(data));
  } catch {
    // localStorage may be unavailable (e.g. private browsing).
  }
}

/** Load stored session from localStorage. */
export function loadSession(): ChatSessionData | null {
  try {
    const raw = localStorage.getItem(SESSION_KEY);
    if (!raw) return null;
    const data = JSON.parse(raw) as ChatSessionData;
    // Check if token is still valid (with 5-minute buffer).
    if (data.expires_at * 1000 < Date.now() + 5 * 60 * 1000) {
      clearSession();
      return null;
    }
    return data;
  } catch {
    return null;
  }
}

/** Clear stored session from localStorage. */
export function clearSession(): void {
  try {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(SESSION_KEY);
  } catch {
    // Ignore.
  }
}

/** Create or restore a chat session. */
export async function createSession(
  baseURL: string,
  embedKey: string,
  fingerprintHash: string,
): Promise<ChatSessionData> {
  // Try to restore existing session first.
  const existing = loadSession();
  if (existing) return existing;

  const resp = await fetch(`${baseURL}/v1/chat/session`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      embed_key: embedKey,
      fingerprint_hash: fingerprintHash,
    }),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Session creation failed (${resp.status}): ${text}`);
  }

  const data = (await resp.json()) as ChatSessionData;
  storeSession(data);
  return data;
}

/** Send a chat message and receive the AI/escalation response. */
export async function sendMessage(
  baseURL: string,
  token: string,
  message: string,
): Promise<ChatMessageResponse> {
  const resp = await fetch(`${baseURL}/v1/chat/message`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ message }),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Message send failed (${resp.status}): ${text}`);
  }

  return (await resp.json()) as ChatMessageResponse;
}

/** Create a WebSocket connection for real-time updates. */
export function connectWebSocket(config: ChatClientConfig, token: string): WebSocket | null {
  if (typeof WebSocket === "undefined") return null;

  const wsURL = config.baseURL.replace(/^http/, "ws");
  const ws = new WebSocket(`${wsURL}/v1/ws?token=${encodeURIComponent(token)}`);

  ws.onopen = () => {
    config.onConnectionChange?.(true);
  };

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data as string) as WSMessage;
      config.onMessage?.(msg);
    } catch {
      // Ignore malformed messages.
    }
  };

  ws.onclose = () => {
    config.onConnectionChange?.(false);
  };

  ws.onerror = () => {
    config.onConnectionChange?.(false);
  };

  return ws;
}
