import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ChatChannelPanel } from "./chat-channel-panel";

// Ensure navigator.clipboard exists for jsdom
if (!navigator.clipboard) {
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    writable: true,
    configurable: true,
  });
}

describe("ChatChannelPanel", () => {
  it("renders the panel container", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    expect(screen.getByTestId("chat-channel-panel")).toBeInTheDocument();
  });

  it("renders widget appearance customization fields", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    expect(screen.getByTestId("widget-theme-input")).toBeInTheDocument();
    expect(screen.getByTestId("widget-greeting-input")).toBeInTheDocument();
    expect(screen.getByTestId("widget-logo-input")).toBeInTheDocument();
  });

  it("renders the chat widget preview", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    expect(screen.getByTestId("chat-widget-preview")).toBeInTheDocument();
  });

  it("renders the embed code snippet", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    expect(screen.getByTestId("embed-code-snippet")).toBeInTheDocument();
  });

  it("updates preview theme color when theme input changes", async () => {
    const user = userEvent.setup();
    render(<ChatChannelPanel embedKey="org_test" />);
    const textInput = screen.getByTestId("widget-theme-text");
    await user.clear(textInput);
    await user.type(textInput, "#ef4444");
    const button = screen.getByTestId("chat-widget-button");
    expect(button).toHaveStyle({ backgroundColor: "#ef4444" });
  });

  it("updates preview greeting when greeting input changes", async () => {
    const user = userEvent.setup();
    render(<ChatChannelPanel embedKey="org_test" />);
    const input = screen.getByTestId("widget-greeting-input");
    await user.clear(input);
    await user.type(input, "Welcome!");
    // Open the panel to see the greeting
    await user.click(screen.getByTestId("chat-widget-button"));
    expect(screen.getByTestId("chat-widget-greeting")).toHaveTextContent("Welcome!");
  });

  it("passes the correct embed key to embed code snippet", () => {
    render(<ChatChannelPanel embedKey="org_mykey" />);
    const code = screen.getByTestId("embed-code-text");
    expect(code.textContent).toContain('data-org-key="org_mykey"');
  });

  it("renders Widget Appearance heading", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    expect(screen.getByText("Widget Appearance")).toBeInTheDocument();
  });

  it("has default theme color of #3b82f6", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    const button = screen.getByTestId("chat-widget-button");
    expect(button).toHaveStyle({ backgroundColor: "#3b82f6" });
  });

  it("has default greeting message", () => {
    render(<ChatChannelPanel embedKey="org_test" />);
    const input = screen.getByTestId("widget-greeting-input");
    expect(input).toHaveValue("Hello! How can we help you today?");
  });
});
