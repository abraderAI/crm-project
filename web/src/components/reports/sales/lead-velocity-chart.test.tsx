import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LeadVelocityChart } from "./lead-velocity-chart";
import type { DailyCount } from "@/lib/reporting-types";

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="responsive-container">{children}</div>
  ),
  AreaChart: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="recharts-area-chart">{children}</div>
  ),
  Area: () => <div data-testid="recharts-area" />,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
}));

const SAMPLE_DATA: DailyCount[] = [
  { date: "2026-03-01", count: 5 },
  { date: "2026-03-02", count: 8 },
  { date: "2026-03-03", count: 3 },
];

describe("LeadVelocityChart", () => {
  it("renders with data", () => {
    render(<LeadVelocityChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("lead-velocity-chart")).toBeInTheDocument();
  });

  it("shows empty state when data is empty", () => {
    render(<LeadVelocityChart data={[]} />);
    expect(screen.getByTestId("lead-velocity-empty")).toBeInTheDocument();
    expect(screen.getByText("No lead velocity data")).toBeInTheDocument();
  });

  it("does not show empty state when data is present", () => {
    render(<LeadVelocityChart data={SAMPLE_DATA} />);
    expect(screen.queryByTestId("lead-velocity-empty")).not.toBeInTheDocument();
  });

  it("renders responsive container", () => {
    render(<LeadVelocityChart data={SAMPLE_DATA} />);
    expect(screen.getByTestId("responsive-container")).toBeInTheDocument();
  });
});
