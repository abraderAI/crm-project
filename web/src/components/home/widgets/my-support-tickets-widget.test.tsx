import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { MySupportTicketsWidget } from "./my-support-tickets-widget";

const mockFetchUserSupportTickets = vi.fn();
vi.mock("@/lib/global-api", () => ({
  fetchUserSupportTickets: (...args: unknown[]) => mockFetchUserSupportTickets(...args),
}));

const MOCK_TICKETS = [
  { id: "s1", title: "Cannot login", status: "open" },
  { id: "s2", title: "Billing question", status: "pending" },
  { id: "s3", title: "API issue resolved", status: "resolved" },
];

describe("MySupportTicketsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchUserSupportTickets.mockReturnValue(new Promise(() => {}));
    render(<MySupportTicketsWidget token="token" />);
    expect(screen.getByTestId("my-support-tickets-loading")).toBeInTheDocument();
  });

  it("renders ticket list on successful fetch", async () => {
    mockFetchUserSupportTickets.mockResolvedValue({
      data: MOCK_TICKETS,
      page_info: { has_more: false },
    });

    render(<MySupportTicketsWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-support-tickets-list")).toBeInTheDocument();
    });

    expect(screen.getByText("Cannot login")).toBeInTheDocument();
    expect(screen.getByText("Billing question")).toBeInTheDocument();
    expect(screen.getByText("API issue resolved")).toBeInTheDocument();
  });

  it("displays status badges for each ticket", async () => {
    mockFetchUserSupportTickets.mockResolvedValue({
      data: MOCK_TICKETS,
      page_info: { has_more: false },
    });

    render(<MySupportTicketsWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-support-tickets-list")).toBeInTheDocument();
    });

    expect(screen.getByTestId("ticket-status-s1")).toHaveTextContent("open");
    expect(screen.getByTestId("ticket-status-s2")).toHaveTextContent("pending");
    expect(screen.getByTestId("ticket-status-s3")).toHaveTextContent("resolved");
  });

  it("defaults status to open when not set", async () => {
    mockFetchUserSupportTickets.mockResolvedValue({
      data: [{ id: "s4", title: "No status" }],
      page_info: { has_more: false },
    });

    render(<MySupportTicketsWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-s4")).toHaveTextContent("open");
    });
  });

  it("shows empty state when no tickets", async () => {
    mockFetchUserSupportTickets.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<MySupportTicketsWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-support-tickets-empty")).toBeInTheDocument();
    });
  });

  it("shows error state on fetch failure", async () => {
    mockFetchUserSupportTickets.mockRejectedValue(new Error("Network error"));

    render(<MySupportTicketsWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-support-tickets-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load your support tickets.")).toBeInTheDocument();
  });

  it("passes token and limit to API", async () => {
    mockFetchUserSupportTickets.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<MySupportTicketsWidget token="auth-token" />);

    await waitFor(() => {
      expect(mockFetchUserSupportTickets).toHaveBeenCalledWith("auth-token", { limit: 5 });
    });
  });
});
