"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Cell,
  LabelList,
  ResponsiveContainer,
} from "recharts";
import type { BarRectangleItem } from "recharts";
import type { StageCount } from "@/lib/reporting-types";

/** Canonical pipeline stage order. */
const STAGE_ORDER = [
  "new_lead",
  "contacted",
  "qualified",
  "proposal",
  "negotiation",
  "closed_won",
  "closed_lost",
];

/** Map stage to display colour. */
function stageColor(stage: string): string {
  if (stage === "closed_won") return "#22c55e";
  if (stage === "closed_lost") return "#ef4444";
  return "#3b82f6";
}

/** Sort stages in pipeline order; unknown stages appended after. */
function sortStages(data: StageCount[]): StageCount[] {
  const known = new Map<string, StageCount>();
  const unknown: StageCount[] = [];

  for (const item of data) {
    if (STAGE_ORDER.includes(item.stage)) {
      known.set(item.stage, item);
    } else {
      unknown.push(item);
    }
  }

  const sorted: StageCount[] = [];
  for (const stage of STAGE_ORDER) {
    const item = known.get(stage);
    if (item) sorted.push(item);
  }
  return [...sorted, ...unknown];
}

export interface PipelineFunnelChartProps {
  data: StageCount[];
  onBarClick?: (stage: string) => void;
}

/** Pipeline funnel as a vertical bar chart with one bar per stage. */
export function PipelineFunnelChart({
  data,
  onBarClick,
}: PipelineFunnelChartProps): React.ReactNode {
  if (data.length === 0) {
    return (
      <div data-testid="pipeline-funnel-empty" className="py-12 text-center text-muted-foreground">
        No pipeline data
      </div>
    );
  }

  const sorted = sortStages(data);

  return (
    <div data-testid="pipeline-funnel-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={sorted} margin={{ top: 20, right: 20, bottom: 5, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="stage" tick={{ fontSize: 12 }} />
          <YAxis allowDecimals={false} />
          <Tooltip />
          <Bar
            dataKey="count"
            cursor="pointer"
            onClick={(_data: BarRectangleItem, index: number) => {
              if (onBarClick) onBarClick(sorted[index]!.stage);
            }}
          >
            <LabelList dataKey="count" position="top" />
            {sorted.map((entry) => (
              <Cell
                key={entry.stage}
                fill={stageColor(entry.stage)}
                data-testid={`pipeline-bar-${entry.stage}`}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
