import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Notification } from "@/lib/api-types";

// Mock Clerk useAuth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock useNotifications hook.
const mockMarkRead = vi.fn();
const mockMarkAllRead = vi.fn();
const mockHandleWSNotification = vi.fn();
const mockRefresh = vi.fn();
const mockUseNotifications = vi.fn();
vi.mock("@/hooks/use-notifications", () => ({
  useNotifications: (...args: unknown[]) => mockUseNotifications(...args),
}));

import { NavNotificationBell } from "./nav-notification-bell";

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

describe("NavNotificationBell", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockUseNotifications.mockReturnValue({
      notifications: [],
      unreadCount: 0,
      loading: false,
      error: null,
      markRead: mockMarkRead,
      markAllRead: mockMarkAllRead,
      handleWSNotification: mockHandleWSNotification,
      refresh: mockRefresh,
    });
  });

  it("fetches auth token and passes it to useNotifications", async () => {
    render(<NavNotificationBell />);
    await waitFor(() => {
      expect(mockGetToken).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(mockUseNotifications).toHaveBeenCalledWith(
        expect.objectContaining({ token: "test-token" }),
      );
    });
  });

  it("renders NotificationBell component", async () => {
    render(<NavNotificationBell />);
    await waitFor(() => {
      expect(screen.getByTestId("notification-bell-container")).toBeInTheDocument();
    });
  });

  it("passes unreadCount from hook to NotificationBell", async () => {
    mockUseNotifications.mockReturnValue({
      notifications: [makeNotif()],
      unreadCount: 3,
      loading: false,
      error: null,
      markRead: mockMarkRead,
      markAllRead: mockMarkAllRead,
      handleWSNotification: mockHandleWSNotification,
      refresh: mockRefresh,
    });
    render(<NavNotificationBell />);
    await waitFor(() => {
      expect(screen.getByTestId("notification-bell-badge")).toBeInTheDocument();
      expect(screen.getByTestId("notification-bell-badge")).toHaveTextContent("3");
    });
  });

  it("passes loading state to NotificationBell", () => {
    mockUseNotifications.mockReturnValue({
      notifications: [],
      unreadCount: 0,
      loading: true,
      error: null,
      markRead: mockMarkRead,
      markAllRead: mockMarkAllRead,
      handleWSNotification: mockHandleWSNotification,
      refresh: mockRefresh,
    });
    render(<NavNotificationBell />);
    expect(screen.getByTestId("notification-bell-container")).toBeInTheDocument();
  });

  it("handles null token when not authenticated", async () => {
    mockGetToken.mockResolvedValue(null);
    render(<NavNotificationBell />);
    await waitFor(() => {
      expect(mockUseNotifications).toHaveBeenCalledWith(expect.objectContaining({ token: null }));
    });
  });

  it("disables notifications when token is not yet loaded", () => {
    // On first render, token is null before async getToken resolves.
    render(<NavNotificationBell />);
    // Initial call should have token: null.
    expect(mockUseNotifications).toHaveBeenCalledWith(expect.objectContaining({ token: null }));
  });

  it("passes notifications array to NotificationBell", async () => {
    const notif = makeNotif({ id: "n-42", title: "Test notif" });
    mockUseNotifications.mockReturnValue({
      notifications: [notif],
      unreadCount: 1,
      loading: false,
      error: null,
      markRead: mockMarkRead,
      markAllRead: mockMarkAllRead,
      handleWSNotification: mockHandleWSNotification,
      refresh: mockRefresh,
    });
    render(<NavNotificationBell />);
    // The bell should be rendered with the notification data.
    expect(screen.getByTestId("notification-bell-container")).toBeInTheDocument();
    expect(screen.getByLabelText("1 unread notifications")).toBeInTheDocument();
  });
});
