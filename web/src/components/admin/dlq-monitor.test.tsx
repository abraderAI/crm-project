import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DLQMonitor } from "./dlq-monitor";
import type { DeadLetterEvent } from "@/lib/api-types";

const failedEvent: DeadLetterEvent = {
  id: "evt1",
  org_id: "org1",
  channel_type: "email",
  event_payload: "{}",
  error_message: "SMTP connection timeout after 30s",
  attempts: 3,
  last_attempt_at: "2026-03-14T10:00:00Z",
  status: "failed",
  created_at: "2026-03-14T08:00:00Z",
};

const retryingEvent: DeadLetterEvent = {
  id: "evt2",
  org_id: "org1",
  channel_type: "email",
  event_payload: "{}",
  error_message: "DNS lookup failed",
  attempts: 1,
  last_attempt_at: "2026-03-14T11:00:00Z",
  status: "retrying",
  created_at: "2026-03-14T10:30:00Z",
};

const resolvedEvent: DeadLetterEvent = {
  id: "evt3",
  org_id: "org1",
  channel_type: "email",
  event_payload: "{}",
  error_message: "Temporary error resolved",
  attempts: 2,
  last_attempt_at: "2026-03-14T12:00:00Z",
  status: "resolved",
  created_at: "2026-03-14T09:00:00Z",
};

const dismissedEvent: DeadLetterEvent = {
  id: "evt4",
  org_id: "org1",
  channel_type: "email",
  event_payload: "{}",
  error_message: "Invalid payload",
  attempts: 1,
  last_attempt_at: "2026-03-14T09:30:00Z",
  status: "dismissed",
  created_at: "2026-03-14T09:00:00Z",
};

const allEvents = [failedEvent, retryingEvent, resolvedEvent, dismissedEvent];

const defaultProps = {
  org: "org1",
  channelType: "email" as const,
  events: allEvents,
  onRetry: vi.fn(),
  onDismiss: vi.fn(),
  onRefresh: vi.fn(),
};

describe("DLQMonitor", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the heading", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByText("Dead Letter Queue")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<DLQMonitor {...defaultProps} events={[]} loading={true} />);
    expect(screen.getByTestId("dlq-loading")).toBeInTheDocument();
  });

  it("shows empty state when no events", () => {
    render(<DLQMonitor {...defaultProps} events={[]} />);
    expect(screen.getByTestId("dlq-empty")).toHaveTextContent("No dead-letter events.");
  });

  it("renders event rows", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-row-evt1")).toBeInTheDocument();
    expect(screen.getByTestId("dlq-row-evt2")).toBeInTheDocument();
    expect(screen.getByTestId("dlq-row-evt3")).toBeInTheDocument();
    expect(screen.getByTestId("dlq-row-evt4")).toBeInTheDocument();
  });

  it("displays error message", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-error-evt1")).toHaveTextContent(
      "SMTP connection timeout after 30s",
    );
  });

  it("displays status badge", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-status-evt1")).toHaveTextContent("failed");
    expect(screen.getByTestId("dlq-status-evt1")).toHaveClass("bg-red-100");
  });

  it("displays retrying status with yellow badge", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-status-evt2")).toHaveClass("bg-yellow-100");
  });

  it("displays resolved status with green badge", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-status-evt3")).toHaveClass("bg-green-100");
  });

  it("calls onRetry when retry button clicked", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<DLQMonitor {...defaultProps} onRetry={onRetry} />);
    await user.click(screen.getByTestId("dlq-retry-evt1"));
    expect(onRetry).toHaveBeenCalledWith("evt1");
  });

  it("disables retry for resolved events", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-retry-evt3")).toBeDisabled();
  });

  it("disables retry for dismissed events", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-retry-evt4")).toBeDisabled();
  });

  it("shows confirmation dialog for dismiss", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    render(<DLQMonitor {...defaultProps} />);
    await user.click(screen.getByTestId("dlq-dismiss-evt1"));
    expect(screen.getByTestId("dlq-dismiss-confirm-evt1")).toBeInTheDocument();
  });

  it("calls onDismiss after confirmation", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    const onDismiss = vi.fn();
    render(<DLQMonitor {...defaultProps} onDismiss={onDismiss} />);
    await user.click(screen.getByTestId("dlq-dismiss-evt1"));
    await user.click(screen.getByTestId("dlq-dismiss-confirm-evt1"));
    expect(onDismiss).toHaveBeenCalledWith("evt1");
  });

  it("cancels dismiss confirmation", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    render(<DLQMonitor {...defaultProps} />);
    await user.click(screen.getByTestId("dlq-dismiss-evt1"));
    await user.click(screen.getByTestId("dlq-dismiss-cancel-evt1"));
    expect(screen.queryByTestId("dlq-dismiss-confirm-evt1")).not.toBeInTheDocument();
  });

  it("disables dismiss for resolved events", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-dismiss-evt3")).toBeDisabled();
  });

  it("filters by status", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    render(<DLQMonitor {...defaultProps} />);
    await user.selectOptions(screen.getByTestId("dlq-status-filter"), "failed");
    expect(screen.getByTestId("dlq-row-evt1")).toBeInTheDocument();
    expect(screen.queryByTestId("dlq-row-evt2")).not.toBeInTheDocument();
    expect(screen.queryByTestId("dlq-row-evt3")).not.toBeInTheDocument();
  });

  it("calls onRefresh when refresh button clicked", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    render(<DLQMonitor {...defaultProps} onRefresh={onRefresh} />);
    await user.click(screen.getByTestId("dlq-refresh-btn"));
    expect(onRefresh).toHaveBeenCalled();
  });

  it("shows last refreshed timestamp", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-last-refreshed")).toBeInTheDocument();
  });

  it("auto-refreshes every 30s", () => {
    const onRefresh = vi.fn();
    render(<DLQMonitor {...defaultProps} onRefresh={onRefresh} />);
    vi.advanceTimersByTime(30_000);
    expect(onRefresh).toHaveBeenCalled();
  });

  it("renders the table", () => {
    render(<DLQMonitor {...defaultProps} />);
    expect(screen.getByTestId("dlq-table")).toBeInTheDocument();
  });

  it("shows empty state when filter matches nothing", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    render(<DLQMonitor {...defaultProps} events={[failedEvent]} />);
    await user.selectOptions(screen.getByTestId("dlq-status-filter"), "resolved");
    expect(screen.getByTestId("dlq-empty")).toBeInTheDocument();
  });
});
