import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { NotificationPreferencesState } from "./notification-preferences";

// Mock Clerk useAuth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock entity-api save functions.
const mockSavePreferences = vi.fn();
const mockSaveDigest = vi.fn();
vi.mock("@/lib/entity-api", () => ({
  saveNotificationPreferences: (...args: unknown[]) => mockSavePreferences(...args),
  saveDigestSchedule: (...args: unknown[]) => mockSaveDigest(...args),
}));

import { NotificationPreferencesView } from "./notification-preferences-view";
import { buildDefaultPreferences } from "./notification-preferences";

const defaultState: NotificationPreferencesState = {
  preferences: buildDefaultPreferences(),
  digestFrequency: "none",
};

describe("NotificationPreferencesView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockSavePreferences.mockResolvedValue(undefined);
    mockSaveDigest.mockResolvedValue(undefined);
  });

  it("renders the NotificationPreferences component", () => {
    render(<NotificationPreferencesView initialState={defaultState} />);
    expect(screen.getByTestId("notification-preferences")).toBeInTheDocument();
  });

  it("passes initial state to NotificationPreferences", () => {
    const state = { ...defaultState, digestFrequency: "daily" as const };
    render(<NotificationPreferencesView initialState={state} />);
    expect(screen.getByTestId("digest-option-daily").className).toContain("bg-primary");
  });

  it("calls save APIs with token when user saves", async () => {
    const user = userEvent.setup();
    render(<NotificationPreferencesView initialState={defaultState} />);

    // Toggle a preference to enable save.
    await user.click(screen.getByTestId("pref-switch-message-email"));
    await user.click(screen.getByTestId("save-preferences-btn"));

    await waitFor(() => {
      expect(mockSavePreferences).toHaveBeenCalledWith(
        "test-token",
        expect.arrayContaining([
          expect.objectContaining({ notificationType: "message", channel: "email" }),
        ]),
      );
    });
    await waitFor(() => {
      expect(mockSaveDigest).toHaveBeenCalledWith("test-token", "none");
    });
  });

  it("shows success message after save", async () => {
    const user = userEvent.setup();
    render(<NotificationPreferencesView initialState={defaultState} />);

    await user.click(screen.getByTestId("pref-switch-mention-in_app"));
    await user.click(screen.getByTestId("save-preferences-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("save-success-message")).toBeInTheDocument();
    });
  });

  it("handles null token gracefully", async () => {
    mockGetToken.mockResolvedValue(null);
    const user = userEvent.setup();
    render(<NotificationPreferencesView initialState={defaultState} />);

    await user.click(screen.getByTestId("pref-switch-message-email"));
    await user.click(screen.getByTestId("save-preferences-btn"));

    // Should not call save APIs without token.
    await waitFor(() => {
      expect(mockSavePreferences).not.toHaveBeenCalled();
    });
  });
});
