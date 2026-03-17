import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { Thread } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock global-api createSupportTicket.
const mockCreateSupportTicket = vi.fn();
vi.mock("@/lib/global-api", () => ({
  createSupportTicket: (...args: unknown[]) => mockCreateSupportTicket(...args),
}));

import { SupportView } from "./support-view";

/** Minimal Thread fixture factory. */
function makeTicket(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "t1",
    title: "Cannot log in",
    body: "Details here",
    status: "open",
    author_id: "u1",
    channel_id: "global-support",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("SupportView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
  });

  // ---------------------------------------------------------------------------
  // Rendering
  // ---------------------------------------------------------------------------

  it("renders the support view container", () => {
    render(<SupportView initialTickets={[]} />);
    expect(screen.getByTestId("support-view")).toBeInTheDocument();
  });

  it("shows empty state when no tickets", () => {
    render(<SupportView initialTickets={[]} />);
    expect(screen.getByTestId("support-empty")).toBeInTheDocument();
  });

  it("does not show ticket list when tickets are empty", () => {
    render(<SupportView initialTickets={[]} />);
    expect(screen.queryByTestId("support-ticket-list")).not.toBeInTheDocument();
  });

  it("renders ticket list when tickets are provided", () => {
    const tickets = [
      makeTicket(),
      makeTicket({ id: "t2", title: "Billing issue", status: "pending" }),
    ];
    render(<SupportView initialTickets={tickets} />);
    expect(screen.getByTestId("support-ticket-list")).toBeInTheDocument();
    expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
    expect(screen.getByTestId("ticket-row-t2")).toBeInTheDocument();
  });

  it("renders ticket titles in list rows", () => {
    const tickets = [makeTicket({ id: "t1", title: "Cannot log in" })];
    render(<SupportView initialTickets={tickets} />);
    expect(screen.getByTestId("ticket-row-t1")).toHaveTextContent("Cannot log in");
  });

  it("does not show empty state when tickets are present", () => {
    render(<SupportView initialTickets={[makeTicket()]} />);
    expect(screen.queryByTestId("support-empty")).not.toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // Status badges
  // ---------------------------------------------------------------------------

  it("renders open status badge with yellow styling", () => {
    render(<SupportView initialTickets={[makeTicket({ id: "t1", status: "open" })]} />);
    const badge = screen.getByTestId("ticket-status-t1");
    expect(badge).toHaveTextContent("open");
    expect(badge).toHaveClass("bg-yellow-100");
  });

  it("renders pending status badge with blue styling", () => {
    render(<SupportView initialTickets={[makeTicket({ id: "t1", status: "pending" })]} />);
    const badge = screen.getByTestId("ticket-status-t1");
    expect(badge).toHaveTextContent("pending");
    expect(badge).toHaveClass("bg-blue-100");
  });

  it("renders resolved status badge with green styling", () => {
    render(<SupportView initialTickets={[makeTicket({ id: "t1", status: "resolved" })]} />);
    const badge = screen.getByTestId("ticket-status-t1");
    expect(badge).toHaveTextContent("resolved");
    expect(badge).toHaveClass("bg-green-100");
  });

  it("renders closed status badge with gray styling", () => {
    render(<SupportView initialTickets={[makeTicket({ id: "t1", status: "closed" })]} />);
    const badge = screen.getByTestId("ticket-status-t1");
    expect(badge).toHaveTextContent("closed");
    expect(badge).toHaveClass("bg-gray-100");
  });

  it("falls back to open styling for unknown status", () => {
    render(<SupportView initialTickets={[makeTicket({ id: "t1", status: "unknown-status" })]} />);
    const badge = screen.getByTestId("ticket-status-t1");
    expect(badge).toHaveClass("bg-yellow-100");
  });

  it("defaults status to open when ticket status is undefined", () => {
    const ticket = makeTicket({ id: "t1" });
    delete (ticket as Partial<Thread>).status;
    render(<SupportView initialTickets={[ticket]} />);
    const badge = screen.getByTestId("ticket-status-t1");
    expect(badge).toHaveTextContent("open");
    expect(badge).toHaveClass("bg-yellow-100");
  });

  // ---------------------------------------------------------------------------
  // Create form toggle
  // ---------------------------------------------------------------------------

  it("does not show create form by default", () => {
    render(<SupportView initialTickets={[]} />);
    expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
  });

  it("shows create form when New Ticket button is clicked", async () => {
    const user = userEvent.setup();
    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.getByTestId("create-ticket-form")).toBeInTheDocument();
  });

  it("toggles create form closed when button is clicked again", async () => {
    const user = userEvent.setup();
    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.getByTestId("create-ticket-form")).toBeInTheDocument();
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
  });

  it("cancel button hides the create form", async () => {
    const user = userEvent.setup();
    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.click(screen.getByTestId("ticket-cancel-btn"));
    expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
  });

  it("cancel button clears title and body inputs", async () => {
    const user = userEvent.setup();
    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "My title");
    await user.type(screen.getByTestId("ticket-body-input"), "My body");
    await user.click(screen.getByTestId("ticket-cancel-btn"));
    // Re-open form to verify inputs were cleared.
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.getByTestId("ticket-title-input")).toHaveValue("");
    expect(screen.getByTestId("ticket-body-input")).toHaveValue("");
  });

  // ---------------------------------------------------------------------------
  // Submit — success
  // ---------------------------------------------------------------------------

  it("submit button is disabled when title is empty", async () => {
    const user = userEvent.setup();
    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.getByTestId("ticket-submit-btn")).toBeDisabled();
  });

  it("submit button is enabled when title is filled", async () => {
    const user = userEvent.setup();
    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Some title");
    expect(screen.getByTestId("ticket-submit-btn")).not.toBeDisabled();
  });

  it("calls createSupportTicket with correct args on submit", async () => {
    const user = userEvent.setup();
    const newTicket = makeTicket({ id: "t99", title: "Help me", status: "open" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Help me");
    await user.type(screen.getByTestId("ticket-body-input"), "Please assist");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).toHaveBeenCalledWith("test-token", {
        title: "Help me",
        body: "Please assist",
      });
    });
  });

  it("omits body when body is blank", async () => {
    const user = userEvent.setup();
    const newTicket = makeTicket({ id: "t99", title: "No body" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "No body");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).toHaveBeenCalledWith("test-token", {
        title: "No body",
        body: undefined,
      });
    });
  });

  it("prepends new ticket to list on successful submit", async () => {
    const user = userEvent.setup();
    const existing = makeTicket({ id: "t1", title: "Old ticket" });
    const newTicket = makeTicket({ id: "t99", title: "New ticket", status: "open" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportView initialTickets={[existing]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "New ticket");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t99")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
    });

    // New ticket should appear before old ticket in the DOM.
    const rows = screen.getAllByTestId(/^ticket-row-/);
    expect(rows[0]).toHaveAttribute("data-testid", "ticket-row-t99");
    expect(rows[1]).toHaveAttribute("data-testid", "ticket-row-t1");
  });

  it("closes the form and clears inputs after successful submit", async () => {
    const user = userEvent.setup();
    const newTicket = makeTicket({ id: "t99", title: "Fixed" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Fixed");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Submit — error handling
  // ---------------------------------------------------------------------------

  it("shows error banner when createSupportTicket rejects", async () => {
    const user = userEvent.setup();
    mockCreateSupportTicket.mockRejectedValue(new Error("Server error"));

    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Broken");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("support-error")).toHaveTextContent("Server error");
    });
  });

  it("shows fallback error message for non-Error rejections", async () => {
    const user = userEvent.setup();
    mockCreateSupportTicket.mockRejectedValue("oops");

    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Oops");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("support-error")).toHaveTextContent("Failed to create ticket");
    });
  });

  it("does not submit when getToken returns null", async () => {
    const user = userEvent.setup();
    mockGetToken.mockResolvedValue(null);

    render(<SupportView initialTickets={[]} />);
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Token gone");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).not.toHaveBeenCalled();
    });
  });
});
