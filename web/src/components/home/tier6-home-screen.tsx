"use client";

import { useMemo, type ReactNode } from "react";
import Link from "next/link";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import { LeadPipelineWidget } from "./widgets/lead-pipeline-widget";
import { RecentLeadsWidget } from "./widgets/recent-leads-widget";
import { ConversionMetricsWidget } from "./widgets/conversion-metrics-widget";
import { TicketQueueWidget } from "./widgets/ticket-queue-widget";
import { TicketStatsWidget } from "./widgets/ticket-stats-widget";
import { BillingOverviewWidget } from "./widgets/billing-overview-widget";
import { SystemHealthWidget } from "./widgets/system-health-widget";
import { RecentAuditLogWidget } from "./widgets/recent-audit-log-widget";

export interface Tier6HomeScreenProps {
  /** Auth token for API calls. */
  token: string;
  /** Ordered widget layout (from preferences or default). */
  layout: WidgetConfig[];
}

/**
 * Build the Tier 6 widget registry with all Tier 4 widgets plus admin-only widgets.
 * No widget is hidden by default for platform admins.
 */
function buildTier6Registry(token: string): WidgetRegistry {
  return {
    [WIDGET_IDS.SYSTEM_HEALTH]: {
      title: "System Health",
      render: () => <SystemHealthWidget token={token} />,
    },
    [WIDGET_IDS.RECENT_AUDIT_LOG]: {
      title: "Recent Audit Log",
      render: () => <RecentAuditLogWidget token={token} />,
    },
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

/**
 * Tier 6 (Platform Admin) home screen.
 * Shows system health, audit log, all T4 widgets, and admin quick links.
 */
export function Tier6HomeScreen({ token, layout }: Tier6HomeScreenProps): ReactNode {
  const registry = useMemo(() => buildTier6Registry(token), [token]);

  return (
    <div data-testid="tier6-home-screen" className="space-y-4">
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold text-foreground">Platform Admin Dashboard</h2>
        <span
          className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800"
          data-testid="tier6-admin-badge"
        >
          Admin
        </span>
      </div>

      {/* Quick links */}
      <div className="flex gap-3" data-testid="tier6-quick-links">
        <Link
          href="/admin/users"
          className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent/50"
          data-testid="link-user-management"
        >
          User Management
        </Link>
        <Link
          href="/admin/feature-flags"
          className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent/50"
          data-testid="link-feature-flags"
        >
          Feature Flags
        </Link>
        <Link
          href="/admin/audit-log"
          className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent/50"
          data-testid="link-audit-log"
        >
          Full Audit Log
        </Link>
      </div>

      <HomeLayout layout={layout} registry={registry} />
    </div>
  );
}
