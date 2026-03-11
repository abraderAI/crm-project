import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import {
  NotificationPreferences,
  buildDefaultPreferences,
  type NotificationPreferencesState,
} from "./notification-preferences";

const defaultState: NotificationPreferencesState = {
  preferences: buildDefaultPreferences(),
  digestFrequency: "none",
};

describe("buildDefaultPreferences", () => {
  it("creates 8 preferences (4 types × 2 channels)", () => {
    const prefs = buildDefaultPreferences();
    expect(prefs).toHaveLength(8);
  });

  it("all defaults are enabled", () => {
    const prefs = buildDefaultPreferences();
    expect(prefs.every((p) => p.enabled)).toBe(true);
  });

  it("includes all notification types", () => {
    const prefs = buildDefaultPreferences();
    const types = [...new Set(prefs.map((p) => p.notificationType))];
    expect(types).toEqual(["message", "mention", "stage_change", "assignment"]);
  });

  it("includes both channels", () => {
    const prefs = buildDefaultPreferences();
    const channels = [...new Set(prefs.map((p) => p.channel))];
    expect(channels).toEqual(["in_app", "email"]);
  });
});

describe("NotificationPreferences", () => {
  it("renders the container", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);
    expect(screen.getByTestId("notification-preferences")).toBeInTheDocument();
  });

  it("renders heading", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);
    expect(screen.getByText("Notification Preferences")).toBeInTheDocument();
  });

  it("renders all preference type sections", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);
    expect(screen.getByTestId("pref-type-message")).toHaveTextContent("New Messages");
    expect(screen.getByTestId("pref-type-mention")).toHaveTextContent("Mentions");
    expect(screen.getByTestId("pref-type-stage_change")).toHaveTextContent(
      "Pipeline Stage Changes",
    );
    expect(screen.getByTestId("pref-type-assignment")).toHaveTextContent("Assignments");
  });

  it("renders toggle switches for each type/channel", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);
    expect(screen.getByTestId("pref-switch-message-in_app")).toBeInTheDocument();
    expect(screen.getByTestId("pref-switch-message-email")).toBeInTheDocument();
    expect(screen.getByTestId("pref-switch-mention-in_app")).toBeInTheDocument();
    expect(screen.getByTestId("pref-switch-mention-email")).toBeInTheDocument();
  });

  it("toggles preference on click", async () => {
    const user = userEvent.setup();
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);

    const sw = screen.getByTestId("pref-switch-message-email");
    expect(sw).toHaveAttribute("aria-checked", "true");

    await user.click(sw);
    expect(sw).toHaveAttribute("aria-checked", "false");
  });

  it("renders digest frequency section", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);
    expect(screen.getByTestId("digest-frequency-section")).toBeInTheDocument();
    expect(screen.getByTestId("digest-option-none")).toBeInTheDocument();
    expect(screen.getByTestId("digest-option-daily")).toBeInTheDocument();
    expect(screen.getByTestId("digest-option-weekly")).toBeInTheDocument();
  });

  it("highlights selected digest frequency", () => {
    const state = { ...defaultState, digestFrequency: "daily" as const };
    render(<NotificationPreferences initialState={state} onSave={vi.fn()} />);
    // The daily button should have the primary styles.
    const daily = screen.getByTestId("digest-option-daily");
    expect(daily.className).toContain("bg-primary");
  });

  it("changes digest frequency on click", async () => {
    const user = userEvent.setup();
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);

    await user.click(screen.getByTestId("digest-option-weekly"));
    // After clicking, weekly should now be primary styled.
    expect(screen.getByTestId("digest-option-weekly").className).toContain("bg-primary");
  });

  it("save button is disabled when no changes", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);
    expect(screen.getByTestId("save-preferences-btn")).toBeDisabled();
  });

  it("save button is enabled after making changes", async () => {
    const user = userEvent.setup();
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} />);

    await user.click(screen.getByTestId("pref-switch-message-email"));
    expect(screen.getByTestId("save-preferences-btn")).not.toBeDisabled();
  });

  it("calls onSave with updated state", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<NotificationPreferences initialState={defaultState} onSave={onSave} />);

    await user.click(screen.getByTestId("pref-switch-message-email"));
    await user.click(screen.getByTestId("save-preferences-btn"));

    expect(onSave).toHaveBeenCalledOnce();
    const savedState = onSave.mock.calls[0]?.[0] as NotificationPreferencesState;
    expect(savedState.preferences).toBeDefined();
    expect(savedState.digestFrequency).toBe("none");
    // The message/email pref should now be disabled.
    const msgEmail = savedState.preferences.find(
      (p) => p.notificationType === "message" && p.channel === "email",
    );
    expect(msgEmail?.enabled).toBe(false);
  });

  it("shows success message after save", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<NotificationPreferences initialState={defaultState} onSave={onSave} />);

    await user.click(screen.getByTestId("pref-switch-message-email"));
    await user.click(screen.getByTestId("save-preferences-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("save-success-message")).toBeInTheDocument();
    });
    expect(screen.getByTestId("save-success-message")).toHaveTextContent("Preferences saved!");
  });

  it("shows saving text when saving prop is true", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} saving={true} />);
    expect(screen.getByTestId("save-preferences-btn")).toHaveTextContent("Saving...");
  });

  it("disables save button when saving is true", () => {
    render(<NotificationPreferences initialState={defaultState} onSave={vi.fn()} saving={true} />);
    expect(screen.getByTestId("save-preferences-btn")).toBeDisabled();
  });

  it("resets dirty state when initialState changes", () => {
    const { rerender } = render(
      <NotificationPreferences initialState={defaultState} onSave={vi.fn()} />,
    );
    // Re-render with new initialState.
    const newState = { ...defaultState, digestFrequency: "weekly" as const };
    rerender(<NotificationPreferences initialState={newState} onSave={vi.fn()} />);
    // Should still be disabled since we just got new initial state.
    expect(screen.getByTestId("save-preferences-btn")).toBeDisabled();
  });
});
