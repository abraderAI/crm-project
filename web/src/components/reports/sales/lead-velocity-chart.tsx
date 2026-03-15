"use client";

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import type { DailyCount } from "@/lib/reporting-types";

export interface LeadVelocityChartProps {
  data: DailyCount[];
}

/** Format ISO date string to short label like "Mar 1". */
function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr + "T00:00:00");
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

/** Area chart showing new leads over time. */
export function LeadVelocityChart({ data }: LeadVelocityChartProps): React.ReactNode {
  if (data.length === 0) {
    return (
      <div data-testid="lead-velocity-empty" className="py-12 text-center text-muted-foreground">
        No lead velocity data
      </div>
    );
  }

  const formatted = data.map((d) => ({ ...d, label: formatDateLabel(d.date) }));

  return (
    <div data-testid="lead-velocity-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={formatted} margin={{ top: 10, right: 20, bottom: 5, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="label" tick={{ fontSize: 12 }} />
          <YAxis allowDecimals={false} />
          <Tooltip />
          <Area type="monotone" dataKey="count" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.2} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
