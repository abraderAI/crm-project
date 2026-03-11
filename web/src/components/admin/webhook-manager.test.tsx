import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { WebhookManager } from "./webhook-manager";
import type { WebhookSubscription } from "@/lib/api-types";

const sub1: WebhookSubscription = {
  id: "ws1",
  org_id: "org1",
  scope_type: "org",
  scope_id: "org1",
  url: "https://example.com/hook",
  event_filter: "message.created",
  is_active: true,
  created_at: "2026-01-10T00:00:00Z",
  updated_at: "2026-01-10T00:00:00Z",
};

const sub2: WebhookSubscription = {
  id: "ws2",
  org_id: "org1",
  scope_type: "space",
  scope_id: "sp1",
  url: "https://other.com/callback",
  event_filter: "",
  is_active: false,
  created_at: "2026-01-11T00:00:00Z",
  updated_at: "2026-01-11T00:00:00Z",
};

const defaultProps = {
  subscriptions: [sub1, sub2],
  onCreate: vi.fn(),
  onDelete: vi.fn(),
  onToggle: vi.fn(),
};

describe("WebhookManager", () => {
  it("renders the heading and count", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByText("Webhooks")).toBeInTheDocument();
    expect(screen.getByTestId("webhook-count")).toHaveTextContent("2");
  });

  it("renders the webhook icon", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByTestId("webhook-icon")).toBeInTheDocument();
  });

  it("shows empty state when no subscriptions", () => {
    render(<WebhookManager {...defaultProps} subscriptions={[]} />);
    expect(screen.getByTestId("webhook-empty")).toHaveTextContent("No webhook subscriptions.");
  });

  it("shows loading state", () => {
    render(<WebhookManager {...defaultProps} subscriptions={[]} loading={true} />);
    expect(screen.getByTestId("webhook-loading")).toBeInTheDocument();
  });

  it("renders subscription items", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByTestId("webhook-item-ws1")).toBeInTheDocument();
    expect(screen.getByTestId("webhook-item-ws2")).toBeInTheDocument();
  });

  it("displays subscription URL", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByTestId("webhook-url-ws1")).toHaveTextContent("https://example.com/hook");
  });

  it("displays scope info", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByTestId("webhook-scope-ws1")).toHaveTextContent("org:org1");
  });

  it("displays event filter when present", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByTestId("webhook-filter-ws1")).toHaveTextContent("message.created");
    expect(screen.queryByTestId("webhook-filter-ws2")).not.toBeInTheDocument();
  });

  it("shows Active/Inactive status", () => {
    render(<WebhookManager {...defaultProps} />);
    expect(screen.getByTestId("webhook-toggle-ws1")).toHaveTextContent("Active");
    expect(screen.getByTestId("webhook-toggle-ws2")).toHaveTextContent("Inactive");
  });

  it("calls onToggle when status button clicked", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<WebhookManager {...defaultProps} onToggle={onToggle} />);

    await user.click(screen.getByTestId("webhook-toggle-ws1"));
    expect(onToggle).toHaveBeenCalledWith("ws1");
  });

  it("calls onDelete when delete clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(<WebhookManager {...defaultProps} onDelete={onDelete} />);

    await user.click(screen.getByTestId("webhook-delete-ws1"));
    expect(onDelete).toHaveBeenCalledWith("ws1");
  });

  it("shows create form when Add Webhook clicked", async () => {
    const user = userEvent.setup();
    render(<WebhookManager {...defaultProps} />);

    expect(screen.queryByTestId("webhook-create-form")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("webhook-create-toggle"));
    expect(screen.getByTestId("webhook-create-form")).toBeInTheDocument();
  });

  it("creates webhook with URL and filter", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    render(<WebhookManager {...defaultProps} onCreate={onCreate} />);

    await user.click(screen.getByTestId("webhook-create-toggle"));
    await user.type(screen.getByTestId("webhook-url-input"), "https://new.com/hook");
    await user.type(screen.getByTestId("webhook-filter-input"), "thread.created");
    await user.click(screen.getByTestId("webhook-save-btn"));
    expect(onCreate).toHaveBeenCalledWith("https://new.com/hook", "thread.created");
  });

  it("shows error for empty URL", async () => {
    const user = userEvent.setup();
    render(<WebhookManager {...defaultProps} />);

    await user.click(screen.getByTestId("webhook-create-toggle"));
    await user.click(screen.getByTestId("webhook-save-btn"));
    expect(screen.getByTestId("webhook-error")).toHaveTextContent("URL is required.");
  });

  it("shows error for invalid URL", async () => {
    const user = userEvent.setup();
    render(<WebhookManager {...defaultProps} />);

    await user.click(screen.getByTestId("webhook-create-toggle"));
    await user.type(screen.getByTestId("webhook-url-input"), "not-a-url");
    await user.click(screen.getByTestId("webhook-save-btn"));
    expect(screen.getByTestId("webhook-error")).toHaveTextContent("Please enter a valid URL.");
  });

  it("hides create form after successful creation", async () => {
    const user = userEvent.setup();
    render(<WebhookManager {...defaultProps} />);

    await user.click(screen.getByTestId("webhook-create-toggle"));
    await user.type(screen.getByTestId("webhook-url-input"), "https://new.com/hook");
    await user.click(screen.getByTestId("webhook-save-btn"));
    expect(screen.queryByTestId("webhook-create-form")).not.toBeInTheDocument();
  });

  it("hides create form on cancel", async () => {
    const user = userEvent.setup();
    render(<WebhookManager {...defaultProps} />);

    await user.click(screen.getByTestId("webhook-create-toggle"));
    await user.click(screen.getByTestId("webhook-cancel-btn"));
    expect(screen.queryByTestId("webhook-create-form")).not.toBeInTheDocument();
  });
});
