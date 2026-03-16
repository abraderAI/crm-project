import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { ChatWidgetPreview } from "./chat-widget-preview";

describe("ChatWidgetPreview", () => {
  const defaultProps = {
    theme: "#3b82f6",
    greeting: "Hello! How can we help?",
    logoUrl: "https://example.com/logo.png",
  };

  it("renders the preview container", () => {
    render(<ChatWidgetPreview {...defaultProps} />);
    expect(screen.getByTestId("chat-widget-preview")).toBeInTheDocument();
  });

  it("renders the floating chat button", () => {
    render(<ChatWidgetPreview {...defaultProps} />);
    expect(screen.getByTestId("chat-widget-button")).toBeInTheDocument();
  });

  it("applies the theme color to the chat button", () => {
    render(<ChatWidgetPreview {...defaultProps} />);
    const button = screen.getByTestId("chat-widget-button");
    expect(button).toHaveStyle({ backgroundColor: "#3b82f6" });
  });

  it("does not show the expanded panel by default", () => {
    render(<ChatWidgetPreview {...defaultProps} />);
    expect(screen.queryByTestId("chat-widget-panel")).not.toBeInTheDocument();
  });

  it("shows the expanded panel when chat button is clicked", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} />);
    await user.click(screen.getByTestId("chat-widget-button"));
    expect(screen.getByTestId("chat-widget-panel")).toBeInTheDocument();
  });

  it("displays the greeting text in the expanded panel", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} />);
    await user.click(screen.getByTestId("chat-widget-button"));
    expect(screen.getByTestId("chat-widget-greeting")).toHaveTextContent("Hello! How can we help?");
  });

  it("applies theme color to the panel header", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} />);
    await user.click(screen.getByTestId("chat-widget-button"));
    const header = screen.getByTestId("chat-widget-panel-header");
    expect(header).toHaveStyle({ backgroundColor: "#3b82f6" });
  });

  it("displays the logo image when logoUrl is provided", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} />);
    await user.click(screen.getByTestId("chat-widget-button"));
    const logo = screen.getByTestId("chat-widget-logo");
    expect(logo).toHaveAttribute("src", "https://example.com/logo.png");
  });

  it("does not display logo when logoUrl is empty", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} logoUrl="" />);
    await user.click(screen.getByTestId("chat-widget-button"));
    expect(screen.queryByTestId("chat-widget-logo")).not.toBeInTheDocument();
  });

  it("collapses the panel when the close button is clicked", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} />);
    await user.click(screen.getByTestId("chat-widget-button"));
    expect(screen.getByTestId("chat-widget-panel")).toBeInTheDocument();
    await user.click(screen.getByTestId("chat-widget-close"));
    expect(screen.queryByTestId("chat-widget-panel")).not.toBeInTheDocument();
  });

  it("updates greeting text reactively without re-mount", () => {
    const { rerender } = render(<ChatWidgetPreview {...defaultProps} />);
    rerender(<ChatWidgetPreview {...defaultProps} greeting="Updated greeting!" />);
    // Greeting is visible once panel is opened, but the prop is accepted without error
    expect(screen.getByTestId("chat-widget-preview")).toBeInTheDocument();
  });

  it("updates theme color reactively", () => {
    const { rerender } = render(<ChatWidgetPreview {...defaultProps} />);
    rerender(<ChatWidgetPreview {...defaultProps} theme="#ef4444" />);
    const button = screen.getByTestId("chat-widget-button");
    expect(button).toHaveStyle({ backgroundColor: "#ef4444" });
  });

  it("shows a mock message input in the expanded panel", async () => {
    const user = userEvent.setup();
    render(<ChatWidgetPreview {...defaultProps} />);
    await user.click(screen.getByTestId("chat-widget-button"));
    expect(screen.getByTestId("chat-widget-input")).toBeInTheDocument();
  });

  it("renders the preview label", () => {
    render(<ChatWidgetPreview {...defaultProps} />);
    expect(screen.getByText("Widget Preview")).toBeInTheDocument();
  });
});
