import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, it, expect, vi } from "vitest";

import { TicketEntryComposer } from "./ticket-entry-composer";
import { NotificationPrefs } from "./notification-prefs";
const mockGetToken = vi.fn().mockResolvedValue("tok");

vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

const mockCreateTicketEntry = vi.fn().mockResolvedValue({});
const mockSetTicketNotificationPref = vi.fn().mockResolvedValue(undefined);
vi.mock("@/lib/support-api", () => ({
  createTicketEntry: (...args: unknown[]) => mockCreateTicketEntry(...args),
  setTicketNotificationPref: (...args: unknown[]) => mockSetTicketNotificationPref(...args),
}));

vi.mock("@/lib/global-api", () => ({
  uploadThreadAttachment: vi.fn().mockResolvedValue({}),
  downloadUpload: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/components/editor/message-editor", () => ({
  MessageEditor: ({
    placeholder,
    onChange,
  }: {
    placeholder?: string;
    onChange?: (content: string) => void;
  }) => (
    <div data-testid="message-editor">
      {placeholder}
      <button data-testid="editor-fill-btn" onClick={() => onChange?.("<p>hello</p>")}>
        fill
      </button>
    </div>
  ),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mockGetToken.mockResolvedValue("tok");
});

describe("TicketEntryComposer", () => {
  it("shows requestor message and draft options for non-DEFT users", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("entry-type-btn-customer")).toBeInTheDocument();
    expect(screen.getByTestId("entry-type-btn-draft")).toBeInTheDocument();
    expect(screen.queryByTestId("entry-type-btn-agent_reply")).not.toBeInTheDocument();
  });

  it("shows DEFT-only checkbox only for DEFT agent reply type", async () => {
    const user = userEvent.setup();
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("deft-only-checkbox")).toBeInTheDocument();
    await user.click(screen.getByTestId("entry-type-btn-context"));
    expect(screen.queryByTestId("deft-only-checkbox")).not.toBeInTheDocument();
  });

  it("renders editor and submit button", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("message-editor")).toBeInTheDocument();
    expect(screen.getByTestId("composer-submit-btn")).toBeInTheDocument();
  });

  it("submits composer payload for DEFT agent reply", async () => {
    const user = userEvent.setup();
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={true} />);
    await user.click(screen.getByTestId("editor-fill-btn"));
    await user.click(screen.getByTestId("composer-submit-btn"));
    expect(mockCreateTicketEntry).toHaveBeenCalledWith(
      "tok",
      "t1",
      expect.objectContaining({ type: "agent_reply", is_deft_only: false }),
    );
  });

  it("forces context entries to DEFT-only", async () => {
    const user = userEvent.setup();
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={true} />);
    await user.click(screen.getByTestId("entry-type-btn-context"));
    await user.click(screen.getByTestId("editor-fill-btn"));
    await user.click(screen.getByTestId("composer-submit-btn"));
    expect(mockCreateTicketEntry).toHaveBeenCalledWith(
      "tok",
      "t1",
      expect.objectContaining({ type: "context", is_deft_only: true }),
    );
  });
});

describe("NotificationPrefs", () => {
  it("does not save when selecting current preference", async () => {
    const user = userEvent.setup();
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    await user.click(screen.getByTestId("notif-pref-full"));
    expect(mockSetTicketNotificationPref).not.toHaveBeenCalled();
  });
  it("renders both full and privacy options", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    expect(screen.getByTestId("notif-pref-full")).toBeInTheDocument();
    expect(screen.getByTestId("notif-pref-privacy")).toBeInTheDocument();
  });

  it("saves preference change and toggles active state", async () => {
    const user = userEvent.setup();
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    await user.click(screen.getByTestId("notif-pref-privacy"));
    expect(mockSetTicketNotificationPref).toHaveBeenCalledWith("tok", "t1", "privacy");
  });

  it("shows fallback error when preference save fails with non-Error", async () => {
    const user = userEvent.setup();
    mockSetTicketNotificationPref.mockRejectedValueOnce("oops");
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    await user.click(screen.getByTestId("notif-pref-privacy"));
    expect(await screen.findByTestId("notif-prefs-error")).toHaveTextContent(
      "Failed to save preference",
    );
  });

  it("does not save when auth token is unavailable", async () => {
    const user = userEvent.setup();
    mockGetToken.mockResolvedValueOnce(null);
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    await user.click(screen.getByTestId("notif-pref-privacy"));
    expect(mockSetTicketNotificationPref).not.toHaveBeenCalledWith(
      expect.anything(),
      "t1",
      "privacy",
    );
  });
});
