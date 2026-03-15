"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import type { StageConversion } from "@/lib/reporting-types";

export interface StageConversionChartProps {
  data: StageConversion[];
}

/** Row for the chart: dominant conversion per from_stage. */
interface DominantConversion {
  from_stage: string;
  to_stage: string;
  rate: number;
}

/** Pick only the dominant (highest rate) transition per from_stage. */
function dominantConversions(data: StageConversion[]): DominantConversion[] {
  const best = new Map<string, DominantConversion>();

  for (const item of data) {
    const existing = best.get(item.from_stage);
    if (!existing || item.rate > existing.rate) {
      best.set(item.from_stage, {
        from_stage: item.from_stage,
        to_stage: item.to_stage,
        rate: item.rate,
      });
    }
  }

  return Array.from(best.values());
}

/** Bar chart showing conversion rate (%) per from_stage. */
export function StageConversionChart({ data }: StageConversionChartProps): React.ReactNode {
  if (data.length === 0) {
    return (
      <div data-testid="stage-conversion-empty" className="py-12 text-center text-muted-foreground">
        No conversion data
      </div>
    );
  }

  const chartData = dominantConversions(data).map((d) => ({
    ...d,
    percentage: Math.round(d.rate * 100 * 10) / 10,
  }));

  return (
    <div data-testid="stage-conversion-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData} margin={{ top: 10, right: 20, bottom: 5, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="from_stage" tick={{ fontSize: 12 }} />
          <YAxis domain={[0, 100]} unit="%" allowDecimals={false} />
          <Tooltip
            formatter={(value: unknown) => [`${Number(value)}%`, "Conversion"]}
            labelFormatter={(label: unknown) => `From: ${String(label)}`}
          />
          <Bar dataKey="percentage" fill="#8b5cf6" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
