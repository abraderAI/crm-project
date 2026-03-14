import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { WebhookSubscription, WebhookDelivery } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock entity-api webhook mutations.
const mockCreateWebhook = vi.fn();
const mockDeleteWebhook = vi.fn();
const mockToggleWebhook = vi.fn();
const mockReplayWebhookDelivery = vi.fn();
vi.mock("@/lib/entity-api", () => ({
  createWebhook: (...args: unknown[]) => mockCreateWebhook(...args),
  deleteWebhook: (...args: unknown[]) => mockDeleteWebhook(...args),
  toggleWebhook: (...args: unknown[]) => mockToggleWebhook(...args),
  replayWebhookDelivery: (...args: unknown[]) => mockReplayWebhookDelivery(...args),
}));

import { WebhookView } from "./webhook-view";

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

const delivery1: WebhookDelivery = {
  id: "d1",
  subscription_id: "ws1",
  event_type: "message.created",
  payload: "{}",
  status_code: 200,
  attempts: 1,
  created_at: "2026-01-15T10:00:00Z",
  updated_at: "2026-01-15T10:00:00Z",
};

describe("WebhookView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockCreateWebhook.mockResolvedValue({ id: "ws-new" });
    mockDeleteWebhook.mockResolvedValue(undefined);
    mockToggleWebhook.mockResolvedValue({ ...sub1, is_active: false });
    mockReplayWebhookDelivery.mockResolvedValue(undefined);
  });

  it("renders WebhookManager with initial subscriptions", () => {
    render(<WebhookView initialSubscriptions={[sub1]} initialDeliveries={[]} />);
    expect(screen.getByTestId("webhook-manager")).toBeInTheDocument();
    expect(screen.getByTestId("webhook-item-ws1")).toBeInTheDocument();
  });

  it("renders WebhookDeliveryLog with initial deliveries", () => {
    render(<WebhookView initialSubscriptions={[]} initialDeliveries={[delivery1]} />);
    expect(screen.getByTestId("delivery-log")).toBeInTheDocument();
    expect(screen.getByTestId("delivery-item-d1")).toBeInTheDocument();
  });

  it("calls createWebhook via entity-api on create", async () => {
    const user = userEvent.setup();
    render(<WebhookView initialSubscriptions={[]} initialDeliveries={[]} />);

    await user.click(screen.getByTestId("webhook-create-toggle"));
    await user.type(screen.getByTestId("webhook-url-input"), "https://new.com/hook");
    await user.type(screen.getByTestId("webhook-filter-input"), "thread.created");
    await user.click(screen.getByTestId("webhook-save-btn"));

    expect(mockCreateWebhook).toHaveBeenCalledWith(
      "test-token",
      "https://new.com/hook",
      "thread.created",
    );
  });

  it("calls deleteWebhook via entity-api on delete", async () => {
    const user = userEvent.setup();
    render(<WebhookView initialSubscriptions={[sub1]} initialDeliveries={[]} />);

    await user.click(screen.getByTestId("webhook-delete-ws1"));

    expect(mockDeleteWebhook).toHaveBeenCalledWith("test-token", "ws1");
  });

  it("calls toggleWebhook via entity-api on toggle", async () => {
    const user = userEvent.setup();
    render(<WebhookView initialSubscriptions={[sub1]} initialDeliveries={[]} />);

    await user.click(screen.getByTestId("webhook-toggle-ws1"));

    expect(mockToggleWebhook).toHaveBeenCalledWith("test-token", "ws1");
  });

  it("calls replayWebhookDelivery via entity-api on replay", async () => {
    const user = userEvent.setup();
    render(<WebhookView initialSubscriptions={[]} initialDeliveries={[delivery1]} />);

    await user.click(screen.getByTestId("delivery-replay-d1"));

    expect(mockReplayWebhookDelivery).toHaveBeenCalledWith("test-token", "d1");
  });
});
