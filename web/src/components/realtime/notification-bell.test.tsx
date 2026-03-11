import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { NotificationBell } from "./notification-bell";
import type { Notification } from "@/lib/api-types";

const makeNotif = (overrides: Partial<Notification> = {}): Notification => ({
  id: "n-1",
  user_id: "u-1",
  type: "message",
  title: "New message",
  is_read: false,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  ...overrides,
});

describe("NotificationBell", () => {
  it("renders the bell button", () => {
    render(<NotificationBell notifications={[]} unreadCount={0} />);
    expect(screen.getByTestId("notification-bell-btn")).toBeInTheDocument();
  });

  it("shows badge when unreadCount > 0", () => {
    render(<NotificationBell notifications={[]} unreadCount={5} />);
    const badge = screen.getByTestId("notification-bell-badge");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent("5");
  });

  it("does not show badge when unreadCount is 0", () => {
    render(<NotificationBell notifications={[]} unreadCount={0} />);
    expect(screen.queryByTestId("notification-bell-badge")).not.toBeInTheDocument();
  });

  it("caps badge at 99+", () => {
    render(<NotificationBell notifications={[]} unreadCount={150} />);
    expect(screen.getByTestId("notification-bell-badge")).toHaveTextContent("99+");
  });

  it("has correct aria-label with unread count", () => {
    render(<NotificationBell notifications={[]} unreadCount={3} />);
    expect(screen.getByLabelText("3 unread notifications")).toBeInTheDocument();
  });

  it("has correct aria-label with zero count", () => {
    render(<NotificationBell notifications={[]} unreadCount={0} />);
    expect(screen.getByLabelText("No unread notifications")).toBeInTheDocument();
  });

  it("toggles dropdown on click", async () => {
    const user = userEvent.setup();
    render(<NotificationBell notifications={[]} unreadCount={0} />);

    // Initially closed.
    expect(screen.queryByTestId("notification-dropdown")).not.toBeInTheDocument();

    // Click to open.
    await user.click(screen.getByTestId("notification-bell-btn"));
    expect(screen.getByTestId("notification-dropdown")).toBeInTheDocument();

    // Click again to close.
    await user.click(screen.getByTestId("notification-bell-btn"));
    expect(screen.queryByTestId("notification-dropdown")).not.toBeInTheDocument();
  });

  it("shows notification feed in dropdown", async () => {
    const user = userEvent.setup();
    const notif = makeNotif();
    render(<NotificationBell notifications={[notif]} unreadCount={1} />);

    await user.click(screen.getByTestId("notification-bell-btn"));
    expect(screen.getByTestId("notification-feed")).toBeInTheDocument();
    expect(screen.getByTestId("notification-item-n-1")).toBeInTheDocument();
  });

  it("closes dropdown on Escape key", async () => {
    const user = userEvent.setup();
    render(<NotificationBell notifications={[]} unreadCount={0} />);

    await user.click(screen.getByTestId("notification-bell-btn"));
    expect(screen.getByTestId("notification-dropdown")).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.queryByTestId("notification-dropdown")).not.toBeInTheDocument();
  });

  it("closes dropdown on outside click", async () => {
    const user = userEvent.setup();
    render(
      <div>
        <NotificationBell notifications={[]} unreadCount={0} />
        <div data-testid="outside">Outside</div>
      </div>,
    );

    await user.click(screen.getByTestId("notification-bell-btn"));
    expect(screen.getByTestId("notification-dropdown")).toBeInTheDocument();

    await user.click(screen.getByTestId("outside"));
    expect(screen.queryByTestId("notification-dropdown")).not.toBeInTheDocument();
  });

  it("has aria-expanded attribute", async () => {
    const user = userEvent.setup();
    render(<NotificationBell notifications={[]} unreadCount={0} />);

    const btn = screen.getByTestId("notification-bell-btn");
    expect(btn).toHaveAttribute("aria-expanded", "false");

    await user.click(btn);
    expect(btn).toHaveAttribute("aria-expanded", "true");
  });

  it("passes onMarkRead to feed", async () => {
    const user = userEvent.setup();
    const onMarkRead = vi.fn();
    const notif = makeNotif();
    render(<NotificationBell notifications={[notif]} unreadCount={1} onMarkRead={onMarkRead} />);

    await user.click(screen.getByTestId("notification-bell-btn"));
    await user.click(screen.getByTestId("notification-item-n-1"));
    expect(onMarkRead).toHaveBeenCalledWith("n-1");
  });

  it("passes onMarkAllRead to feed", async () => {
    const user = userEvent.setup();
    const onMarkAllRead = vi.fn();
    const notif = makeNotif();
    render(
      <NotificationBell notifications={[notif]} unreadCount={1} onMarkAllRead={onMarkAllRead} />,
    );

    await user.click(screen.getByTestId("notification-bell-btn"));
    await user.click(screen.getByTestId("mark-all-read-btn"));
    expect(onMarkAllRead).toHaveBeenCalledOnce();
  });
});
