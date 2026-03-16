import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { TicketQueueWidget } from "./ticket-queue-widget";

const mockFetchOpenTickets = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchOpenTickets: (...args: unknown[]) => mockFetchOpenTickets(...args),
}));

const sampleTickets = [
  { id: "t1", title: "Login issue", status: "open", org_name: "Acme", created_at: "2026-01-01" },
  { id: "t2", title: "API error", status: "pending", org_name: "Beta", created_at: "2026-01-02" },
];

describe("TicketQueueWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchOpenTickets.mockReturnValue(new Promise(() => {}));
    render(<TicketQueueWidget token="tok" />);
    expect(screen.getByTestId("ticket-queue-loading")).toBeInTheDocument();
  });

  it("renders tickets after loading", async () => {
    mockFetchOpenTickets.mockResolvedValue(sampleTickets);

    render(<TicketQueueWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-queue-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("ticket-t1")).toBeInTheDocument();
    expect(screen.getByTestId("ticket-t2")).toBeInTheDocument();
    expect(screen.getByText("Login issue")).toBeInTheDocument();
    expect(screen.getByText("Acme")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    mockFetchOpenTickets.mockResolvedValue(sampleTickets);

    render(<TicketQueueWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-status-t1")).toHaveTextContent("open");
      expect(screen.getByTestId("ticket-status-t2")).toHaveTextContent("pending");
    });
  });

  it("shows error state on failure", async () => {
    mockFetchOpenTickets.mockRejectedValue(new Error("fail"));

    render(<TicketQueueWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-queue-error")).toBeInTheDocument();
    });
  });

  it("shows empty state when no tickets", async () => {
    mockFetchOpenTickets.mockResolvedValue([]);

    render(<TicketQueueWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-queue-empty")).toBeInTheDocument();
    });
  });

  it("passes token to API", async () => {
    mockFetchOpenTickets.mockResolvedValue([]);

    render(<TicketQueueWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchOpenTickets).toHaveBeenCalledWith("my-token");
    });
  });
});
