/**
 * CRM Chat Widget — main entry point.
 * Self-initialising IIFE that reads config from the script tag's data attributes:
 *   data-embed-key  — required, the org embed key
 *   data-theme      — optional, JSON theme override
 *   data-position   — optional, "bottom-right" (default) or "bottom-left"
 */

import { createSession, sendMessage, connectWebSocket, type WSMessage } from "./chat";
import { generateFingerprint } from "./fingerprint";
import { createWidgetUI, type WidgetPosition, type WidgetTheme } from "./ui";

/** Default widget theme. */
const DEFAULT_THEME: WidgetTheme = {
  primaryColor: "#3B82F6",
  greeting: "Hello! How can we help you today?",
};

/** Parse the script tag's data attributes. */
export function parseScriptConfig(scriptEl: HTMLScriptElement | null): {
  embedKey: string;
  theme: WidgetTheme;
  position: WidgetPosition;
  baseURL: string;
} {
  const embedKey = scriptEl?.dataset["embedKey"] ?? "";
  const position = (scriptEl?.dataset["position"] ?? "bottom-right") as WidgetPosition;

  let theme = { ...DEFAULT_THEME };
  const themeRaw = scriptEl?.dataset["theme"];
  if (themeRaw) {
    try {
      const parsed = JSON.parse(themeRaw) as Partial<WidgetTheme>;
      theme = { ...theme, ...parsed };
    } catch {
      // Ignore invalid JSON, use defaults.
    }
  }

  // Derive API base URL from script src or use current origin.
  let baseURL = "";
  if (scriptEl?.src) {
    try {
      const url = new URL(scriptEl.src);
      baseURL = url.origin;
    } catch /* v8 ignore next */ {
      baseURL = window.location.origin;
    }
  } else {
    baseURL = window.location.origin;
  }

  return { embedKey, theme, position, baseURL };
}

/** Initialise the chat widget. */
export async function initWidget(config: {
  embedKey: string;
  theme: WidgetTheme;
  position: WidgetPosition;
  baseURL: string;
}): Promise<void> {
  if (!config.embedKey) {
    console.error("[CRM Widget] Missing data-embed-key attribute.");
    return;
  }

  // Create host element.
  const host = document.createElement("div");
  host.id = "crm-chat-widget";
  document.body.appendChild(host);

  // Build UI.
  const ui = createWidgetUI(host, {
    theme: config.theme,
    position: config.position,
    onSendMessage: (message: string) => void handleSend(message),
  });

  // Show greeting.
  ui.addMessage({
    body: config.theme.greeting,
    author: "ai",
    timestamp: new Date(),
  });

  // Generate fingerprint and create session.
  const fingerprint = generateFingerprint();
  let session = await createSession(config.baseURL, config.embedKey, fingerprint);

  // Show returning visitor greeting if applicable.
  if (session.returning && session.greeting) {
    ui.addMessage({
      body: `Welcome back! ${session.greeting}`,
      author: "ai",
      timestamp: new Date(),
    });
  }

  // Connect WebSocket for real-time updates.
  connectWebSocket(
    {
      baseURL: config.baseURL,
      embedKey: config.embedKey,
      onMessage: (msg: WSMessage) => {
        if (msg.type === "chat.message" && msg.payload["author"] !== "visitor") {
          ui.addMessage({
            body: msg.payload["body"] as string,
            author: (msg.payload["type"] as string) === "escalation_timeout" ? "ai" : "ai",
            timestamp: new Date(),
          });
        }
        if (msg.type === "chat.escalated") {
          ui.setStatus("Connecting you to an agent...");
        }
      },
      onConnectionChange: (connected: boolean) => {
        if (!connected) {
          // Silently handle disconnect — messages still work via HTTP.
        }
      },
    },
    session.token,
  );

  /** Handle sending a message. */
  async function handleSend(message: string): Promise<void> {
    // Display visitor message immediately.
    ui.addMessage({
      body: message,
      author: "visitor",
      timestamp: new Date(),
    });
    ui.setSendEnabled(false);

    try {
      const resp = await sendMessage(config.baseURL, session.token, message);

      if (resp.type === "escalation") {
        ui.setStatus("Connecting you to a human agent. Please wait...");
        ui.addMessage({
          body: resp.message,
          author: "ai",
          timestamp: new Date(),
        });
      } else {
        ui.setStatus(null);
        ui.addMessage({
          body: resp.message,
          author: "ai",
          timestamp: new Date(),
        });
      }
    } catch (err) {
      ui.addMessage({
        body: "Sorry, something went wrong. Please try again.",
        author: "ai",
        timestamp: new Date(),
      });
    } finally {
      ui.setSendEnabled(true);
    }
  }
}

// Auto-initialise when loaded as a script tag.
/* v8 ignore start -- auto-init runs on page load, not in test environment */
if (typeof document !== "undefined") {
  const currentScript = document.currentScript as HTMLScriptElement | null;
  if (currentScript) {
    const config = parseScriptConfig(currentScript);
    void initWidget(config);
  }
}
/* v8 ignore stop */
