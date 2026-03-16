import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { TicketStatsWidget } from "./ticket-stats-widget";

const mockFetchTicketStats = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchTicketStats: (...args: unknown[]) => mockFetchTicketStats(...args),
}));

describe("TicketStatsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchTicketStats.mockReturnValue(new Promise(() => {}));
    render(<TicketStatsWidget token="tok" />);
    expect(screen.getByTestId("ticket-stats-loading")).toBeInTheDocument();
  });

  it("renders stats after loading", async () => {
    mockFetchTicketStats.mockResolvedValue({
      open: 15,
      pending: 8,
      resolved: 42,
      avg_response_time: "1h 30m",
    });

    render(<TicketStatsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-stats-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("stat-open")).toHaveTextContent("15");
    expect(screen.getByTestId("stat-pending")).toHaveTextContent("8");
    expect(screen.getByTestId("stat-resolved")).toHaveTextContent("42");
    expect(screen.getByTestId("stat-avg-response")).toHaveTextContent("1h 30m");
  });

  it("shows error state on failure", async () => {
    mockFetchTicketStats.mockRejectedValue(new Error("fail"));

    render(<TicketStatsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("ticket-stats-error")).toBeInTheDocument();
    });
  });

  it("passes token to API", async () => {
    mockFetchTicketStats.mockResolvedValue({
      open: 0,
      pending: 0,
      resolved: 0,
      avg_response_time: "N/A",
    });

    render(<TicketStatsWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchTicketStats).toHaveBeenCalledWith("my-token");
    });
  });

  it("displays zero counts correctly", async () => {
    mockFetchTicketStats.mockResolvedValue({
      open: 0,
      pending: 0,
      resolved: 0,
      avg_response_time: "N/A",
    });

    render(<TicketStatsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("stat-open")).toHaveTextContent("0");
    });
  });
});
