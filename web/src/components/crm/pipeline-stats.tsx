"use client";

import { Users, DollarSign, TrendingUp, BarChart3, Target } from "lucide-react";
import type { PipelineStats } from "@/lib/crm-types";
import { formatCurrency, STAGE_LABELS, PIPELINE_STAGES } from "@/lib/crm-types";

export interface PipelineStatsProps {
  stats: PipelineStats;
}

interface StatCardProps {
  icon: typeof Users;
  label: string;
  value: string;
  testId: string;
}

function StatCard({ icon: Icon, label, value, testId }: StatCardProps): React.ReactNode {
  return (
    <div className="rounded-lg border border-border bg-background p-4" data-testid={testId}>
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="text-xs text-muted-foreground">{label}</span>
      </div>
      <p className="mt-1 text-lg font-bold text-foreground">{value}</p>
    </div>
  );
}

/** Pipeline dashboard showing summary statistics. */
export function PipelineDashboard({ stats }: PipelineStatsProps): React.ReactNode {
  return (
    <div data-testid="pipeline-stats">
      {/* Summary cards */}
      <div
        className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5"
        data-testid="pipeline-stats-cards"
      >
        <StatCard
          icon={Users}
          label="Total Leads"
          value={String(stats.total_leads)}
          testId="stat-total-leads"
        />
        <StatCard
          icon={DollarSign}
          label="Total Value"
          value={formatCurrency(stats.total_value)}
          testId="stat-total-value"
        />
        <StatCard
          icon={TrendingUp}
          label="Avg Value"
          value={formatCurrency(stats.average_value)}
          testId="stat-avg-value"
        />
        <StatCard
          icon={Target}
          label="Conversion Rate"
          value={`${stats.conversion_rate.toFixed(1)}%`}
          testId="stat-conversion-rate"
        />
        <StatCard
          icon={BarChart3}
          label="Active Stages"
          value={String(Object.keys(stats.stage_counts).length)}
          testId="stat-active-stages"
        />
      </div>

      {/* Stage breakdown */}
      {Object.keys(stats.stage_counts).length > 0 && (
        <div
          className="mt-4 rounded-lg border border-border bg-background p-4"
          data-testid="pipeline-stage-breakdown"
        >
          <h3 className="mb-3 text-sm font-semibold text-foreground">Stage Breakdown</h3>
          <div className="space-y-2">
            {PIPELINE_STAGES.filter((s) => (stats.stage_counts[s] ?? 0) > 0).map((stage) => {
              const count = stats.stage_counts[stage] ?? 0;
              const pct = stats.total_leads > 0 ? (count / stats.total_leads) * 100 : 0;
              return (
                <div
                  key={stage}
                  className="flex items-center gap-3"
                  data-testid={`stage-row-${stage}`}
                >
                  <span className="w-24 text-xs text-muted-foreground">{STAGE_LABELS[stage]}</span>
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary transition-all"
                      style={{ width: `${pct}%` }}
                      data-testid={`stage-bar-${stage}`}
                    />
                  </div>
                  <span
                    className="w-8 text-right text-xs font-medium text-foreground"
                    data-testid={`stage-count-${stage}`}
                  >
                    {count}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
