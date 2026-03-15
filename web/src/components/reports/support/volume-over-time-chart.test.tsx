import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DailyCount } from "@/lib/reporting-types";

/* eslint-disable @typescript-eslint/no-require-imports */
vi.mock("recharts", () => {
  const React = require("react");
  function ResponsiveContainer({ children }: { children: React.ReactNode }): React.ReactNode {
    return React.createElement("div", { "data-testid": "responsive-container" }, children);
  }
  function AreaChart({
    children,
    data,
  }: {
    children: React.ReactNode;
    data: DailyCount[];
  }): React.ReactNode {
    return React.createElement(
      "div",
      { "data-testid": "area-chart", "data-count": data.length },
      children,
    );
  }
  function Area(): React.ReactNode {
    return React.createElement("div", { "data-testid": "area" });
  }
  function XAxis(): React.ReactNode {
    return React.createElement("div", { "data-testid": "x-axis" });
  }
  function YAxis(): React.ReactNode {
    return React.createElement("div", { "data-testid": "y-axis" });
  }
  function Tooltip(): React.ReactNode {
    return null;
  }
  function CartesianGrid(): React.ReactNode {
    return null;
  }
  return { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid };
});

import { VolumeOverTimeChart } from "./volume-over-time-chart";

const MOCK_DATA: DailyCount[] = [
  { date: "2026-03-01", count: 5 },
  { date: "2026-03-02", count: 8 },
  { date: "2026-03-03", count: 3 },
];

describe("VolumeOverTimeChart", () => {
  it("renders chart with data", () => {
    render(<VolumeOverTimeChart data={MOCK_DATA} />);
    expect(screen.getByTestId("volume-over-time-chart")).toBeInTheDocument();
    expect(screen.getByTestId("area-chart")).toHaveAttribute("data-count", "3");
    expect(screen.getByTestId("area")).toBeInTheDocument();
  });

  it('shows "No data" empty state when data is empty array', () => {
    render(<VolumeOverTimeChart data={[]} />);
    expect(screen.getByTestId("volume-empty")).toBeInTheDocument();
    expect(screen.getByText("No data for this period")).toBeInTheDocument();
  });

  it("renders x-axis", () => {
    render(<VolumeOverTimeChart data={MOCK_DATA} />);
    expect(screen.getByTestId("x-axis")).toBeInTheDocument();
  });

  it("renders y-axis", () => {
    render(<VolumeOverTimeChart data={MOCK_DATA} />);
    expect(screen.getByTestId("y-axis")).toBeInTheDocument();
  });
});
