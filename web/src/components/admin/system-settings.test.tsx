import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { SystemSettings } from "./system-settings";

// Mock api-client for PATCH calls.
const mockClientMutate = vi.fn();
vi.mock("@/lib/api-client", () => ({
  clientMutate: (...args: unknown[]) => mockClientMutate(...args),
}));

// Mock @clerk/nextjs to provide useAuth with getToken.
const mockGetToken = vi.fn().mockResolvedValue("test-token");
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

const sampleSettings: Record<string, unknown> = {
  default_pipeline_stages: ["lead", "qualified", "proposal", "closed_won"],
  notification_defaults: {
    digest_frequency: "daily",
    email_from: "noreply@example.com",
  },
  file_upload_limits: {
    max_size: 10485760,
    allowed_types: ["image/png", "application/pdf"],
  },
  webhook_retry_policy: {
    max_attempts: 5,
    backoff_multiplier: 2.0,
  },
  llm_rate_limits: {
    requests_per_minute: 60,
  },
};

describe("SystemSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClientMutate.mockResolvedValue(sampleSettings);
  });

  it("renders the settings form", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    expect(screen.getByTestId("system-settings-form")).toBeInTheDocument();
  });

  it("renders a heading", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    expect(screen.getByText("System Settings")).toBeInTheDocument();
  });

  it("renders a row for each setting key", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    for (const key of Object.keys(sampleSettings)) {
      expect(screen.getByTestId(`setting-row-${key}`)).toBeInTheDocument();
    }
  });

  it("renders setting keys as labels", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    expect(screen.getByText("default_pipeline_stages")).toBeInTheDocument();
    expect(screen.getByText("notification_defaults")).toBeInTheDocument();
  });

  it("renders JSON textarea for object/array values", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    const row = screen.getByTestId("setting-row-notification_defaults");
    const textarea = within(row).getByTestId("setting-input-notification_defaults");
    expect(textarea.tagName).toBe("TEXTAREA");
  });

  it("renders JSON textarea for array values", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    const row = screen.getByTestId("setting-row-default_pipeline_stages");
    const textarea = within(row).getByTestId("setting-input-default_pipeline_stages");
    expect(textarea.tagName).toBe("TEXTAREA");
  });

  it("renders text input for string values", () => {
    const stringSettings = { site_name: "My CRM" };
    render(<SystemSettings initialSettings={stringSettings} />);
    const input = screen.getByTestId("setting-input-site_name");
    expect(input.tagName).toBe("INPUT");
    expect(input).toHaveAttribute("type", "text");
    expect(input).toHaveValue("My CRM");
  });

  it("renders number input for number values", () => {
    const numberSettings = { max_users: 100 };
    render(<SystemSettings initialSettings={numberSettings} />);
    const input = screen.getByTestId("setting-input-max_users");
    expect(input.tagName).toBe("INPUT");
    expect(input).toHaveAttribute("type", "number");
    expect(input).toHaveValue(100);
  });

  it("renders toggle for boolean values", () => {
    const boolSettings = { maintenance_mode: true };
    render(<SystemSettings initialSettings={boolSettings} />);
    const toggle = screen.getByTestId("setting-input-maintenance_mode");
    expect(toggle).toHaveAttribute("role", "switch");
    expect(toggle).toHaveAttribute("aria-checked", "true");
  });

  it("toggles boolean value on click", async () => {
    const user = userEvent.setup();
    const boolSettings = { maintenance_mode: true };
    render(<SystemSettings initialSettings={boolSettings} />);
    const toggle = screen.getByTestId("setting-input-maintenance_mode");
    expect(toggle).toHaveAttribute("aria-checked", "true");
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-checked", "false");
  });

  it("renders save button", () => {
    render(<SystemSettings initialSettings={sampleSettings} />);
    expect(screen.getByTestId("settings-save-btn")).toBeInTheDocument();
    expect(screen.getByTestId("settings-save-btn")).toHaveTextContent("Save");
  });

  it("calls PATCH /admin/settings on save", async () => {
    const user = userEvent.setup();
    render(<SystemSettings initialSettings={sampleSettings} />);
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(mockClientMutate).toHaveBeenCalledWith(
      "PATCH",
      "/admin/settings",
      expect.objectContaining({
        token: "test-token",
        body: expect.any(Object),
      }),
    );
  });

  it("shows success toast after save", async () => {
    const user = userEvent.setup();
    render(<SystemSettings initialSettings={sampleSettings} />);
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(await screen.findByTestId("settings-toast")).toHaveTextContent("Settings saved");
  });

  it("shows error message on save failure", async () => {
    const user = userEvent.setup();
    mockClientMutate.mockRejectedValue(new Error("Server error"));
    render(<SystemSettings initialSettings={sampleSettings} />);
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(await screen.findByTestId("settings-error")).toHaveTextContent("Server error");
  });

  it("shows saving state on button during save", async () => {
    const user = userEvent.setup();
    let resolvePromise: (() => void) | undefined;
    mockClientMutate.mockImplementation(
      () =>
        new Promise<Record<string, unknown>>((resolve) => {
          resolvePromise = () => resolve(sampleSettings);
        }),
    );
    render(<SystemSettings initialSettings={sampleSettings} />);
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(screen.getByTestId("settings-save-btn")).toHaveTextContent("Saving...");
    resolvePromise?.();
  });

  it("can edit a text input value", async () => {
    const user = userEvent.setup();
    const stringSettings = { site_name: "My CRM" };
    render(<SystemSettings initialSettings={stringSettings} />);
    const input = screen.getByTestId("setting-input-site_name");
    await user.clear(input);
    await user.type(input, "New CRM");
    expect(input).toHaveValue("New CRM");
  });

  it("can edit a textarea value", async () => {
    const user = userEvent.setup();
    render(<SystemSettings initialSettings={sampleSettings} />);
    const textarea = screen.getByTestId(
      "setting-input-notification_defaults",
    ) as HTMLTextAreaElement;
    await user.clear(textarea);
    await user.type(textarea, '{{"test": true}}');
    expect(textarea.value).toContain("test");
  });

  it("sends correct body with edited string value", async () => {
    const user = userEvent.setup();
    const stringSettings = { site_name: "My CRM" };
    render(<SystemSettings initialSettings={stringSettings} />);
    const input = screen.getByTestId("setting-input-site_name");
    await user.clear(input);
    await user.type(input, "New CRM");
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(mockClientMutate).toHaveBeenCalledWith(
      "PATCH",
      "/admin/settings",
      expect.objectContaining({
        body: { site_name: "New CRM" },
      }),
    );
  });

  it("sends correct body with edited number value", async () => {
    const user = userEvent.setup();
    const numberSettings = { max_users: 100 };
    render(<SystemSettings initialSettings={numberSettings} />);
    const input = screen.getByTestId("setting-input-max_users");
    await user.clear(input);
    await user.type(input, "200");
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(mockClientMutate).toHaveBeenCalledWith(
      "PATCH",
      "/admin/settings",
      expect.objectContaining({
        body: { max_users: 200 },
      }),
    );
  });

  it("sends correct body with toggled boolean", async () => {
    const user = userEvent.setup();
    const boolSettings = { maintenance_mode: false };
    render(<SystemSettings initialSettings={boolSettings} />);
    await user.click(screen.getByTestId("setting-input-maintenance_mode"));
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(mockClientMutate).toHaveBeenCalledWith(
      "PATCH",
      "/admin/settings",
      expect.objectContaining({
        body: { maintenance_mode: true },
      }),
    );
  });

  it("renders empty state when no settings", () => {
    render(<SystemSettings initialSettings={{}} />);
    expect(screen.getByTestId("system-settings-form")).toBeInTheDocument();
    expect(screen.getByTestId("settings-empty")).toBeInTheDocument();
  });

  it("hides toast after 3 seconds", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<SystemSettings initialSettings={sampleSettings} />);
    await user.click(screen.getByTestId("settings-save-btn"));
    expect(await screen.findByTestId("settings-toast")).toBeInTheDocument();
    await act(async () => {
      vi.advanceTimersByTime(3100);
    });
    expect(screen.queryByTestId("settings-toast")).not.toBeInTheDocument();
    vi.useRealTimers();
  });
});
