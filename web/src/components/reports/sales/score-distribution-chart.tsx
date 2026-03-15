"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Cell,
  ResponsiveContainer,
} from "recharts";
import type { BucketCount } from "@/lib/reporting-types";

/** 5-step colour gradient from red (low) to green (high). */
const BUCKET_COLORS: Record<string, string> = {
  "0-20": "#ef4444",
  "20-40": "#f97316",
  "40-60": "#eab308",
  "60-80": "#84cc16",
  "80-100": "#22c55e",
};

/** Default colour for unknown ranges. */
const DEFAULT_COLOR = "#3b82f6";

export interface ScoreDistributionChartProps {
  data: BucketCount[];
}

/** Histogram showing lead score distribution across buckets. */
export function ScoreDistributionChart({ data }: ScoreDistributionChartProps): React.ReactNode {
  if (data.length === 0) {
    return (
      <div
        data-testid="score-distribution-empty"
        className="py-12 text-center text-muted-foreground"
      >
        No scored leads
      </div>
    );
  }

  return (
    <div data-testid="score-distribution-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 10, right: 20, bottom: 5, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="range" tick={{ fontSize: 12 }} />
          <YAxis allowDecimals={false} />
          <Tooltip />
          <Bar dataKey="count">
            {data.map((entry) => (
              <Cell
                key={entry.range}
                fill={BUCKET_COLORS[entry.range] ?? DEFAULT_COLOR}
                data-testid={`score-bucket-${entry.range}`}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
