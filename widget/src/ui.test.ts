import { describe, it, expect, vi } from "vitest";
import {
  generateStyles,
  CHAT_ICON_SVG,
  createWidgetUI,
  type WidgetTheme,
  type WidgetPosition,
  type UIConfig,
} from "./ui";

const defaultTheme: WidgetTheme = {
  primaryColor: "#3B82F6",
  greeting: "Hello!",
};

describe("generateStyles", () => {
  it("includes the primary color", () => {
    const styles = generateStyles(defaultTheme, "bottom-right");
    expect(styles).toContain("#3B82F6");
  });

  it("positions bottom-right by default", () => {
    const styles = generateStyles(defaultTheme, "bottom-right");
    expect(styles).toContain("right: 20px");
  });

  it("positions bottom-left when specified", () => {
    const styles = generateStyles(defaultTheme, "bottom-left");
    expect(styles).toContain("left: 20px");
  });

  it("uses custom primary color", () => {
    const theme: WidgetTheme = { ...defaultTheme, primaryColor: "#FF0000" };
    const styles = generateStyles(theme, "bottom-right");
    expect(styles).toContain("#FF0000");
  });

  it("includes all required CSS classes", () => {
    const styles = generateStyles(defaultTheme, "bottom-right");
    expect(styles).toContain(".crm-widget-btn");
    expect(styles).toContain(".crm-widget-panel");
    expect(styles).toContain(".crm-panel-header");
    expect(styles).toContain(".crm-panel-messages");
    expect(styles).toContain(".crm-panel-input");
    expect(styles).toContain(".crm-msg-bubble");
    expect(styles).toContain(".crm-privacy-notice");
    expect(styles).toContain(".crm-status");
  });
});

describe("CHAT_ICON_SVG", () => {
  it("contains an svg element", () => {
    expect(CHAT_ICON_SVG).toContain("<svg");
    expect(CHAT_ICON_SVG).toContain("</svg>");
  });

  it("contains a path element", () => {
    expect(CHAT_ICON_SVG).toContain("<path");
  });
});

describe("createWidgetUI", () => {
  function createHost(): HTMLElement {
    const host = document.createElement("div");
    document.body.appendChild(host);
    return host;
  }

  function createConfig(overrides?: Partial<UIConfig>): UIConfig {
    return {
      theme: defaultTheme,
      position: "bottom-right" as WidgetPosition,
      onSendMessage: vi.fn(),
      ...overrides,
    };
  }

  it("creates a shadow root on the host", () => {
    const host = createHost();
    createWidgetUI(host, createConfig());
    expect(host.shadowRoot).not.toBeNull();
  });

  it("inserts style element", () => {
    const host = createHost();
    createWidgetUI(host, createConfig());
    const style = host.shadowRoot?.querySelector("style");
    expect(style).not.toBeNull();
    expect(style?.textContent).toContain("#3B82F6");
  });

  it("inserts chat button", () => {
    const host = createHost();
    createWidgetUI(host, createConfig());
    const btn = host.shadowRoot?.querySelector(".crm-widget-btn");
    expect(btn).not.toBeNull();
    expect(btn?.getAttribute("aria-label")).toBe("Open chat");
  });

  it("inserts chat panel (initially hidden)", () => {
    const host = createHost();
    createWidgetUI(host, createConfig());
    const panel = host.shadowRoot?.querySelector(".crm-widget-panel");
    expect(panel).not.toBeNull();
    expect(panel?.classList.contains("open")).toBe(false);
  });

  it("toggles panel open/close on button click", () => {
    const host = createHost();
    const onToggle = vi.fn();
    createWidgetUI(host, createConfig({ onToggle }));

    const btn = host.shadowRoot?.querySelector(".crm-widget-btn") as HTMLButtonElement;
    const panel = host.shadowRoot?.querySelector(".crm-widget-panel");

    // Open.
    btn.click();
    expect(panel?.classList.contains("open")).toBe(true);
    expect(onToggle).toHaveBeenCalledWith(true);

    // Close.
    btn.click();
    expect(panel?.classList.contains("open")).toBe(false);
    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("closes panel via close button", () => {
    const host = createHost();
    const onToggle = vi.fn();
    createWidgetUI(host, createConfig({ onToggle }));

    const btn = host.shadowRoot?.querySelector(".crm-widget-btn") as HTMLButtonElement;
    btn.click(); // Open.

    const closeBtn = host.shadowRoot?.querySelector(".crm-panel-close") as HTMLButtonElement;
    closeBtn.click();
    const panel = host.shadowRoot?.querySelector(".crm-widget-panel");
    expect(panel?.classList.contains("open")).toBe(false);
  });

  it("sends message on button click", () => {
    const onSendMessage = vi.fn();
    const host = createHost();
    createWidgetUI(host, createConfig({ onSendMessage }));

    const input = host.shadowRoot?.querySelector("input") as HTMLInputElement;
    const sendBtn = host.shadowRoot?.querySelector(
      ".crm-panel-input button",
    ) as HTMLButtonElement;

    input.value = "Hello!";
    sendBtn.click();
    expect(onSendMessage).toHaveBeenCalledWith("Hello!");
    expect(input.value).toBe(""); // Input cleared.
  });

  it("does not send empty messages", () => {
    const onSendMessage = vi.fn();
    const host = createHost();
    createWidgetUI(host, createConfig({ onSendMessage }));

    const sendBtn = host.shadowRoot?.querySelector(
      ".crm-panel-input button",
    ) as HTMLButtonElement;
    sendBtn.click();
    expect(onSendMessage).not.toHaveBeenCalled();
  });

  it("sends message on Enter key", () => {
    const onSendMessage = vi.fn();
    const host = createHost();
    createWidgetUI(host, createConfig({ onSendMessage }));

    const input = host.shadowRoot?.querySelector("input") as HTMLInputElement;
    input.value = "Test";
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
    expect(onSendMessage).toHaveBeenCalledWith("Test");
  });

  it("does not send on Shift+Enter", () => {
    const onSendMessage = vi.fn();
    const host = createHost();
    createWidgetUI(host, createConfig({ onSendMessage }));

    const input = host.shadowRoot?.querySelector("input") as HTMLInputElement;
    input.value = "Test";
    input.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Enter", shiftKey: true, bubbles: true }),
    );
    expect(onSendMessage).not.toHaveBeenCalled();
  });

  it("addMessage adds messages to container", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());

    controller.addMessage({
      body: "Hello",
      author: "visitor",
      timestamp: new Date("2024-01-01T12:00:00Z"),
    });

    const msgs = host.shadowRoot?.querySelectorAll(".crm-msg");
    expect(msgs?.length).toBe(1);
    expect(msgs?.[0]?.classList.contains("crm-msg-visitor")).toBe(true);
    expect(msgs?.[0]?.querySelector(".crm-msg-bubble")?.textContent).toBe("Hello");
  });

  it("addMessage adds AI messages", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());

    controller.addMessage({
      body: "How can I help?",
      author: "ai",
      timestamp: new Date(),
    });

    const msgs = host.shadowRoot?.querySelectorAll(".crm-msg-ai");
    expect(msgs?.length).toBe(1);
  });

  it("addMessage adds agent messages", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());

    controller.addMessage({
      body: "Agent here",
      author: "agent",
      timestamp: new Date(),
    });

    const msgs = host.shadowRoot?.querySelectorAll(".crm-msg-agent");
    expect(msgs?.length).toBe(1);
  });

  it("setStatus shows and hides status bar", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());

    controller.setStatus("Connecting...");
    const status = host.shadowRoot?.querySelector(".crm-status") as HTMLElement;
    expect(status.style.display).toBe("block");
    expect(status.textContent).toBe("Connecting...");

    controller.setStatus(null);
    expect(status.style.display).toBe("none");
  });

  it("setSendEnabled toggles button disabled state", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());

    const sendBtn = host.shadowRoot?.querySelector(
      ".crm-panel-input button",
    ) as HTMLButtonElement;
    controller.setSendEnabled(false);
    expect(sendBtn.disabled).toBe(true);

    controller.setSendEnabled(true);
    expect(sendBtn.disabled).toBe(false);
  });

  it("open/close/isOpen work correctly", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());

    expect(controller.isOpen()).toBe(false);
    controller.open();
    expect(controller.isOpen()).toBe(true);
    controller.close();
    expect(controller.isOpen()).toBe(false);
  });

  it("getPanel returns the panel element", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());
    const panel = controller.getPanel();
    expect(panel.className).toBe("crm-widget-panel");
  });

  it("getMessagesContainer returns the messages container", () => {
    const host = createHost();
    const controller = createWidgetUI(host, createConfig());
    const container = controller.getMessagesContainer();
    expect(container.className).toBe("crm-panel-messages");
  });

  it("includes privacy notice", () => {
    const host = createHost();
    createWidgetUI(host, createConfig());
    const privacy = host.shadowRoot?.querySelector(".crm-privacy-notice");
    expect(privacy).not.toBeNull();
    expect(privacy?.textContent).toContain("fingerprinting");
  });
});
