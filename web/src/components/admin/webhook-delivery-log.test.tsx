import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { WebhookDeliveryLog } from "./webhook-delivery-log";
import type { WebhookDelivery } from "@/lib/api-types";

const delivery200: WebhookDelivery = {
  id: "d1",
  subscription_id: "ws1",
  event_type: "message.created",
  payload: "{}",
  status_code: 200,
  attempts: 1,
  created_at: "2026-01-15T10:00:00Z",
  updated_at: "2026-01-15T10:00:00Z",
};

const delivery500: WebhookDelivery = {
  id: "d2",
  subscription_id: "ws1",
  event_type: "thread.updated",
  payload: "{}",
  status_code: 500,
  attempts: 3,
  created_at: "2026-01-14T10:00:00Z",
  updated_at: "2026-01-14T10:00:00Z",
};

const delivery404: WebhookDelivery = {
  id: "d3",
  subscription_id: "ws1",
  event_type: "thread.deleted",
  payload: "{}",
  status_code: 404,
  attempts: 2,
  created_at: "2026-01-13T10:00:00Z",
  updated_at: "2026-01-13T10:00:00Z",
};

describe("WebhookDeliveryLog", () => {
  it("renders the heading", () => {
    render(<WebhookDeliveryLog deliveries={[]} onReplay={vi.fn()} />);
    expect(screen.getByText("Delivery Log")).toBeInTheDocument();
  });

  it("shows empty state", () => {
    render(<WebhookDeliveryLog deliveries={[]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-empty")).toHaveTextContent("No deliveries recorded.");
  });

  it("shows loading state", () => {
    render(<WebhookDeliveryLog deliveries={[]} loading={true} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-loading")).toBeInTheDocument();
  });

  it("renders delivery items", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200, delivery500]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-item-d1")).toBeInTheDocument();
    expect(screen.getByTestId("delivery-item-d2")).toBeInTheDocument();
  });

  it("displays status code", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-status-d1")).toHaveTextContent("200");
  });

  it("applies green color for 2xx", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-status-d1")).toHaveClass("bg-green-100");
  });

  it("applies red color for 5xx", () => {
    render(<WebhookDeliveryLog deliveries={[delivery500]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-status-d2")).toHaveClass("bg-red-100");
  });

  it("applies yellow color for 4xx", () => {
    render(<WebhookDeliveryLog deliveries={[delivery404]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-status-d3")).toHaveClass("bg-yellow-100");
  });

  it("displays event type", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-event-d1")).toHaveTextContent("message.created");
  });

  it("displays attempt count", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200, delivery500]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-attempts-d1")).toHaveTextContent("1 attempt");
    expect(screen.getByTestId("delivery-attempts-d2")).toHaveTextContent("3 attempts");
  });

  it("calls onReplay when replay clicked", async () => {
    const user = userEvent.setup();
    const onReplay = vi.fn();
    render(<WebhookDeliveryLog deliveries={[delivery200]} onReplay={onReplay} />);

    await user.click(screen.getByTestId("delivery-replay-d1"));
    expect(onReplay).toHaveBeenCalledWith("d1");
  });

  it("renders load more button when hasMore", () => {
    render(
      <WebhookDeliveryLog
        deliveries={[delivery200]}
        onReplay={vi.fn()}
        hasMore={true}
        onLoadMore={vi.fn()}
      />,
    );
    expect(screen.getByTestId("delivery-load-more")).toBeInTheDocument();
  });

  it("hides load more when not hasMore", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200]} onReplay={vi.fn()} hasMore={false} />);
    expect(screen.queryByTestId("delivery-load-more")).not.toBeInTheDocument();
  });

  it("calls onLoadMore when load more clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(
      <WebhookDeliveryLog
        deliveries={[delivery200]}
        onReplay={vi.fn()}
        hasMore={true}
        onLoadMore={onLoadMore}
      />,
    );

    await user.click(screen.getByTestId("delivery-load-more"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("displays formatted date", () => {
    render(<WebhookDeliveryLog deliveries={[delivery200]} onReplay={vi.fn()} />);
    expect(screen.getByTestId("delivery-date-d1")).toBeInTheDocument();
  });
});
