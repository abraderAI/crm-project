"use client";

import type { BarProps } from "recharts";
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts";
import type { AssigneeCount } from "@/lib/reporting-types";

export interface TicketsByAssigneeChartProps {
  /** Per-assignee ticket counts. */
  data: AssigneeCount[];
  /** Called when a bar is clicked with the user_id. */
  onBarClick?: (userId: string) => void;
}

const MAX_DISPLAY = 10;

/** Recharts horizontal BarChart showing tickets by assignee (top 10). */
export function TicketsByAssigneeChart({
  data,
  onBarClick,
}: TicketsByAssigneeChartProps): React.ReactNode {
  const sorted = [...data].sort((a, b) => b.count - a.count).slice(0, MAX_DISPLAY);

  if (sorted.length === 0) {
    return (
      <div
        data-testid="assignee-chart-empty"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        No assignee data available
      </div>
    );
  }

  function handleClick(_data: unknown, index: number): void {
    const entry = sorted[index];
    if (entry && onBarClick) {
      onBarClick(entry.user_id);
    }
  }

  return (
    <div data-testid="tickets-by-assignee-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={sorted} layout="vertical">
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" allowDecimals={false} fontSize={12} />
          <YAxis type="category" dataKey="name" width={120} fontSize={12} tickLine={false} />
          <Tooltip formatter={(value) => [String(value), "Tickets"]} />
          <Bar
            dataKey="count"
            fill="#3b82f6"
            onClick={handleClick as BarProps["onClick"]}
            style={{ cursor: onBarClick ? "pointer" : "default" }}
            data-testid="assignee-bar"
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
