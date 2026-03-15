import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StageConversionChart } from "./stage-conversion-chart";
import type { StageConversion } from "@/lib/reporting-types";

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

const SAMPLE_DATA: StageConversion[] = [
  { from_stage: "new_lead", to_stage: "contacted", rate: 0.75 },
  { from_stage: "new_lead", to_stage: "closed_lost", rate: 0.25 },
  { from_stage: "contacted", to_stage: "qualified", rate: 0.6 },
  { from_stage: "qualified", to_stage: "proposal", rate: 0.5 },
];

describe("StageConversionChart", () => {
  it("renders bars for each from_stage", () => {
    render(<StageConversionChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("stage-conversion-chart")).toBeInTheDocument();
  });

  it("shows 'No conversion data' when data is empty array", () => {
    render(<StageConversionChart data={[]} />);
    expect(screen.getByTestId("stage-conversion-empty")).toBeInTheDocument();
    expect(screen.getByText("No conversion data")).toBeInTheDocument();
  });

  it("does not show empty state when data is present", () => {
    render(<StageConversionChart data={SAMPLE_DATA} />);
    expect(screen.queryByTestId("stage-conversion-empty")).not.toBeInTheDocument();
  });

  it("renders responsive container", () => {
    render(<StageConversionChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("responsive-container")).toBeInTheDocument();
  });

  it("handles single conversion entry", () => {
    render(
      <StageConversionChart
        data={[{ from_stage: "new_lead", to_stage: "contacted", rate: 0.8 }]}
      />,
    );
    expect(screen.getByTestId("stage-conversion-chart")).toBeInTheDocument();
  });
});
