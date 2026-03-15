"use client";

import { PieChart, Pie, Cell, Legend, ResponsiveContainer, Tooltip } from "recharts";

export interface StatusBreakdownChartProps {
  /** Status → count mapping, e.g. { open: 14, in_progress: 7 }. */
  data: Record<string, number>;
  /** Called when a pie segment is clicked with the status key. */
  onSegmentClick?: (status: string) => void;
}

/** Colour map for known ticket statuses. */
const STATUS_COLOURS: Record<string, string> = {
  open: "#3b82f6",
  in_progress: "#eab308",
  resolved: "#22c55e",
  closed: "#6b7280",
};

const DEFAULT_COLOUR = "#94a3b8";

interface PieEntry {
  name: string;
  value: number;
}

/** Recharts PieChart showing the distribution of tickets by status. */
export function StatusBreakdownChart({
  data,
  onSegmentClick,
}: StatusBreakdownChartProps): React.ReactNode {
  const entries: PieEntry[] = Object.entries(data).map(([name, value]) => ({
    name,
    value,
  }));

  if (entries.length === 0) {
    return (
      <div
        data-testid="status-breakdown-empty"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        No status data available
      </div>
    );
  }

  function handleClick(_: unknown, index: number): void {
    const entry = entries[index];
    if (entry && onSegmentClick) {
      onSegmentClick(entry.name);
    }
  }

  return (
    <div data-testid="status-breakdown-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={entries}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="45%"
            outerRadius={80}
            onClick={handleClick}
            style={{ cursor: onSegmentClick ? "pointer" : "default" }}
          >
            {entries.map((entry) => (
              <Cell
                key={entry.name}
                fill={STATUS_COLOURS[entry.name] ?? DEFAULT_COLOUR}
                data-testid={`status-cell-${entry.name}`}
              />
            ))}
          </Pie>
          <Tooltip />
          <Legend verticalAlign="bottom" data-testid="status-legend" />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
