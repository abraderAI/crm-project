"use client";

import { useMemo, type ReactNode } from "react";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { DeftDepartment } from "@/lib/tier-types";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import { LeadPipelineWidget } from "./widgets/lead-pipeline-widget";
import { RecentLeadsWidget } from "./widgets/recent-leads-widget";
import { ConversionMetricsWidget } from "./widgets/conversion-metrics-widget";
import { TicketQueueWidget } from "./widgets/ticket-queue-widget";
import { TicketStatsWidget } from "./widgets/ticket-stats-widget";
import { BillingOverviewWidget } from "./widgets/billing-overview-widget";
import type { WidgetConfig } from "@/lib/tier-types";

export interface Tier4HomeScreenProps {
  /** Auth token for API calls. */
  token: string;
  /** DEFT department for the current user. */
  department: DeftDepartment;
  /** Ordered widget layout (from preferences or default). */
  layout: WidgetConfig[];
}

/**
 * Build the Tier 4 widget registry with all available DEFT employee widgets.
 * The layout config determines which widgets are visible; the registry provides all.
 */
function buildTier4Registry(token: string): WidgetRegistry {
  return {
    [WIDGET_IDS.LEAD_PIPELINE]: {
      title: "Lead Pipeline",
      render: () => <LeadPipelineWidget token={token} />,
    },
    [WIDGET_IDS.RECENT_LEADS]: {
      title: "Recent Leads",
      render: () => <RecentLeadsWidget token={token} />,
    },
    [WIDGET_IDS.CONVERSION_METRICS]: {
      title: "Conversion Metrics",
      render: () => <ConversionMetricsWidget token={token} />,
    },
    [WIDGET_IDS.TICKET_QUEUE]: {
      title: "Ticket Queue",
      render: () => <TicketQueueWidget token={token} />,
    },
    [WIDGET_IDS.TICKET_STATS]: {
      title: "Ticket Stats",
      render: () => <TicketStatsWidget token={token} />,
    },
    [WIDGET_IDS.BILLING_OVERVIEW]: {
      title: "Billing Overview",
      render: () => <BillingOverviewWidget token={token} />,
    },
  };
}

/** Department labels for display. */
const DEPARTMENT_LABELS: Record<DeftDepartment, string> = {
  sales: "Sales",
  support: "Support",
  finance: "Finance",
};

/**
 * Tier 4 (DEFT Employee) home screen.
 * Renders department-specific widgets via the HomeLayout grid.
 */
export function Tier4HomeScreen({ token, department, layout }: Tier4HomeScreenProps): ReactNode {
  const registry = useMemo(() => buildTier4Registry(token), [token]);

  return (
    <div data-testid="tier4-home-screen" className="space-y-4">
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold text-foreground">DEFT Employee Dashboard</h2>
        <span
          className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
          data-testid="tier4-department-badge"
        >
          {DEPARTMENT_LABELS[department]}
        </span>
      </div>
      <HomeLayout layout={layout} registry={registry} />
    </div>
  );
}
