"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import type { StageAvgTime } from "@/lib/reporting-types";

export interface TimeInStageChartProps {
  data: StageAvgTime[];
}

/** Horizontal bar chart showing average time per pipeline stage. */
export function TimeInStageChart({ data }: TimeInStageChartProps): React.ReactNode {
  const filtered = data.filter((d) => d.avg_hours !== null);

  if (filtered.length === 0) {
    return (
      <div data-testid="time-in-stage-empty" className="py-12 text-center text-muted-foreground">
        No stage timing data
      </div>
    );
  }

  return (
    <div data-testid="time-in-stage-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={filtered}
          layout="vertical"
          margin={{ top: 5, right: 20, bottom: 5, left: 80 }}
        >
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" allowDecimals />
          <YAxis dataKey="stage" type="category" tick={{ fontSize: 12 }} width={80} />
          <Tooltip
            formatter={(value: unknown) => [`${Number(value).toFixed(1)} hrs`, "Avg Time"]}
          />
          <Bar dataKey="avg_hours" fill="#f59e0b" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
