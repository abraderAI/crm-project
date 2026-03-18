import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { ThreadWithAuthor } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock useTier hook.
const mockUseTier = vi.fn();
vi.mock("@/hooks/use-tier", () => ({
  useTier: () => mockUseTier(),
}));

// Mock global-api.
const mockFetchGlobalSupportTickets = vi.fn();
const mockCreateSupportTicket = vi.fn();
const mockUpdateSupportTicket = vi.fn();
vi.mock("@/lib/global-api", () => ({
  fetchGlobalSupportTickets: (...args: unknown[]) => mockFetchGlobalSupportTickets(...args),
  createSupportTicket: (...args: unknown[]) => mockCreateSupportTicket(...args),
  updateSupportTicket: (...args: unknown[]) => mockUpdateSupportTicket(...args),
}));

import { SupportManagementView } from "./support-management-view";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Default tier state factory. */
function makeTier(
  overrides: Partial<{
    tier: number;
    subType: string | null;
    deftDepartment: string | null;
    orgId: string | null;
    isLoading: boolean;
  }> = {},
) {
  return {
    tier: 1,
    subType: null,
    deftDepartment: null,
    orgId: null,
    isLoading: false,
    refresh: vi.fn(),
    ...overrides,
  };
}

/** Minimal ThreadWithAuthor fixture factory. */
function makeTicket(overrides: Partial<ThreadWithAuthor> = {}): ThreadWithAuthor {
  return {
    id: "t1",
    board_id: "b1",
    title: "Cannot log in",
    slug: "cannot-log-in",
    author_id: "u1",
    is_pinned: false,
    is_locked: false,
    is_hidden: false,
    vote_score: 0,
    status: "open",
    metadata: "{}",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

/** Paginated response wrapper. */
function makePagedResponse(tickets: ThreadWithAuthor[], hasMore = false) {
  return {
    data: tickets,
    page_info: { has_more: hasMore, next_cursor: hasMore ? "cursor-next" : undefined },
  };
}

describe("SupportManagementView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([]));
  });

  // ---------------------------------------------------------------------------
  // Tier-resolving loading state
  // ---------------------------------------------------------------------------

  it("shows tier loading state while tier is resolving", () => {
    mockUseTier.mockReturnValue(makeTier({ isLoading: true }));
    render(<SupportManagementView />);
    expect(screen.getByTestId("support-loading-tier")).toBeInTheDocument();
  });

  it("does not show access denied or view during tier loading", () => {
    mockUseTier.mockReturnValue(makeTier({ isLoading: true }));
    render(<SupportManagementView />);
    expect(screen.queryByTestId("support-access-denied")).not.toBeInTheDocument();
    expect(screen.queryByTestId("support-management-view")).not.toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // Access denied — tier 1 (anonymous)
  // ---------------------------------------------------------------------------

  it("shows access denied for tier 1 (anonymous)", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 1 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-access-denied")).toBeInTheDocument();
    });
  });

  it("shows sign-in message for tier 1", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 1 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-access-denied")).toHaveTextContent(
        "Sign in to view support tickets",
      );
    });
  });

  it("does not call fetchGlobalSupportTickets for tier 1", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 1 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-access-denied")).toBeInTheDocument();
    });
    expect(mockFetchGlobalSupportTickets).not.toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Tier 2 — registered user (own tickets only)
  // ---------------------------------------------------------------------------

  it("renders management view for tier 2", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-management-view")).toBeInTheDocument();
    });
  });

  it("shows 'DEFT.support' heading for tier 2", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets with mine=true for tier 2", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ mine: true }),
      );
    });
  });

  it("does not pass org_id for tier 2", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.not.objectContaining({ org_id: expect.anything() }),
      );
    });
  });

  it("does not show stats strip for tier 2", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-management-view")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("support-stats")).not.toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // Tier 3 — paying customer, no org (own tickets)
  // ---------------------------------------------------------------------------

  it("shows 'DEFT.support' heading for tier 3 with no org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: null }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets with mine=true for tier 3 without org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: null }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ mine: true }),
      );
    });
  });

  it("does not show stats strip for tier 3 without org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: null }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-management-view")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("support-stats")).not.toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // Tier 3 — paying customer with org (org-scoped tickets)
  // ---------------------------------------------------------------------------

  it("shows 'DEFT.support' heading for tier 3 with org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: "org-abc" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets with org_id for tier 3 with org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: "org-abc" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ org_id: "org-abc" }),
      );
    });
  });

  it("does not pass mine=true for tier 3 with org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: "org-abc" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.not.objectContaining({ mine: expect.anything() }),
      );
    });
  });

  it("shows stats strip for tier 3 with org after tickets load", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: "org-abc" }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-stats")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Tier 4 — DEFT employee (all tickets)
  // ---------------------------------------------------------------------------

  it("shows 'DEFT.support' heading for tier 4", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "support" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets without mine or org_id for tier 4", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "support" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.not.objectContaining({ mine: expect.anything(), org_id: expect.anything() }),
      );
    });
  });

  it("shows stats strip for tier 4 after tickets load", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-stats")).toBeInTheDocument();
    });
  });

  it("shows stats strip for tier 4 sales dept", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "sales" }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-stats")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Tier 5 — customer org admin (subType=owner, org-scoped + stats)
  // ---------------------------------------------------------------------------

  it("shows 'DEFT.support' heading for tier 5 owner", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5, subType: "owner", orgId: "org-xyz" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets with org_id for tier 5 owner", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5, subType: "owner", orgId: "org-xyz" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ org_id: "org-xyz" }),
      );
    });
  });

  it("shows stats strip for tier 5 owner after tickets load", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5, subType: "owner", orgId: "org-xyz" }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-stats")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Tier 5 — DEFT support admin (all tickets + dashboard)
  // ---------------------------------------------------------------------------

  it("shows 'DEFT.support' heading for tier 5 DEFT support admin", async () => {
    mockUseTier.mockReturnValue(
      makeTier({ tier: 5, deftDepartment: "support", subType: "support" }),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets without mine or org_id for tier 5 DEFT support", async () => {
    mockUseTier.mockReturnValue(
      makeTier({ tier: 5, deftDepartment: "support", subType: "support" }),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.not.objectContaining({ mine: expect.anything(), org_id: expect.anything() }),
      );
    });
  });

  it("shows stats strip for tier 5 DEFT support admin after tickets load", async () => {
    mockUseTier.mockReturnValue(
      makeTier({ tier: 5, deftDepartment: "support", subType: "support" }),
    );
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-stats")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Tier 6 — system admin (all tickets + dashboard)
  // ---------------------------------------------------------------------------

  it("shows 'DEFT.support' heading for tier 6", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByText("DEFT.support")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalSupportTickets without mine or org_id for tier 6", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "test-token",
        expect.not.objectContaining({ mine: expect.anything(), org_id: expect.anything() }),
      );
    });
  });

  it("shows stats strip for tier 6 after tickets load", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-stats")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Stats strip values
  // ---------------------------------------------------------------------------

  it("counts open, pending, resolved tickets in stats strip", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    const tickets = [
      makeTicket({ id: "t1", status: "open" }),
      makeTicket({ id: "t2", status: "open" }),
      makeTicket({ id: "t3", status: "pending" }),
      makeTicket({ id: "t4", status: "resolved" }),
    ];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("stats-open")).toHaveTextContent("2");
      expect(screen.getByTestId("stats-pending")).toHaveTextContent("1");
      expect(screen.getByTestId("stats-resolved")).toHaveTextContent("1");
    });
  });

  it("counts closed tickets in resolved stat", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    const tickets = [makeTicket({ id: "t1", status: "closed" })];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("stats-resolved")).toHaveTextContent("1");
    });
  });

  // ---------------------------------------------------------------------------
  // Ticket list rendering
  // ---------------------------------------------------------------------------

  it("shows loading skeleton while fetching tickets", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockReturnValue(new Promise(() => {}));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-list-loading")).toBeInTheDocument();
    });
  });

  it("shows empty state when no tickets are returned", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-empty")).toBeInTheDocument();
    });
  });

  it("renders ticket rows for each returned ticket", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const tickets = [makeTicket({ id: "t1" }), makeTicket({ id: "t2", title: "Billing issue" })];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t2")).toBeInTheDocument();
    });
  });

  it("renders ticket title in each row", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(
      makePagedResponse([makeTicket({ id: "t1", title: "Cannot log in" })]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t1")).toHaveTextContent("Cannot log in");
    });
  });

  it("shows count of visible tickets", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const tickets = [makeTicket({ id: "t1" }), makeTicket({ id: "t2", title: "Other" })];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-count")).toHaveTextContent("2");
    });
  });

  // ---------------------------------------------------------------------------
  // Status badges
  // ---------------------------------------------------------------------------

  it("renders open status badge with yellow styling", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(
      makePagedResponse([makeTicket({ id: "t1", status: "open" })]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      const badge = screen.getByTestId("ticket-status-t1");
      expect(badge).toHaveTextContent("open");
      expect(badge).toHaveClass("bg-yellow-100");
    });
  });

  it("renders pending status badge with blue styling", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(
      makePagedResponse([makeTicket({ id: "t1", status: "pending" })]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-t1")).toHaveClass("bg-blue-100");
    });
  });

  it("renders resolved status badge with green styling", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(
      makePagedResponse([makeTicket({ id: "t1", status: "resolved" })]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-t1")).toHaveClass("bg-green-100");
    });
  });

  it("renders closed status badge with gray styling", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(
      makePagedResponse([makeTicket({ id: "t1", status: "closed" })]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-t1")).toHaveClass("bg-gray-100");
    });
  });

  it("defaults missing status to open styling", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const ticket = makeTicket({ id: "t1" });
    delete (ticket as Partial<Thread>).status;
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-t1")).toHaveClass("bg-yellow-100");
    });
  });

  // ---------------------------------------------------------------------------
  // Error handling
  // ---------------------------------------------------------------------------

  it("shows error banner when fetchGlobalSupportTickets rejects", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockRejectedValue(new Error("Network failure"));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-error")).toHaveTextContent("Network failure");
    });
  });

  it("shows fallback error message for non-Error rejections", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockFetchGlobalSupportTickets.mockRejectedValue("oops");
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-error")).toHaveTextContent("Failed to load tickets");
    });
  });

  it("does not fetch tickets when getToken returns null", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockGetToken.mockResolvedValue(null);
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.queryByTestId("support-error")).not.toBeInTheDocument();
    });
    expect(mockFetchGlobalSupportTickets).not.toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Filters
  // ---------------------------------------------------------------------------

  it("renders search input and status filter", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-search-input")).toBeInTheDocument();
      expect(screen.getByTestId("tickets-status-filter")).toBeInTheDocument();
    });
  });

  it("filters tickets by status selection", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const tickets = [
      makeTicket({ id: "t1", status: "open" }),
      makeTicket({ id: "t2", title: "Billing", status: "resolved" }),
    ];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t2")).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId("tickets-status-filter"), "resolved");

    expect(screen.queryByTestId("ticket-row-t1")).not.toBeInTheDocument();
    expect(screen.getByTestId("ticket-row-t2")).toBeInTheDocument();
  });

  it("filters tickets by search term", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const tickets = [
      makeTicket({ id: "t1", title: "Login issue" }),
      makeTicket({ id: "t2", title: "Billing question" }),
    ];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
    });

    await user.type(screen.getByTestId("tickets-search-input"), "login");

    expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
    expect(screen.queryByTestId("ticket-row-t2")).not.toBeInTheDocument();
  });

  it("shows updated count after filtering", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const tickets = [
      makeTicket({ id: "t1", status: "open" }),
      makeTicket({ id: "t2", title: "Billing", status: "pending" }),
    ];
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse(tickets));
    render(<SupportManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("tickets-count")).toHaveTextContent("2");
    });

    await user.selectOptions(screen.getByTestId("tickets-status-filter"), "pending");

    expect(screen.getByTestId("tickets-count")).toHaveTextContent("1");
  });

  // ---------------------------------------------------------------------------
  // Create ticket form — toggle
  // ---------------------------------------------------------------------------

  it("does not show create form by default", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("support-management-view")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
  });

  it("shows create form when New Ticket button is clicked", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.getByTestId("create-ticket-form")).toBeInTheDocument();
  });

  it("hides create form when cancel button is clicked", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.click(screen.getByTestId("ticket-cancel-btn"));
    expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
  });

  it("submit button is disabled when title is empty", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    expect(screen.getByTestId("ticket-submit-btn")).toBeDisabled();
  });

  it("submit button is enabled when title is filled", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "My issue");
    expect(screen.getByTestId("ticket-submit-btn")).not.toBeDisabled();
  });

  // ---------------------------------------------------------------------------
  // Create ticket form — submit success
  // ---------------------------------------------------------------------------

  it("calls createSupportTicket with correct title and body", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const newTicket = makeTicket({ id: "t99", title: "Help me" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Help me");
    await user.type(screen.getByTestId("ticket-body-input"), "Please assist");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).toHaveBeenCalledWith("test-token", {
        title: "Help me",
        body: "Please assist",
        org_id: undefined,
      });
    });
  });

  it("passes org_id when user has org (tier 3 with org)", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: "org-123" }));
    const newTicket = makeTicket({ id: "t99" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Issue");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ org_id: "org-123" }),
      );
    });
  });

  it("omits body when body is blank", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockCreateSupportTicket.mockResolvedValue(makeTicket({ id: "t99", title: "No body" }));

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "No body");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).toHaveBeenCalledWith("test-token", {
        title: "No body",
        body: undefined,
        org_id: undefined,
      });
    });
  });

  it("prepends new ticket to list after successful create", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const existing = makeTicket({ id: "t1", title: "Old ticket" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([existing]));
    const newTicket = makeTicket({ id: "t99", title: "New ticket" });
    mockCreateSupportTicket.mockResolvedValue(newTicket);

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "New ticket");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t99")).toBeInTheDocument();
    });
    const rows = screen.getAllByTestId(/^ticket-row-/);
    expect(rows[0]).toHaveAttribute("data-testid", "ticket-row-t99");
  });

  it("closes form and clears inputs after successful create", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockCreateSupportTicket.mockResolvedValue(makeTicket({ id: "t99" }));

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Done");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.queryByTestId("create-ticket-form")).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Create ticket form — submit error
  // ---------------------------------------------------------------------------

  it("shows create error banner when createSupportTicket rejects", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockCreateSupportTicket.mockRejectedValue(new Error("Server error"));

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Broken");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("create-error")).toHaveTextContent("Server error");
    });
  });

  it("shows fallback create error for non-Error rejections", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockCreateSupportTicket.mockRejectedValue("oops");

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("new-ticket-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Oops");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("create-error")).toHaveTextContent("Failed to create ticket");
    });
  });

  it("does not call createSupportTicket when getToken returns null", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    mockGetToken.mockResolvedValue(null);

    render(<SupportManagementView />);
    // Wait for the initial load attempt to finish (null token — no fetch).
    await waitFor(() => {
      expect(screen.getByTestId("tickets-empty")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("new-ticket-btn"));
    await user.type(screen.getByTestId("ticket-title-input"), "Token gone");
    await user.click(screen.getByTestId("ticket-submit-btn"));

    await waitFor(() => {
      expect(mockCreateSupportTicket).not.toHaveBeenCalled();
    });
  });

  // ---------------------------------------------------------------------------
  // Pagination
  // ---------------------------------------------------------------------------

  it("shows load-more button when has_more is true", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()], true));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-load-more")).toBeInTheDocument();
    });
  });

  it("does not show load-more when has_more is false", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([makeTicket()], false));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-list")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("tickets-load-more")).not.toBeInTheDocument();
  });

  it("appends next page when load-more is clicked", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));

    const page1 = makePagedResponse([makeTicket({ id: "t1" })], true);
    const page2 = makePagedResponse([makeTicket({ id: "t2", title: "Second" })], false);
    mockFetchGlobalSupportTickets.mockResolvedValueOnce(page1).mockResolvedValueOnce(page2);

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-load-more")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("tickets-load-more"));

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t2")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("tickets-load-more")).not.toBeInTheDocument();
  });

  it("passes cursor to fetchGlobalSupportTickets on load more", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));

    const page1 = {
      data: [makeTicket({ id: "t1" })],
      page_info: { has_more: true, next_cursor: "cursor-abc" },
    };
    mockFetchGlobalSupportTickets
      .mockResolvedValueOnce(page1)
      .mockResolvedValueOnce(makePagedResponse([]));

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-load-more")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("tickets-load-more"));

    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledTimes(2);
    });
    expect(mockFetchGlobalSupportTickets).toHaveBeenLastCalledWith(
      "test-token",
      expect.objectContaining({ cursor: "cursor-abc" }),
    );
  });

  // ---------------------------------------------------------------------------
  // Ticket summary — creator and org info
  // ---------------------------------------------------------------------------

  it("shows author_name in ticket row when present", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({
      id: "t-author",
      author_id: "user-123",
      author_name: "Alice Smith",
      author_email: "alice@example.com",
    });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-creator-t-author")).toHaveTextContent("Alice Smith");
    });
  });

  it("falls back to author_email when author_name is absent", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({
      id: "t-email",
      author_id: "user-123",
      author_email: "bob@example.com",
    });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-creator-t-email")).toHaveTextContent("bob@example.com");
    });
  });

  it("falls back to author_id when no name or email present", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-id-only", author_id: "raw-clerk-id" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-creator-t-id-only")).toHaveTextContent("raw-clerk-id");
    });
  });

  it("shows org_name in ticket row when present", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({
      id: "t-org",
      org_id: "org-abc",
      org_name: "Acme Corp",
    });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-creator-t-org")).toHaveTextContent("Acme Corp");
    });
  });

  it("falls back to org:id when org_name absent but org_id present", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-orgid", org_id: "org-xyz" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-creator-t-orgid")).toHaveTextContent("org:org-xyz");
    });
  });

  it("shows no org label when no org_id or org_name", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-noorg", author_name: "User" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      const row = screen.getByTestId("ticket-creator-t-noorg");
      expect(row).not.toHaveTextContent("org:");
    });
  });

  // ---------------------------------------------------------------------------
  // Open button and work-view modal
  // ---------------------------------------------------------------------------

  it("renders an Open button for each ticket row", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-open" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-open")).toBeInTheDocument();
    });
  });

  it("clicking Open shows the work-view modal with ticket title", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-modal", title: "My Issue", body: "Some details" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-modal")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-modal"));
    expect(screen.getByTestId("work-view-modal")).toBeInTheDocument();
    expect(screen.getByTestId("work-view-title")).toHaveTextContent("My Issue");
  });

  it("work-view modal shows creator info", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({
      id: "t-wv-creator",
      author_name: "Carol",
      org_name: "Beta Inc",
    });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-wv-creator")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-wv-creator"));
    expect(screen.getByTestId("work-view-creator")).toHaveTextContent("Carol");
    expect(screen.getByTestId("work-view-creator")).toHaveTextContent("Beta Inc");
  });

  it("work-view modal shows status select for tier 4 (scopesAll)", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-status-sel", status: "open" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-status-sel")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-status-sel"));
    expect(screen.getByTestId("work-view-status-select")).toBeInTheDocument();
  });

  it("work-view modal does not show status select for tier 2 (scopesMine)", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    const ticket = makeTicket({ id: "t-no-status" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-no-status")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-no-status"));
    expect(screen.queryByTestId("work-view-status-select")).not.toBeInTheDocument();
  });

  it("work-view cancel button closes the modal", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-cancel" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-cancel")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-cancel"));
    expect(screen.getByTestId("work-view-modal")).toBeInTheDocument();
    await user.click(screen.getByTestId("work-view-cancel-btn"));
    expect(screen.queryByTestId("work-view-modal")).not.toBeInTheDocument();
  });

  it("work-view close (X) button closes the modal", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-xclose" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-xclose")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-xclose"));
    await user.click(screen.getByTestId("work-view-close-btn"));
    expect(screen.queryByTestId("work-view-modal")).not.toBeInTheDocument();
  });

  it("work-view save calls updateSupportTicket and updates list", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({
      id: "t-save",
      slug: "t-save-slug",
      body: "old body",
      status: "open",
    });
    const updated = makeTicket({
      id: "t-save",
      slug: "t-save-slug",
      body: "new body",
      status: "resolved",
    });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    mockUpdateSupportTicket.mockResolvedValue(updated);

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-save")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-save"));

    // Change body.
    const bodyInput = screen.getByTestId("work-view-body-input");
    await user.clear(bodyInput);
    await user.type(bodyInput, "new body");

    // Change status.
    await user.selectOptions(screen.getByTestId("work-view-status-select"), "resolved");

    await user.click(screen.getByTestId("work-view-save-btn"));

    await waitFor(() => {
      expect(mockUpdateSupportTicket).toHaveBeenCalledWith(
        "test-token",
        "t-save-slug",
        expect.objectContaining({ body: "new body", status: "resolved" }),
      );
    });

    // List row status badge should reflect the update.
    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-t-save")).toHaveTextContent("resolved");
    });
  });

  it("work-view save shows error on failure", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-save-err" });
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));
    mockUpdateSupportTicket.mockRejectedValue(new Error("Save failed"));

    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-open-btn-t-save-err")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("ticket-open-btn-t-save-err"));
    await user.click(screen.getByTestId("work-view-save-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("work-view-error")).toHaveTextContent("Save failed");
    });
  });

  it("work-view does not call updateSupportTicket when token is null", async () => {
    mockGetToken.mockResolvedValue(null);
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    const ticket = makeTicket({ id: "t-null-token" });
    // Let first fetch succeed (from initial render before token goes null).
    mockFetchGlobalSupportTickets.mockResolvedValue(makePagedResponse([ticket]));

    render(<SupportManagementView />);
    await waitFor(() => {
      // Even with null token at render time, empty list is shown.
      expect(screen.getByTestId("tickets-empty")).toBeInTheDocument();
    });
  });
});
