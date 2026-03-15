/**
 * Shadow DOM UI components for the chat widget.
 * All styles are encapsulated in the shadow root to avoid CSS conflicts.
 */

/** Widget theme configuration. */
export interface WidgetTheme {
  primaryColor: string;
  logoURL?: string;
  greeting: string;
}

/** Widget position on the page. */
export type WidgetPosition = "bottom-right" | "bottom-left";

/** Configuration for the UI. */
export interface UIConfig {
  theme: WidgetTheme;
  position: WidgetPosition;
  onSendMessage: (message: string) => void;
  onToggle?: (open: boolean) => void;
}

/** Message displayed in the chat panel. */
export interface ChatMessage {
  body: string;
  author: "visitor" | "ai" | "agent";
  timestamp: Date;
}

/** Generate the widget CSS styles. */
export function generateStyles(theme: WidgetTheme, position: WidgetPosition): string {
  const positionCSS =
    position === "bottom-left" ? "left: 20px; right: auto;" : "right: 20px; left: auto;";

  const panelPositionCSS =
    position === "bottom-left" ? "left: 20px; right: auto;" : "right: 20px; left: auto;";

  return `
    :host { all: initial; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
    .crm-widget-btn {
      position: fixed; bottom: 20px; ${positionCSS}
      width: 56px; height: 56px; border-radius: 50%; border: none;
      background: ${theme.primaryColor}; color: #fff; cursor: pointer;
      box-shadow: 0 4px 12px rgba(0,0,0,0.15); z-index: 999999;
      display: flex; align-items: center; justify-content: center;
      transition: transform 0.2s ease;
    }
    .crm-widget-btn:hover { transform: scale(1.05); }
    .crm-widget-btn svg { width: 24px; height: 24px; fill: #fff; }
    .crm-widget-panel {
      position: fixed; bottom: 90px; ${panelPositionCSS}
      width: 380px; max-height: 520px; border-radius: 12px;
      background: #fff; box-shadow: 0 8px 32px rgba(0,0,0,0.15);
      z-index: 999999; display: none; flex-direction: column;
      overflow: hidden;
    }
    .crm-widget-panel.open { display: flex; }
    .crm-panel-header {
      background: ${theme.primaryColor}; color: #fff; padding: 16px;
      font-size: 16px; font-weight: 600; display: flex;
      align-items: center; justify-content: space-between;
    }
    .crm-panel-close {
      background: none; border: none; color: #fff; cursor: pointer;
      font-size: 20px; line-height: 1; padding: 0;
    }
    .crm-panel-messages {
      flex: 1; overflow-y: auto; padding: 16px;
      max-height: 340px; min-height: 200px;
    }
    .crm-msg { margin-bottom: 12px; max-width: 85%; }
    .crm-msg-visitor { margin-left: auto; text-align: right; }
    .crm-msg-ai, .crm-msg-agent { margin-right: auto; }
    .crm-msg-bubble {
      display: inline-block; padding: 10px 14px; border-radius: 16px;
      font-size: 14px; line-height: 1.4; word-wrap: break-word;
    }
    .crm-msg-visitor .crm-msg-bubble {
      background: ${theme.primaryColor}; color: #fff;
      border-bottom-right-radius: 4px;
    }
    .crm-msg-ai .crm-msg-bubble, .crm-msg-agent .crm-msg-bubble {
      background: #f0f0f0; color: #333;
      border-bottom-left-radius: 4px;
    }
    .crm-msg-time {
      font-size: 11px; color: #999; margin-top: 4px;
    }
    .crm-panel-input {
      display: flex; padding: 12px; border-top: 1px solid #eee;
    }
    .crm-panel-input input {
      flex: 1; border: 1px solid #ddd; border-radius: 8px;
      padding: 10px 14px; font-size: 14px; outline: none;
    }
    .crm-panel-input input:focus { border-color: ${theme.primaryColor}; }
    .crm-panel-input button {
      margin-left: 8px; background: ${theme.primaryColor}; color: #fff;
      border: none; border-radius: 8px; padding: 10px 16px;
      cursor: pointer; font-size: 14px;
    }
    .crm-panel-input button:disabled { opacity: 0.5; cursor: not-allowed; }
    .crm-privacy-notice {
      padding: 8px 16px; font-size: 11px; color: #999;
      text-align: center; border-top: 1px solid #eee;
    }
    .crm-status {
      padding: 8px 16px; font-size: 12px; color: #666;
      text-align: center; background: #fefce8;
    }
  `;
}

/** Chat icon SVG markup. */
export const CHAT_ICON_SVG = `<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
  <path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/>
</svg>`;

/** Create the chat widget UI within a shadow root. */
export function createWidgetUI(host: HTMLElement, config: UIConfig): WidgetController {
  const shadow = host.attachShadow({ mode: "open" });

  const style = document.createElement("style");
  style.textContent = generateStyles(config.theme, config.position);
  shadow.appendChild(style);

  // Floating button.
  const btn = document.createElement("button");
  btn.className = "crm-widget-btn";
  btn.innerHTML = CHAT_ICON_SVG;
  btn.setAttribute("aria-label", "Open chat");
  shadow.appendChild(btn);

  // Chat panel.
  const panel = document.createElement("div");
  panel.className = "crm-widget-panel";

  // Header.
  const header = document.createElement("div");
  header.className = "crm-panel-header";
  header.innerHTML = `<span>Chat with us</span><button class="crm-panel-close" aria-label="Close chat">&times;</button>`;
  panel.appendChild(header);

  // Status bar (hidden by default).
  const statusBar = document.createElement("div");
  statusBar.className = "crm-status";
  statusBar.style.display = "none";
  panel.appendChild(statusBar);

  // Messages container.
  const messagesContainer = document.createElement("div");
  messagesContainer.className = "crm-panel-messages";
  panel.appendChild(messagesContainer);

  // Input area.
  const inputArea = document.createElement("div");
  inputArea.className = "crm-panel-input";
  const input = document.createElement("input");
  input.type = "text";
  input.placeholder = "Type a message...";
  input.setAttribute("aria-label", "Chat message input");
  const sendBtn = document.createElement("button");
  sendBtn.textContent = "Send";
  sendBtn.setAttribute("aria-label", "Send message");
  inputArea.appendChild(input);
  inputArea.appendChild(sendBtn);
  panel.appendChild(inputArea);

  // Privacy notice.
  const privacy = document.createElement("div");
  privacy.className = "crm-privacy-notice";
  privacy.textContent =
    "This chat uses browser fingerprinting for session continuity. By chatting, you consent to this.";
  panel.appendChild(privacy);

  shadow.appendChild(panel);

  // State.
  let isOpen = false;

  // Event handlers.
  btn.addEventListener("click", () => {
    isOpen = !isOpen;
    panel.classList.toggle("open", isOpen);
    config.onToggle?.(isOpen);
    if (isOpen) input.focus();
  });

  header.querySelector(".crm-panel-close")?.addEventListener("click", () => {
    isOpen = false;
    panel.classList.remove("open");
    config.onToggle?.(false);
  });

  const handleSend = (): void => {
    const text = input.value.trim();
    if (!text) return;
    config.onSendMessage(text);
    input.value = "";
  };

  sendBtn.addEventListener("click", handleSend);
  input.addEventListener("keydown", (e: KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  });

  return {
    addMessage(msg: ChatMessage): void {
      const div = document.createElement("div");
      div.className = `crm-msg crm-msg-${msg.author}`;
      const bubble = document.createElement("div");
      bubble.className = "crm-msg-bubble";
      bubble.textContent = msg.body;
      div.appendChild(bubble);
      const time = document.createElement("div");
      time.className = "crm-msg-time";
      time.textContent = msg.timestamp.toLocaleTimeString();
      div.appendChild(time);
      messagesContainer.appendChild(div);
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    },
    setStatus(text: string | null): void {
      if (text) {
        statusBar.textContent = text;
        statusBar.style.display = "block";
      } else {
        statusBar.style.display = "none";
      }
    },
    setSendEnabled(enabled: boolean): void {
      sendBtn.disabled = !enabled;
    },
    open(): void {
      isOpen = true;
      panel.classList.add("open");
      config.onToggle?.(true);
    },
    close(): void {
      isOpen = false;
      panel.classList.remove("open");
      config.onToggle?.(false);
    },
    isOpen(): boolean {
      return isOpen;
    },
    getPanel(): HTMLElement {
      return panel;
    },
    getMessagesContainer(): HTMLElement {
      return messagesContainer;
    },
  };
}

/** Controller interface for the widget UI. */
export interface WidgetController {
  addMessage(msg: ChatMessage): void;
  setStatus(text: string | null): void;
  setSendEnabled(enabled: boolean): void;
  open(): void;
  close(): void;
  isOpen(): boolean;
  getPanel(): HTMLElement;
  getMessagesContainer(): HTMLElement;
}
