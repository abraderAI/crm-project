import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AssigneeCount } from "@/lib/reporting-types";

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
    data: AssigneeCount[];
    layout?: string;
  }): React.ReactNode {
    return React.createElement(
      "div",
      { "data-testid": "bar-chart", "data-count": data.length },
      children,
    );
  }
  function Bar({
    onClick,
  }: {
    dataKey: string;
    fill?: string;
    onClick?: (_data: unknown, index: number) => void;
    style?: Record<string, string>;
  }): React.ReactNode {
    return React.createElement("div", {
      "data-testid": "bar",
      onClick: () => onClick?.(null, 0),
    });
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

  return { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid };
});

import { TicketsByAssigneeChart } from "./tickets-by-assignee-chart";

function makeAssignees(count: number): AssigneeCount[] {
  return Array.from({ length: count }, (_, i) => ({
    user_id: `user-${i + 1}`,
    name: `User ${i + 1}`,
    count: count - i,
  }));
}

describe("TicketsByAssigneeChart", () => {
  it("renders chart for assignees", () => {
    const data = makeAssignees(5);
    render(<TicketsByAssigneeChart data={data} />);
    expect(screen.getByTestId("tickets-by-assignee-chart")).toBeInTheDocument();
    expect(screen.getByTestId("bar-chart")).toHaveAttribute("data-count", "5");
  });

  it("calls onBarClick with user_id when bar clicked", () => {
    const onClick = vi.fn();
    const data = makeAssignees(3);
    render(<TicketsByAssigneeChart data={data} onBarClick={onClick} />);
    fireEvent.click(screen.getByTestId("bar"));
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(onClick).toHaveBeenCalledWith("user-1");
  });

  it("caps at 10 entries when more than 10 assignees provided", () => {
    const data = makeAssignees(15);
    render(<TicketsByAssigneeChart data={data} />);
    expect(screen.getByTestId("bar-chart")).toHaveAttribute("data-count", "10");
  });

  it("renders empty state when data is empty", () => {
    render(<TicketsByAssigneeChart data={[]} />);
    expect(screen.getByTestId("assignee-chart-empty")).toBeInTheDocument();
  });

  it("does not call onBarClick when not provided", () => {
    const data = makeAssignees(3);
    render(<TicketsByAssigneeChart data={data} />);
    fireEvent.click(screen.getByTestId("bar"));
  });
});
