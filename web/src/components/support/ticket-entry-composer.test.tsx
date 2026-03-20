import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { TicketEntryComposer } from "./ticket-entry-composer";
import { NotificationPrefs } from "./notification-prefs";

// Mock Clerk auth.
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue("tok") }),
}));

// Mock API.
vi.mock("@/lib/support-api", () => ({
  createTicketEntry: vi.fn().mockResolvedValue({}),
  setTicketNotificationPref: vi.fn().mockResolvedValue(undefined),
}));

// Mock MessageEditor to avoid Tiptap JSDOM issues.
vi.mock("@/components/editor/message-editor", () => ({
  MessageEditor: ({ placeholder }: { placeholder?: string }) => (
    <div data-testid="message-editor">{placeholder}</div>
  ),
}));

// ── TicketEntryComposer ──────────────────────────────────────────────────────

describe("TicketEntryComposer", () => {
  it("renders submit button", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("composer-submit-btn")).toBeInTheDocument();
  });

  it("hides entry type selector for non-DEFT members", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={false} />);
    expect(screen.queryByTestId("entry-type-btn-agent_reply")).toBeNull();
  });

  it("shows all 5 entry type buttons for DEFT members", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("entry-type-btn-agent_reply")).toBeInTheDocument();
    expect(screen.getByTestId("entry-type-btn-draft")).toBeInTheDocument();
    expect(screen.getByTestId("entry-type-btn-context")).toBeInTheDocument();
    expect(screen.getByTestId("entry-type-btn-customer")).toBeInTheDocument();
    expect(screen.getByTestId("entry-type-btn-system_event")).toBeInTheDocument();
  });

  it("shows DEFT-only checkbox for DEFT members on non-context types", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={true} />);
    expect(screen.getByTestId("deft-only-checkbox")).toBeInTheDocument();
  });

  it("hides DEFT-only checkbox for non-DEFT members", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={false} />);
    expect(screen.queryByTestId("deft-only-checkbox")).toBeNull();
  });

  it("renders the message editor", () => {
    render(<TicketEntryComposer ticketSlug="t1" isDeftMember={false} />);
    expect(screen.getByTestId("message-editor")).toBeInTheDocument();
  });
});

// ── NotificationPrefs ────────────────────────────────────────────────────────

describe("NotificationPrefs", () => {
  it("renders both preference options", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    expect(screen.getByTestId("notif-pref-full")).toBeInTheDocument();
    expect(screen.getByTestId("notif-pref-privacy")).toBeInTheDocument();
  });

  it("renders with full level initially selected", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    // The full button should have the selected styles (border-primary class present).
    const fullBtn = screen.getByTestId("notif-pref-full");
    expect(fullBtn.className).toContain("border-primary");
  });

  it("renders with privacy level initially selected", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="privacy" />);
    const privacyBtn = screen.getByTestId("notif-pref-privacy");
    expect(privacyBtn.className).toContain("border-primary");
  });

  it("shows heading text", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    expect(screen.getByTestId("notification-prefs")).toHaveTextContent("Email notifications");
  });

  it("displays full detail description", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="full" />);
    expect(screen.getByTestId("notif-pref-full")).toHaveTextContent("Full detail");
  });

  it("displays privacy mode description", () => {
    render(<NotificationPrefs ticketSlug="t1" currentLevel="privacy" />);
    expect(screen.getByTestId("notif-pref-privacy")).toHaveTextContent("Privacy mode");
  });
});
