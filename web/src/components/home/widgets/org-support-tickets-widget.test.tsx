import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OrgSupportTicketsWidget } from "./org-support-tickets-widget";

const mockFetchOrgSupportTickets = vi.fn();
vi.mock("@/lib/org-api", () => ({
  fetchOrgSupportTickets: (...args: unknown[]) => mockFetchOrgSupportTickets(...args),
}));

const MOCK_TICKETS = [
  { id: "t1", title: "Server downtime", status: "open" },
  { id: "t2", title: "API rate limiting", status: "pending" },
  { id: "t3", title: "Bug fix deployed", status: "resolved" },
];

describe("OrgSupportTicketsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchOrgSupportTickets.mockReturnValue(new Promise(() => {}));
    render(<OrgSupportTicketsWidget token="token" orgId="org-1" />);
    expect(screen.getByTestId("org-support-tickets-loading")).toBeInTheDocument();
  });

  it("renders ticket list on successful fetch", async () => {
    mockFetchOrgSupportTickets.mockResolvedValue({
      data: MOCK_TICKETS,
      page_info: { has_more: false },
    });

    render(<OrgSupportTicketsWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-tickets-list")).toBeInTheDocument();
    });

    expect(screen.getByText("Server downtime")).toBeInTheDocument();
    expect(screen.getByText("API rate limiting")).toBeInTheDocument();
    expect(screen.getByText("Bug fix deployed")).toBeInTheDocument();
  });

  it("displays status badges for each ticket", async () => {
    mockFetchOrgSupportTickets.mockResolvedValue({
      data: MOCK_TICKETS,
      page_info: { has_more: false },
    });

    render(<OrgSupportTicketsWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-tickets-list")).toBeInTheDocument();
    });

    expect(screen.getByTestId("org-ticket-status-t1")).toHaveTextContent("open");
    expect(screen.getByTestId("org-ticket-status-t2")).toHaveTextContent("pending");
    expect(screen.getByTestId("org-ticket-status-t3")).toHaveTextContent("resolved");
  });

  it("shows empty state when no tickets", async () => {
    mockFetchOrgSupportTickets.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<OrgSupportTicketsWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-tickets-empty")).toBeInTheDocument();
    });
  });

  it("shows error state on fetch failure", async () => {
    mockFetchOrgSupportTickets.mockRejectedValue(new Error("Network error"));

    render(<OrgSupportTicketsWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-tickets-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load support tickets.")).toBeInTheDocument();
  });

  it("passes token, orgId and limit to API", async () => {
    mockFetchOrgSupportTickets.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<OrgSupportTicketsWidget token="auth-token" orgId="my-org" />);

    await waitFor(() => {
      expect(mockFetchOrgSupportTickets).toHaveBeenCalledWith("auth-token", "my-org", {
        limit: 5,
      });
    });
  });
});
