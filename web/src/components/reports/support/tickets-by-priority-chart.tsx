"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Cell,
} from "recharts";

export interface TicketsByPriorityChartProps {
  /** Priority → count mapping. */
  data: Record<string, number>;
}

/** Fixed display order for priorities. */
const PRIORITY_ORDER = ["urgent", "high", "medium", "low", "none"] as const;

/** Colour map for each priority level. */
const PRIORITY_COLOURS: Record<string, string> = {
  urgent: "#ef4444",
  high: "#f97316",
  medium: "#eab308",
  low: "#3b82f6",
  none: "#6b7280",
};

const DEFAULT_COLOUR = "#94a3b8";

interface PriorityEntry {
  name: string;
  count: number;
}

/** Recharts BarChart showing tickets by priority in fixed order. */
export function TicketsByPriorityChart({ data }: TicketsByPriorityChartProps): React.ReactNode {
  const entries: PriorityEntry[] = PRIORITY_ORDER.filter((p) => data[p] !== undefined).map((p) => ({
    name: p,
    count: data[p] ?? 0,
  }));

  if (entries.length === 0) {
    return (
      <div
        data-testid="priority-chart-empty"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        No priority data available
      </div>
    );
  }

  return (
    <div data-testid="tickets-by-priority-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={entries}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" fontSize={12} />
          <YAxis allowDecimals={false} fontSize={12} />
          <Tooltip formatter={(value) => [String(value), "Tickets"]} />
          <Bar dataKey="count" data-testid="priority-bar">
            {entries.map((entry) => (
              <Cell
                key={entry.name}
                fill={PRIORITY_COLOURS[entry.name] ?? DEFAULT_COLOUR}
                data-testid={`priority-cell-${entry.name}`}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
