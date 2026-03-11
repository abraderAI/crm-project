import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { NotificationFeed, formatRelativeTime } from "./notification-feed";
import type { Notification } from "@/lib/api-types";

const makeNotif = (overrides: Partial<Notification> = {}): Notification => ({
  id: "n-1",
  user_id: "u-1",
  type: "message",
  title: "New message from Alice",
  body: "Hey, check this out!",
  entity_type: "thread",
  entity_id: "t-1",
  is_read: false,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  ...overrides,
});

describe("formatRelativeTime", () => {
  it("returns 'just now' for recent timestamps", () => {
    const now = new Date().toISOString();
    expect(formatRelativeTime(now)).toBe("just now");
  });

  it("returns minutes ago", () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    expect(formatRelativeTime(fiveMinAgo)).toBe("5m ago");
  });

  it("returns hours ago", () => {
    const twoHrAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(twoHrAgo)).toBe("2h ago");
  });

  it("returns days ago", () => {
    const threeDayAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(threeDayAgo)).toBe("3d ago");
  });

  it("returns formatted date for older timestamps", () => {
    const oldDate = new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString();
    const result = formatRelativeTime(oldDate);
    // Should be a locale date string (e.g. "2/25/2026").
    expect(result).not.toContain("ago");
  });
});

describe("NotificationFeed", () => {
  it("renders feed container", () => {
    render(<NotificationFeed notifications={[]} />);
    expect(screen.getByTestId("notification-feed")).toBeInTheDocument();
  });

  it("renders header with title", () => {
    render(<NotificationFeed notifications={[]} />);
    expect(screen.getByText("Notifications")).toBeInTheDocument();
  });

  it("renders empty state when no notifications", () => {
    render(<NotificationFeed notifications={[]} />);
    expect(screen.getByTestId("notification-feed-empty")).toBeInTheDocument();
    expect(screen.getByText("No notifications yet.")).toBeInTheDocument();
  });

  it("renders loading state", () => {
    render(<NotificationFeed notifications={[]} loading={true} />);
    expect(screen.getByTestId("notification-feed-loading")).toBeInTheDocument();
  });

  it("renders notification items", () => {
    const notif = makeNotif();
    render(<NotificationFeed notifications={[notif]} />);
    expect(screen.getByTestId("notification-item-n-1")).toBeInTheDocument();
    expect(screen.getByTestId("notification-title-n-1")).toHaveTextContent(
      "New message from Alice",
    );
  });

  it("renders notification body", () => {
    const notif = makeNotif();
    render(<NotificationFeed notifications={[notif]} />);
    expect(screen.getByTestId("notification-body-n-1")).toHaveTextContent("Hey, check this out!");
  });

  it("shows unread dot for unread notifications", () => {
    const notif = makeNotif({ is_read: false });
    render(<NotificationFeed notifications={[notif]} />);
    expect(screen.getByTestId("notification-unread-dot-n-1")).toBeInTheDocument();
  });

  it("hides unread dot for read notifications", () => {
    const notif = makeNotif({ id: "n-2", is_read: true });
    render(<NotificationFeed notifications={[notif]} />);
    expect(screen.queryByTestId("notification-unread-dot-n-2")).not.toBeInTheDocument();
  });

  it("shows mark all read button when there are unread notifications", () => {
    const notif = makeNotif({ is_read: false });
    render(<NotificationFeed notifications={[notif]} onMarkAllRead={vi.fn()} />);
    expect(screen.getByTestId("mark-all-read-btn")).toBeInTheDocument();
  });

  it("hides mark all read button when all are read", () => {
    const notif = makeNotif({ is_read: true });
    render(<NotificationFeed notifications={[notif]} onMarkAllRead={vi.fn()} />);
    expect(screen.queryByTestId("mark-all-read-btn")).not.toBeInTheDocument();
  });

  it("calls onMarkRead when clicking a notification", async () => {
    const user = userEvent.setup();
    const onMarkRead = vi.fn();
    const notif = makeNotif();
    render(<NotificationFeed notifications={[notif]} onMarkRead={onMarkRead} />);

    await user.click(screen.getByTestId("notification-item-n-1"));
    expect(onMarkRead).toHaveBeenCalledWith("n-1");
  });

  it("calls onMarkAllRead when clicking mark all read", async () => {
    const user = userEvent.setup();
    const onMarkAllRead = vi.fn();
    const notif = makeNotif();
    render(<NotificationFeed notifications={[notif]} onMarkAllRead={onMarkAllRead} />);

    await user.click(screen.getByTestId("mark-all-read-btn"));
    expect(onMarkAllRead).toHaveBeenCalledOnce();
  });

  it("renders notifications without body", () => {
    const notif = makeNotif({ body: undefined });
    render(<NotificationFeed notifications={[notif]} />);
    expect(screen.queryByTestId("notification-body-n-1")).not.toBeInTheDocument();
  });

  it("renders multiple notifications", () => {
    const notifs = [
      makeNotif({ id: "n-1" }),
      makeNotif({ id: "n-2", title: "Second notification" }),
    ];
    render(<NotificationFeed notifications={notifs} />);
    expect(screen.getByTestId("notification-item-n-1")).toBeInTheDocument();
    expect(screen.getByTestId("notification-item-n-2")).toBeInTheDocument();
  });
});
