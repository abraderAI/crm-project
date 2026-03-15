"use client";

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import type { DailyCount } from "@/lib/reporting-types";

export interface VolumeOverTimeChartProps {
  /** Daily volume data points. */
  data: DailyCount[];
}

/** Format ISO date string as "Mar 1". */
function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr + "T00:00:00");
  if (isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

/** Recharts AreaChart showing ticket volume over time. */
export function VolumeOverTimeChart({ data }: VolumeOverTimeChartProps): React.ReactNode {
  if (data.length === 0) {
    return (
      <div data-testid="volume-empty" className="py-8 text-center text-sm text-muted-foreground">
        No data for this period
      </div>
    );
  }

  return (
    <div data-testid="volume-over-time-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" tickFormatter={formatDateLabel} fontSize={12} />
          <YAxis allowDecimals={false} fontSize={12} />
          <Tooltip
            labelFormatter={(label) => formatDateLabel(String(label))}
            formatter={(value) => [String(value), "Tickets"]}
          />
          <Area
            type="monotone"
            dataKey="count"
            stroke="#3b82f6"
            fill="#3b82f680"
            data-testid="volume-area"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
