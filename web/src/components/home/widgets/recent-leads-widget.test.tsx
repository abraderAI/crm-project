import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { RecentLeadsWidget } from "./recent-leads-widget";

const mockFetchRecentLeads = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchRecentLeads: (...args: unknown[]) => mockFetchRecentLeads(...args),
}));

const sampleLeads = [
  { id: "l1", title: "Acme Corp", source: "chatbot", status: "new_lead", created_at: "2026-01-01" },
  { id: "l2", title: "Beta Inc", source: "website", status: "contacted", created_at: "2026-01-02" },
];

describe("RecentLeadsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchRecentLeads.mockReturnValue(new Promise(() => {}));
    render(<RecentLeadsWidget token="tok" />);
    expect(screen.getByTestId("recent-leads-loading")).toBeInTheDocument();
  });

  it("renders leads after loading", async () => {
    mockFetchRecentLeads.mockResolvedValue(sampleLeads);

    render(<RecentLeadsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("recent-leads-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("lead-l1")).toBeInTheDocument();
    expect(screen.getByTestId("lead-l2")).toBeInTheDocument();
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByText("chatbot")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    mockFetchRecentLeads.mockResolvedValue(sampleLeads);

    render(<RecentLeadsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-status-l1")).toHaveTextContent("new_lead");
    });
  });

  it("shows error state on failure", async () => {
    mockFetchRecentLeads.mockRejectedValue(new Error("fail"));

    render(<RecentLeadsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("recent-leads-error")).toBeInTheDocument();
    });
  });

  it("shows empty state when no leads", async () => {
    mockFetchRecentLeads.mockResolvedValue([]);

    render(<RecentLeadsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("recent-leads-empty")).toBeInTheDocument();
    });
  });

  it("passes token and limit to API", async () => {
    mockFetchRecentLeads.mockResolvedValue([]);

    render(<RecentLeadsWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchRecentLeads).toHaveBeenCalledWith("my-token", 10);
    });
  });
});
