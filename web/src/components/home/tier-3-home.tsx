"use client";

import type { ReactNode } from "react";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { TierSubType, WidgetConfig } from "@/lib/tier-types";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import { OrgOverviewWidget } from "./widgets/org-overview-widget";
import { OrgSupportTicketsWidget } from "./widgets/org-support-tickets-widget";
import { BillingStatusWidget } from "./widgets/billing-status-widget";
import { OrgSupportDashboardWidget } from "./widgets/org-support-dashboard-widget";
import { MyForumActivityWidget } from "./widgets/my-forum-activity-widget";

interface Tier3HomeProps {
  /** Widget layout config (from useHomeLayout or default). */
  layout: WidgetConfig[];
  /** Auth token for API calls. */
  token: string;
  /** Org ID the user belongs to. */
  orgId: string;
  /** Sub-type: "owner" if user is the org owner. */
  subType: TierSubType;
}

/**
 * Tier 3 (Paying Customer) home screen.
 * Member variant: org overview, org support tickets, forum activity.
 * Owner variant: org support dashboard, billing status, org overview.
 */
export function Tier3Home({ layout, token, orgId, subType }: Tier3HomeProps): ReactNode {
  const isOwner = subType === "owner";

  const registry: WidgetRegistry = {
    [WIDGET_IDS.ORG_OVERVIEW]: {
      title: "Organization",
      render: () => <OrgOverviewWidget token={token} orgId={orgId} isOwner={isOwner} />,
    },
    [WIDGET_IDS.ORG_SUPPORT_TICKETS]: {
      title: "Support Tickets",
      render: () => <OrgSupportTicketsWidget token={token} orgId={orgId} />,
    },
    [WIDGET_IDS.BILLING_STATUS]: {
      title: "Billing",
      render: () => <BillingStatusWidget isOwner={isOwner} />,
    },
    [WIDGET_IDS.ORG_SUPPORT_DASHBOARD]: {
      title: "Support Dashboard",
      render: () => <OrgSupportDashboardWidget token={token} orgId={orgId} />,
    },
    [WIDGET_IDS.MY_FORUM_ACTIVITY]: {
      title: "Forum Activity",
      render: () => <MyForumActivityWidget token={token} />,
    },
  };

  return (
    <div data-testid="tier-3-home">
      <h2 className="mb-4 text-lg font-semibold text-foreground">
        {isOwner ? "Organization Dashboard" : "Home"}
      </h2>
      <HomeLayout layout={layout} registry={registry} />
    </div>
  );
}
