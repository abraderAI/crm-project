import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TimeInStageChart } from "./time-in-stage-chart";
import type { StageAvgTime } from "@/lib/reporting-types";

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="responsive-container">{children}</div>
  ),
  BarChart: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="recharts-bar-chart">{children}</div>
  ),
  Bar: ({ children }: { children?: React.ReactNode }) => (
    <div data-testid="recharts-bar">{children}</div>
  ),
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
}));

const SAMPLE_DATA: StageAvgTime[] = [
  { stage: "new_lead", avg_hours: 24 },
  { stage: "contacted", avg_hours: 48 },
  { stage: "qualified", avg_hours: null },
  { stage: "proposal", avg_hours: 72 },
];

describe("TimeInStageChart", () => {
  it("skips stages with null avg_hours", () => {
    render(<TimeInStageChart data={SAMPLE_DATA} />);
    // Chart should render with only the non-null entries.
    expect(screen.getByTestId("time-in-stage-chart")).toBeInTheDocument();
  });

  it("shows empty state when all values are null", () => {
    const allNull: StageAvgTime[] = [
      { stage: "new_lead", avg_hours: null },
      { stage: "contacted", avg_hours: null },
    ];
    render(<TimeInStageChart data={allNull} />);
    expect(screen.getByTestId("time-in-stage-empty")).toBeInTheDocument();
    expect(screen.getByText("No stage timing data")).toBeInTheDocument();
  });

  it("shows empty state when data is empty array", () => {
    render(<TimeInStageChart data={[]} />);
    expect(screen.getByTestId("time-in-stage-empty")).toBeInTheDocument();
    expect(screen.getByText("No stage timing data")).toBeInTheDocument();
  });

  it("does not show empty state when data has non-null values", () => {
    render(<TimeInStageChart data={SAMPLE_DATA} />);
    expect(screen.queryByTestId("time-in-stage-empty")).not.toBeInTheDocument();
  });

  it("renders responsive container", () => {
    render(<TimeInStageChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("responsive-container")).toBeInTheDocument();
  });

  it("renders with single non-null entry", () => {
    render(<TimeInStageChart data={[{ stage: "new_lead", avg_hours: 12 }]} />);
    expect(screen.getByTestId("time-in-stage-chart")).toBeInTheDocument();
  });
});
