import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

import type { ThreadWithAuthor } from "@/lib/api-types";
import { SupportManagementView } from "./support-management-view";

const mockGetToken = vi.fn();
const mockUseTier = vi.fn();
const mockFetchGlobalSupportTickets = vi.fn();

vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

vi.mock("@/hooks/use-tier", () => ({
  useTier: () => mockUseTier(),
}));

vi.mock("@/lib/global-api", () => ({
  fetchGlobalSupportTickets: (...args: unknown[]) => mockFetchGlobalSupportTickets(...args),
}));

function makeTier(
  overrides: Partial<{
    tier: number;
    subType: string | null;
    orgId: string | null;
    isLoading: boolean;
  }> = {},
) {
  return {
    tier: 2,
    subType: null,
    orgId: null,
    isLoading: false,
    ...overrides,
  };
}

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

function paged(tickets: ThreadWithAuthor[]) {
  return {
    data: tickets,
    page_info: { has_more: false, next_cursor: undefined },
  };
}

describe("SupportManagementView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("tok");
    mockUseTier.mockReturnValue(makeTier());
    mockFetchGlobalSupportTickets.mockResolvedValue(paged([]));
  });

  it("shows tier loading state while tier is resolving", () => {
    mockUseTier.mockReturnValue(makeTier({ isLoading: true }));
    render(<SupportManagementView />);
    expect(screen.getByTestId("support-loading-tier")).toBeInTheDocument();
  });

  it("shows neutral DEFT.support placeholder for non-access tiers", () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 1 }));
    render(<SupportManagementView />);
    expect(screen.getByTestId("support-access-denied")).toHaveTextContent("DEFT.support");
  });

  it("fetches mine=true for tier 2 scope", async () => {
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "tok",
        expect.objectContaining({ mine: true }),
      );
    });
  });

  it("fetches org-scoped tickets when tier 3 has org", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3, orgId: "org-123" }));
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalSupportTickets).toHaveBeenCalledWith(
        "tok",
        expect.objectContaining({ org_id: "org-123" }),
      );
    });
  });
  it("renders sort filter and applies oldest-first ordering", async () => {
    const user = userEvent.setup();
    mockFetchGlobalSupportTickets.mockResolvedValue(
      paged([
        makeTicket({ id: "t-new", title: "New", created_at: "2026-03-21T00:00:00Z" }),
        makeTicket({ id: "t-old", title: "Old", created_at: "2026-01-01T00:00:00Z" }),
      ]),
    );
    render(<SupportManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t-new")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t-old")).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId("tickets-sort-filter"), "oldest");
    const rows = screen.getAllByTestId(/^ticket-row-/);
    expect(rows[0]).toHaveAttribute("data-testid", "ticket-row-t-old");
    expect(rows[1]).toHaveAttribute("data-testid", "ticket-row-t-new");
  });

  it("applies recently-updated ordering", async () => {
    const user = userEvent.setup();
    mockFetchGlobalSupportTickets.mockResolvedValue(
      paged([
        makeTicket({ id: "t-a", title: "A", updated_at: "2026-01-01T00:00:00Z" }),
        makeTicket({ id: "t-b", title: "B", updated_at: "2026-03-01T00:00:00Z" }),
      ]),
    );
    render(<SupportManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t-a")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t-b")).toBeInTheDocument();
    });
    await user.selectOptions(screen.getByTestId("tickets-sort-filter"), "updated");
    const rows = screen.getAllByTestId(/^ticket-row-/);
    expect(rows[0]).toHaveAttribute("data-testid", "ticket-row-t-b");
  });

  it("filters ticket rows by status selection", async () => {
    const user = userEvent.setup();
    mockFetchGlobalSupportTickets.mockResolvedValue(
      paged([
        makeTicket({ id: "t-open", status: "open" }),
        makeTicket({ id: "t-closed", status: "closed" }),
      ]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t-open")).toBeInTheDocument();
      expect(screen.getByTestId("ticket-row-t-closed")).toBeInTheDocument();
    });
    await user.selectOptions(screen.getByTestId("tickets-status-filter"), "closed");
    expect(screen.queryByTestId("ticket-row-t-open")).not.toBeInTheDocument();
    expect(screen.getByTestId("ticket-row-t-closed")).toBeInTheDocument();
  });

  it("filters ticket rows by search text", async () => {
    const user = userEvent.setup();
    mockFetchGlobalSupportTickets.mockResolvedValue(
      paged([
        makeTicket({ id: "t-a", title: "Login issue" }),
        makeTicket({ id: "t-b", title: "Billing issue" }),
      ]),
    );
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t-a")).toBeInTheDocument();
    });
    await user.type(screen.getByTestId("tickets-search-input"), "login");
    expect(screen.getByTestId("ticket-row-t-a")).toBeInTheDocument();
    expect(screen.queryByTestId("ticket-row-t-b")).not.toBeInTheDocument();
  });

  it("shows error banner on fetch failure", async () => {
    mockFetchGlobalSupportTickets.mockRejectedValueOnce(new Error("network failure"));
    render(<SupportManagementView />);
    expect(await screen.findByTestId("support-error")).toHaveTextContent("network failure");
  });

  it("supports load-more pagination flow", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 4 }));
    mockFetchGlobalSupportTickets
      .mockResolvedValueOnce({
        data: [makeTicket({ id: "t-1", title: "First" })],
        page_info: { has_more: true, next_cursor: "cursor-2" },
      })
      .mockResolvedValueOnce({
        data: [makeTicket({ id: "t-2", title: "Second" })],
        page_info: { has_more: false, next_cursor: undefined },
      });
    render(<SupportManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("tickets-load-more")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("tickets-load-more"));
    await waitFor(() => {
      expect(screen.getByTestId("ticket-row-t-2")).toBeInTheDocument();
    });
  });
});
