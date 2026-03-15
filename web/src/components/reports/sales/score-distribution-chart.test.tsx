import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ScoreDistributionChart } from "./score-distribution-chart";
import type { BucketCount } from "@/lib/reporting-types";

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
}));

const SAMPLE_DATA: BucketCount[] = [
  { range: "0-20", count: 5 },
  { range: "20-40", count: 10 },
  { range: "40-60", count: 15 },
  { range: "60-80", count: 8 },
  { range: "80-100", count: 3 },
];

describe("ScoreDistributionChart", () => {
  it("renders all 5 buckets when data present", () => {
    render(<ScoreDistributionChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("score-distribution-chart")).toBeInTheDocument();
    expect(screen.getByTestId("score-bucket-0-20")).toBeInTheDocument();
    expect(screen.getByTestId("score-bucket-20-40")).toBeInTheDocument();
    expect(screen.getByTestId("score-bucket-40-60")).toBeInTheDocument();
    expect(screen.getByTestId("score-bucket-60-80")).toBeInTheDocument();
    expect(screen.getByTestId("score-bucket-80-100")).toBeInTheDocument();
  });

  it("shows empty state when data is empty", () => {
    render(<ScoreDistributionChart data={[]} />);
    expect(screen.getByTestId("score-distribution-empty")).toBeInTheDocument();
    expect(screen.getByText("No scored leads")).toBeInTheDocument();
  });

  it("applies correct colour for low score bucket", () => {
    render(<ScoreDistributionChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("score-bucket-0-20")).toHaveAttribute("fill", "#ef4444");
  });

  it("applies correct colour for high score bucket", () => {
    render(<ScoreDistributionChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("score-bucket-80-100")).toHaveAttribute("fill", "#22c55e");
  });

  it("does not show empty state when data is present", () => {
    render(<ScoreDistributionChart data={SAMPLE_DATA} />);
    expect(screen.queryByTestId("score-distribution-empty")).not.toBeInTheDocument();
  });

  it("renders responsive container", () => {
    render(<ScoreDistributionChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("responsive-container")).toBeInTheDocument();
  });
});
