import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { PipelineStats } from "@/lib/crm-types";
import { PipelineDashboard } from "./pipeline-stats";

const sampleStats: PipelineStats = {
  total_leads: 25,
  total_value: 500000,
  stage_counts: {
    new_lead: 8,
    contacted: 5,
    qualified: 4,
    proposal: 3,
    negotiation: 2,
    closed_won: 2,
    closed_lost: 1,
  },
  conversion_rate: 66.7,
  average_value: 20000,
};

const emptyStats: PipelineStats = {
  total_leads: 0,
  total_value: 0,
  stage_counts: {},
  conversion_rate: 0,
  average_value: 0,
};

describe("PipelineDashboard", () => {
  it("renders container", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("pipeline-stats")).toBeInTheDocument();
  });

  it("renders stat cards grid", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("pipeline-stats-cards")).toBeInTheDocument();
  });

  it("shows total leads count", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stat-total-leads")).toHaveTextContent("25");
  });

  it("shows total value formatted as currency", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stat-total-value")).toHaveTextContent("$500,000");
  });

  it("shows average value", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stat-avg-value")).toHaveTextContent("$20,000");
  });

  it("shows conversion rate", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stat-conversion-rate")).toHaveTextContent("66.7%");
  });

  it("shows active stages count", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stat-active-stages")).toHaveTextContent("7");
  });

  it("renders stage breakdown section", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("pipeline-stage-breakdown")).toBeInTheDocument();
  });

  it("renders rows for each active stage", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stage-row-new_lead")).toBeInTheDocument();
    expect(screen.getByTestId("stage-row-contacted")).toBeInTheDocument();
    expect(screen.getByTestId("stage-row-qualified")).toBeInTheDocument();
    expect(screen.getByTestId("stage-row-proposal")).toBeInTheDocument();
    expect(screen.getByTestId("stage-row-negotiation")).toBeInTheDocument();
    expect(screen.getByTestId("stage-row-closed_won")).toBeInTheDocument();
    expect(screen.getByTestId("stage-row-closed_lost")).toBeInTheDocument();
  });

  it("shows correct stage counts", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stage-count-new_lead")).toHaveTextContent("8");
    expect(screen.getByTestId("stage-count-closed_won")).toHaveTextContent("2");
  });

  it("renders bar elements for each stage", () => {
    render(<PipelineDashboard stats={sampleStats} />);
    expect(screen.getByTestId("stage-bar-new_lead")).toBeInTheDocument();
  });

  it("hides stage breakdown when empty stats", () => {
    render(<PipelineDashboard stats={emptyStats} />);
    expect(screen.queryByTestId("pipeline-stage-breakdown")).not.toBeInTheDocument();
  });

  it("shows zero values for empty stats", () => {
    render(<PipelineDashboard stats={emptyStats} />);
    expect(screen.getByTestId("stat-total-leads")).toHaveTextContent("0");
    expect(screen.getByTestId("stat-total-value")).toHaveTextContent("$0");
    expect(screen.getByTestId("stat-conversion-rate")).toHaveTextContent("0.0%");
  });

  it("does not render rows for stages with zero count", () => {
    const stats: PipelineStats = {
      ...sampleStats,
      stage_counts: { new_lead: 3 },
    };
    render(<PipelineDashboard stats={stats} />);
    expect(screen.getByTestId("stage-row-new_lead")).toBeInTheDocument();
    expect(screen.queryByTestId("stage-row-contacted")).not.toBeInTheDocument();
  });

  it("displays stage labels correctly", () => {
    const stats: PipelineStats = {
      ...sampleStats,
      stage_counts: { new_lead: 1 },
    };
    render(<PipelineDashboard stats={stats} />);
    expect(screen.getByText("New Lead")).toBeInTheDocument();
  });
});
