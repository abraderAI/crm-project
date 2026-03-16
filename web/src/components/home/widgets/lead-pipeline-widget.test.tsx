import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { LeadPipelineWidget } from "./lead-pipeline-widget";

const mockFetchLeadsByStatus = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchLeadsByStatus: (...args: unknown[]) => mockFetchLeadsByStatus(...args),
}));

describe("LeadPipelineWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchLeadsByStatus.mockReturnValue(new Promise(() => {}));
    render(<LeadPipelineWidget token="tok" />);
    expect(screen.getByTestId("lead-pipeline-loading")).toBeInTheDocument();
  });

  it("renders pipeline stages with counts after loading", async () => {
    mockFetchLeadsByStatus.mockResolvedValue({
      new_lead: 10,
      contacted: 5,
      qualified: 3,
      proposal: 2,
      negotiation: 1,
      closed_won: 4,
      closed_lost: 1,
      nurturing: 2,
    });

    render(<LeadPipelineWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-pipeline-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("lead-pipeline-total")).toHaveTextContent("28");
    expect(screen.getByTestId("stage-new_lead")).toBeInTheDocument();
    expect(screen.getByTestId("stage-closed_won")).toBeInTheDocument();
  });

  it("shows error state on API failure", async () => {
    mockFetchLeadsByStatus.mockRejectedValue(new Error("Network error"));

    render(<LeadPipelineWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-pipeline-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load lead pipeline data")).toBeInTheDocument();
  });

  it("passes token to API function", async () => {
    mockFetchLeadsByStatus.mockResolvedValue({
      new_lead: 0,
      contacted: 0,
      qualified: 0,
      proposal: 0,
      negotiation: 0,
      closed_won: 0,
      closed_lost: 0,
      nurturing: 0,
    });

    render(<LeadPipelineWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchLeadsByStatus).toHaveBeenCalledWith("my-token");
    });
  });

  it("renders all 8 pipeline stages", async () => {
    mockFetchLeadsByStatus.mockResolvedValue({
      new_lead: 1,
      contacted: 1,
      qualified: 1,
      proposal: 1,
      negotiation: 1,
      closed_won: 1,
      closed_lost: 1,
      nurturing: 1,
    });

    render(<LeadPipelineWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("lead-pipeline-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("stage-new_lead")).toBeInTheDocument();
    expect(screen.getByTestId("stage-contacted")).toBeInTheDocument();
    expect(screen.getByTestId("stage-qualified")).toBeInTheDocument();
    expect(screen.getByTestId("stage-proposal")).toBeInTheDocument();
    expect(screen.getByTestId("stage-negotiation")).toBeInTheDocument();
    expect(screen.getByTestId("stage-closed_won")).toBeInTheDocument();
    expect(screen.getByTestId("stage-closed_lost")).toBeInTheDocument();
    expect(screen.getByTestId("stage-nurturing")).toBeInTheDocument();
  });

  it("displays stage labels correctly", async () => {
    mockFetchLeadsByStatus.mockResolvedValue({
      new_lead: 5,
      contacted: 0,
      qualified: 0,
      proposal: 0,
      negotiation: 0,
      closed_won: 0,
      closed_lost: 0,
      nurturing: 0,
    });

    render(<LeadPipelineWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByText("New Lead")).toBeInTheDocument();
    });

    expect(screen.getByText("Closed Won")).toBeInTheDocument();
  });
});
