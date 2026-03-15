import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

/* eslint-disable @typescript-eslint/no-require-imports */
vi.mock("recharts", () => {
  const React = require("react");
  function ResponsiveContainer({ children }: { children: React.ReactNode }): React.ReactNode {
    return React.createElement("div", { "data-testid": "responsive-container" }, children);
  }
  function BarChart({
    children,
    data,
  }: {
    children: React.ReactNode;
    data: { name: string; count: number }[];
  }): React.ReactNode {
    return React.createElement(
      "div",
      {
        "data-testid": "bar-chart",
        "data-count": data.length,
        "data-order": data.map((d) => d.name).join(","),
      },
      children,
    );
  }
  function Bar({ children }: { children?: React.ReactNode; dataKey?: string }): React.ReactNode {
    return React.createElement("div", { "data-testid": "bar" }, children);
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
  return { ResponsiveContainer, BarChart, Bar, Cell, XAxis, YAxis, Tooltip, CartesianGrid };
});

import { TicketsByPriorityChart } from "./tickets-by-priority-chart";

const MOCK_DATA: Record<string, number> = {
  urgent: 3,
  high: 7,
  medium: 12,
  low: 5,
  none: 2,
};

describe("TicketsByPriorityChart", () => {
  it("renders in correct priority order", () => {
    render(<TicketsByPriorityChart data={MOCK_DATA} />);
    expect(screen.getByTestId("tickets-by-priority-chart")).toBeInTheDocument();
    const chart = screen.getByTestId("bar-chart");
    expect(chart).toHaveAttribute("data-order", "urgent,high,medium,low,none");
  });

  it("uses correct colour for urgent", () => {
    render(<TicketsByPriorityChart data={MOCK_DATA} />);
    const urgentCell = screen.getByTestId("priority-cell-urgent");
    expect(urgentCell).toBeInTheDocument();
    expect(urgentCell).toHaveAttribute("fill", "#ef4444");
  });

  it("renders cells for each priority", () => {
    render(<TicketsByPriorityChart data={MOCK_DATA} />);
    expect(screen.getByTestId("bar-chart")).toHaveAttribute("data-count", "5");
  });

  it("renders empty state when data is empty", () => {
    render(<TicketsByPriorityChart data={{}} />);
    expect(screen.getByTestId("priority-chart-empty")).toBeInTheDocument();
  });

  it("only renders priorities present in data", () => {
    const partial = { urgent: 1, low: 3 };
    render(<TicketsByPriorityChart data={partial} />);
    expect(screen.getByTestId("bar-chart")).toHaveAttribute("data-count", "2");
  });
});
