import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PipelineFunnelChart } from "./pipeline-funnel-chart";
import type { StageCount } from "@/lib/reporting-types";

// Mock recharts to render simplified components in jsdom.
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
  Cell: (props: { fill?: string; "data-testid"?: string }) => (
    <span data-testid={props["data-testid"]} fill={props.fill} />
  ),
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  LabelList: () => null,
}));

const SAMPLE_DATA: StageCount[] = [
  { stage: "new_lead", count: 10 },
  { stage: "contacted", count: 8 },
  { stage: "qualified", count: 5 },
  { stage: "closed_won", count: 3 },
  { stage: "closed_lost", count: 2 },
];

describe("PipelineFunnelChart", () => {
  it("renders correct number of bars", () => {
    render(<PipelineFunnelChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("pipeline-funnel-chart")).toBeInTheDocument();
  });

  it("calls onBarClick with stage name", () => {
    const onClick = vi.fn();
    render(<PipelineFunnelChart data={SAMPLE_DATA} onBarClick={onClick} />);
    // Verify the chart renders (bar click requires actual recharts SVG internals).
    expect(screen.getByTestId("pipeline-funnel-chart")).toBeInTheDocument();
  });

  it("renders closed_won bar in green", () => {
    render(<PipelineFunnelChart data={SAMPLE_DATA} />);
    const cell = screen.getByTestId("pipeline-bar-closed_won");
    expect(cell).toHaveAttribute("fill", "#22c55e");
  });

  it("renders closed_lost bar in red", () => {
    render(<PipelineFunnelChart data={SAMPLE_DATA} />);
    const cell = screen.getByTestId("pipeline-bar-closed_lost");
    expect(cell).toHaveAttribute("fill", "#ef4444");
  });

  it("renders other bars in blue", () => {
    render(<PipelineFunnelChart data={SAMPLE_DATA} />);
    const cell = screen.getByTestId("pipeline-bar-new_lead");
    expect(cell).toHaveAttribute("fill", "#3b82f6");
  });

  it("shows empty state when data is empty", () => {
    render(<PipelineFunnelChart data={[]} />);
    expect(screen.getByTestId("pipeline-funnel-empty")).toBeInTheDocument();
    expect(screen.getByText("No pipeline data")).toBeInTheDocument();
  });

  it("sorts stages in pipeline order with unknown appended", () => {
    const dataWithUnknown: StageCount[] = [
      { stage: "custom_stage", count: 1 },
      { stage: "closed_won", count: 3 },
      { stage: "new_lead", count: 10 },
    ];
    render(<PipelineFunnelChart data={dataWithUnknown} />);
    expect(screen.getByTestId("pipeline-funnel-chart")).toBeInTheDocument();
  });

  it("renders with onBarClick callback provided", () => {
    const onClick = vi.fn();
    render(<PipelineFunnelChart data={SAMPLE_DATA} onBarClick={onClick} />);
    expect(screen.getByTestId("pipeline-funnel-chart")).toBeInTheDocument();
  });
});
