import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

/* eslint-disable @typescript-eslint/no-require-imports */
vi.mock("recharts", () => {
  const React = require("react");

  // Track the latest Pie onClick handler so tests can invoke it.
  let latestPieOnClick: ((_: unknown, index: number) => void) | undefined;

  function ResponsiveContainer({ children }: { children: React.ReactNode }): React.ReactNode {
    return React.createElement("div", { "data-testid": "responsive-container" }, children);
  }
  function PieChart({ children }: { children: React.ReactNode }): React.ReactNode {
    return React.createElement("div", { "data-testid": "pie-chart" }, children);
  }
  function Pie({
    children,
    data,
    onClick,
  }: {
    children: React.ReactNode;
    data: { name: string; value: number }[];
    dataKey?: string;
    nameKey?: string;
    cx?: string;
    cy?: string;
    outerRadius?: number;
    onClick?: (_: unknown, index: number) => void;
    style?: Record<string, string>;
  }): React.ReactNode {
    latestPieOnClick = onClick;
    return React.createElement(
      "div",
      { "data-testid": "pie", "data-count": data.length },
      children,
    );
  }
  function Cell({
    fill,
    "data-testid": testId,
  }: {
    fill?: string;
    "data-testid"?: string;
  }): React.ReactNode {
    return React.createElement("div", { "data-testid": testId, fill });
  }
  function Legend(): React.ReactNode {
    return React.createElement("div", { "data-testid": "legend" });
  }
  function Tooltip(): React.ReactNode {
    return null;
  }

  return {
    ResponsiveContainer,
    PieChart,
    Pie,
    Cell,
    Legend,
    Tooltip,
    __getLatestPieOnClick: () => latestPieOnClick,
  };
});

import { StatusBreakdownChart } from "./status-breakdown-chart";

const MOCK_DATA: Record<string, number> = {
  open: 14,
  in_progress: 7,
  resolved: 5,
  closed: 3,
};

describe("StatusBreakdownChart", () => {
  it("renders correct number of pie segments", () => {
    render(<StatusBreakdownChart data={MOCK_DATA} />);
    expect(screen.getByTestId("status-breakdown-chart")).toBeInTheDocument();
    expect(screen.getByTestId("pie")).toHaveAttribute("data-count", "4");
  });

  it("calls onSegmentClick with correct status when clicked", async () => {
    const onClick = vi.fn();
    render(<StatusBreakdownChart data={MOCK_DATA} onSegmentClick={onClick} />);
    // Access the onClick handler captured inside the Pie mock.
    const recharts = await import("recharts");
    const handler = (
      recharts as unknown as {
        __getLatestPieOnClick: () => ((_: unknown, index: number) => void) | undefined;
      }
    ).__getLatestPieOnClick();
    expect(handler).toBeDefined();
    handler?.(null, 0);
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(onClick).toHaveBeenCalledWith("open");
  });

  it("renders cells for all statuses", () => {
    render(<StatusBreakdownChart data={MOCK_DATA} />);
    expect(screen.getByTestId("status-cell-open")).toBeInTheDocument();
    expect(screen.getByTestId("status-cell-in_progress")).toBeInTheDocument();
    expect(screen.getByTestId("status-cell-resolved")).toBeInTheDocument();
    expect(screen.getByTestId("status-cell-closed")).toBeInTheDocument();
  });

  it("renders empty state when data is empty", () => {
    render(<StatusBreakdownChart data={{}} />);
    expect(screen.getByTestId("status-breakdown-empty")).toBeInTheDocument();
  });

  it("does not call onSegmentClick when not provided", async () => {
    render(<StatusBreakdownChart data={MOCK_DATA} />);
    const recharts = await import("recharts");
    const handler = (
      recharts as unknown as {
        __getLatestPieOnClick: () => ((_: unknown, index: number) => void) | undefined;
      }
    ).__getLatestPieOnClick();
    // Should not throw.
    handler?.(null, 0);
  });
});
