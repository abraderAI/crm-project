import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { Thread } from "@/lib/api-types";

// Mock Next.js Link to a plain anchor for test assertions.
vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode;
    href: string;
    className?: string;
    "data-testid"?: string;
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

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

// Mock fetchGlobalLeads.
const mockFetchGlobalLeads = vi.fn();
vi.mock("@/lib/global-api", () => ({
  fetchGlobalLeads: (...args: unknown[]) => mockFetchGlobalLeads(...args),
}));

import { LeadsManagementView } from "./leads-management-view";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Default no-loading tier state factory. */
function makeTier(
  overrides: Partial<{
    tier: number;
    deftDepartment: string | null;
    isLoading: boolean;
  }> = {},
) {
  return {
    tier: 1,
    deftDepartment: null,
    subType: null,
    orgId: null,
    isLoading: false,
    refresh: vi.fn(),
    ...overrides,
  };
}

/** Minimal Thread fixture factory. */
function makeThread(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "t1",
    board_id: "b1",
    title: "Acme Corp Deal",
    slug: "acme-corp-deal",
    metadata: JSON.stringify({ company: "Acme Corp", assigned_to: "alice", score: 80 }),
    author_id: "u1",
    is_pinned: false,
    is_locked: false,
    is_hidden: false,
    vote_score: 0,
    stage: "new_lead",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

/** Paginated response wrapper. */
function makePagedResponse(threads: Thread[], hasMore = false) {
  return {
    data: threads,
    page_info: { has_more: hasMore, next_cursor: hasMore ? "cursor-next" : undefined },
  };
}

describe("LeadsManagementView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([]));
  });

  // ---------------------------------------------------------------------------
  // Tier-resolving loading state
  // ---------------------------------------------------------------------------

  it("shows tier loading state while tier is resolving", () => {
    mockUseTier.mockReturnValue(makeTier({ isLoading: true }));
    render(<LeadsManagementView />);
    expect(screen.getByTestId("leads-loading-tier")).toBeInTheDocument();
  });

  it("does not show access denied or view during tier loading", () => {
    mockUseTier.mockReturnValue(makeTier({ isLoading: true }));
    render(<LeadsManagementView />);
    expect(screen.queryByTestId("leads-access-denied")).not.toBeInTheDocument();
    expect(screen.queryByTestId("leads-management-view")).not.toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // Access denied — tiers without access
  // ---------------------------------------------------------------------------

  it("shows access denied for tier 1 (anonymous)", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 1 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
  });

  it("shows access denied for tier 2 (registered developer)", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
  });

  it("shows access denied for tier 3 (paying customer)", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 3 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
  });

  it("shows access denied for tier 4 with support department", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "support" }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
  });

  it("shows access denied for tier 4 with finance department", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "finance" }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
  });

  it("shows access denied for tier 4 with no department", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: null }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
  });

  it("access denied message mentions DEFT sales staff", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 1 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toHaveTextContent("DEFT sales staff");
    });
  });

  it("does not call fetchGlobalLeads for unauthorized tiers", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 2 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-access-denied")).toBeInTheDocument();
    });
    expect(mockFetchGlobalLeads).not.toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Access granted — tier 4 sales rep (my leads)
  // ---------------------------------------------------------------------------

  it("renders leads view for tier 4 sales rep", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "sales" }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-management-view")).toBeInTheDocument();
    });
  });

  it("shows 'My Leads' heading for tier 4 sales rep", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "sales" }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByText("My Leads")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalLeads with mine=true for tier 4 sales rep", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "sales" }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalLeads).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ mine: true }),
      );
    });
  });

  it("does not show assignee filter for tier 4 sales rep", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "sales" }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-management-view")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("leads-assignee-filter")).not.toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // Access granted — tier 5 (org manager / all leads)
  // ---------------------------------------------------------------------------

  it("renders leads view for tier 5 (org manager)", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-management-view")).toBeInTheDocument();
    });
  });

  it("shows 'All Leads' heading for tier 5", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByText("All Leads")).toBeInTheDocument();
    });
  });

  it("calls fetchGlobalLeads without mine for tier 5", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(mockFetchGlobalLeads).toHaveBeenCalledWith(
        "test-token",
        expect.objectContaining({ mine: false }),
      );
    });
  });

  it("shows assignee filter dropdown for tier 5", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-assignee-filter")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Access granted — tier 6 (platform admin / all leads)
  // ---------------------------------------------------------------------------

  it("renders leads view for tier 6 (platform admin)", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-management-view")).toBeInTheDocument();
    });
  });

  it("shows 'All Leads' heading for tier 6", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByText("All Leads")).toBeInTheDocument();
    });
  });

  it("shows assignee filter dropdown for tier 6", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 6 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-assignee-filter")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Lead list rendering
  // ---------------------------------------------------------------------------

  it("shows loading skeleton while fetching leads", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    // Return a never-resolving promise to hold the loading state.
    mockFetchGlobalLeads.mockReturnValue(new Promise(() => {}));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-list-loading")).toBeInTheDocument();
    });
  });

  it("shows empty state when no leads are returned", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([]));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-empty")).toBeInTheDocument();
    });
  });

  it("renders lead rows for each returned thread", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1" }),
      makeThread({ id: "t2", slug: "deal-2", title: "Deal 2" }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("lead-row-t2")).toBeInTheDocument();
    });
  });

  it("renders lead title in each row", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread({ id: "t1" })]));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toHaveTextContent("Acme Corp Deal");
    });
  });

  it("renders company name from metadata", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread({ id: "t1" })]));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("lead-company-t1")).toHaveTextContent("Acme Corp");
    });
  });

  it("renders stage badge for each lead", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(
      makePagedResponse([makeThread({ id: "t1", stage: "qualified" })]),
    );
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("lead-stage-t1")).toHaveTextContent("Qualified");
    });
  });

  it("renders score for leads that have one", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread({ id: "t1" })]));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("lead-score-t1")).toHaveTextContent("80");
    });
  });

  it("links each lead row to the global lead detail route", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(
      makePagedResponse([makeThread({ id: "t1", slug: "acme-corp-deal" })]),
    );
    render(<LeadsManagementView />);
    await waitFor(() => {
      const link = screen.getByTestId("lead-row-t1");
      expect(link).toHaveAttribute("href", "/crm/leads/global/acme-corp-deal");
    });
  });

  it("shows assignee for tier 5+ leads", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread({ id: "t1" })]));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("lead-assignee-t1")).toHaveTextContent("alice");
    });
  });

  it("does not show assignee column for tier 4 sales rep", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 4, deftDepartment: "sales" }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread({ id: "t1" })]));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-management-view")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("lead-assignee-t1")).not.toBeInTheDocument();
  });

  it("shows count of visible leads", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1" }),
      makeThread({ id: "t2", slug: "t2", title: "Deal 2" }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-count")).toHaveTextContent("2");
    });
  });

  // ---------------------------------------------------------------------------
  // Error handling
  // ---------------------------------------------------------------------------

  it("shows error banner when fetchGlobalLeads rejects", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockRejectedValue(new Error("Network failure"));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-error")).toHaveTextContent("Network failure");
    });
  });

  it("shows fallback error message for non-Error rejections", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockRejectedValue("oops");
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-error")).toHaveTextContent("Failed to load leads");
    });
  });

  it("does not fetch leads when getToken returns null", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockGetToken.mockResolvedValue(null);
    render(<LeadsManagementView />);
    await waitFor(() => {
      // Should remain in loading or empty state, not error.
      expect(screen.queryByTestId("leads-error")).not.toBeInTheDocument();
    });
    expect(mockFetchGlobalLeads).not.toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Filters
  // ---------------------------------------------------------------------------

  it("renders search input", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-search-input")).toBeInTheDocument();
    });
  });

  it("renders stage filter dropdown", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-stage-filter")).toBeInTheDocument();
    });
  });

  it("filters leads by stage selection", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1", stage: "new_lead" }),
      makeThread({ id: "t2", slug: "t2", title: "Deal 2", stage: "qualified" }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("lead-row-t2")).toBeInTheDocument();
    });

    // Select "qualified" stage.
    await user.selectOptions(screen.getByTestId("leads-stage-filter"), "qualified");

    expect(screen.queryByTestId("lead-row-t1")).not.toBeInTheDocument();
    expect(screen.getByTestId("lead-row-t2")).toBeInTheDocument();
  });

  it("filters leads by search term matching title", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1", title: "Acme Corp Deal", slug: "acme" }),
      makeThread({
        id: "t2",
        slug: "t2",
        title: "Widget Inc Deal",
        metadata: JSON.stringify({ company: "Widget Inc" }),
      }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
    });

    await user.type(screen.getByTestId("leads-search-input"), "acme");

    expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
    expect(screen.queryByTestId("lead-row-t2")).not.toBeInTheDocument();
  });

  it("filters leads by search term matching company", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1", metadata: JSON.stringify({ company: "Acme Corp" }) }),
      makeThread({
        id: "t2",
        slug: "t2",
        title: "Other Deal",
        metadata: JSON.stringify({ company: "Widget Inc" }),
      }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
    });

    await user.type(screen.getByTestId("leads-search-input"), "widget");

    expect(screen.queryByTestId("lead-row-t1")).not.toBeInTheDocument();
    expect(screen.getByTestId("lead-row-t2")).toBeInTheDocument();
  });

  it("filters leads by assignee for tier 5+", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1", metadata: JSON.stringify({ company: "Acme", assigned_to: "alice" }) }),
      makeThread({
        id: "t2",
        slug: "t2",
        title: "Deal 2",
        metadata: JSON.stringify({ company: "Beta", assigned_to: "bob" }),
      }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("lead-row-t2")).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId("leads-assignee-filter"), "alice");

    expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
    expect(screen.queryByTestId("lead-row-t2")).not.toBeInTheDocument();
  });

  it("shows updated count after filtering", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    const threads = [
      makeThread({ id: "t1", stage: "new_lead" }),
      makeThread({ id: "t2", slug: "t2", title: "Deal 2", stage: "qualified" }),
    ];
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse(threads));
    render(<LeadsManagementView />);

    await waitFor(() => {
      expect(screen.getByTestId("leads-count")).toHaveTextContent("2");
    });

    await user.selectOptions(screen.getByTestId("leads-stage-filter"), "qualified");

    expect(screen.getByTestId("leads-count")).toHaveTextContent("1");
  });

  // ---------------------------------------------------------------------------
  // Pagination
  // ---------------------------------------------------------------------------

  it("shows load-more button when has_more is true", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread()], true));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-load-more")).toBeInTheDocument();
    });
  });

  it("does not show load-more button when has_more is false", async () => {
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));
    mockFetchGlobalLeads.mockResolvedValue(makePagedResponse([makeThread()], false));
    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-list")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("leads-load-more")).not.toBeInTheDocument();
  });

  it("appends next page of leads when load-more is clicked", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));

    const page1 = makePagedResponse([makeThread({ id: "t1" })], true);
    const page2 = makePagedResponse([makeThread({ id: "t2", slug: "t2", title: "Deal 2" })], false);
    mockFetchGlobalLeads.mockResolvedValueOnce(page1).mockResolvedValueOnce(page2);

    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-load-more")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("leads-load-more"));

    await waitFor(() => {
      expect(screen.getByTestId("lead-row-t1")).toBeInTheDocument();
      expect(screen.getByTestId("lead-row-t2")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("leads-load-more")).not.toBeInTheDocument();
  });

  it("passes cursor to fetchGlobalLeads on load more", async () => {
    const user = userEvent.setup();
    mockUseTier.mockReturnValue(makeTier({ tier: 5 }));

    const page1 = {
      data: [makeThread({ id: "t1" })],
      page_info: { has_more: true, next_cursor: "cursor-abc" },
    };
    mockFetchGlobalLeads.mockResolvedValueOnce(page1).mockResolvedValueOnce(makePagedResponse([]));

    render(<LeadsManagementView />);
    await waitFor(() => {
      expect(screen.getByTestId("leads-load-more")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("leads-load-more"));

    await waitFor(() => {
      expect(mockFetchGlobalLeads).toHaveBeenCalledTimes(2);
    });
    expect(mockFetchGlobalLeads).toHaveBeenLastCalledWith(
      "test-token",
      expect.objectContaining({ cursor: "cursor-abc" }),
    );
  });
});
