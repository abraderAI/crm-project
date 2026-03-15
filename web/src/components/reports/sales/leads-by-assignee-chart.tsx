"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import type { BarRectangleItem } from "recharts";
import type { AssigneeCount } from "@/lib/reporting-types";

/** Maximum number of assignees to display. */
const MAX_ASSIGNEES = 10;

export interface LeadsByAssigneeChartProps {
  data: AssigneeCount[];
  onBarClick?: (userId: string) => void;
}

/** Horizontal bar chart showing leads per assignee (top 10). */
export function LeadsByAssigneeChart({
  data,
  onBarClick,
}: LeadsByAssigneeChartProps): React.ReactNode {
  if (data.length === 0) {
    return (
      <div
        data-testid="leads-by-assignee-empty"
        className="py-12 text-center text-muted-foreground"
      >
        No assignee data
      </div>
    );
  }

  const capped = data
    .slice()
    .sort((a, b) => b.count - a.count)
    .slice(0, MAX_ASSIGNEES);

  return (
    <div data-testid="leads-by-assignee-chart" className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={capped}
          layout="vertical"
          margin={{ top: 5, right: 20, bottom: 5, left: 80 }}
        >
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" allowDecimals={false} />
          <YAxis dataKey="name" type="category" tick={{ fontSize: 12 }} width={80} />
          <Tooltip />
          <Bar
            dataKey="count"
            fill="#3b82f6"
            cursor="pointer"
            onClick={(_data: BarRectangleItem, index: number) => {
              if (onBarClick) onBarClick(capped[index]!.user_id);
            }}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
