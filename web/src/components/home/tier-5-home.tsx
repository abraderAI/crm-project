"use client";

import type { ReactNode } from "react";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import { OrgAccessControlWidget } from "./widgets/org-access-control-widget";
import { OrgRBACEditorWidget } from "./widgets/org-rbac-editor-widget";
import { OrgSupportDashboardWidget } from "./widgets/org-support-dashboard-widget";
import { BillingStatusWidget } from "./widgets/billing-status-widget";

interface Tier5HomeProps {
  /** Widget layout config (from useHomeLayout or default). */
  layout: WidgetConfig[];
  /** Auth token for API calls. */
  token: string;
  /** Org ID the admin manages. */
  orgId: string;
}

/**
 * Tier 5 (Customer Org Admin) home screen.
 * Shows org access controls, RBAC editor, support dashboard, and billing status.
 */
export function Tier5Home({ layout, token, orgId }: Tier5HomeProps): ReactNode {
  const registry: WidgetRegistry = {
    [WIDGET_IDS.ORG_ACCESS_CONTROL]: {
      title: "Access Controls",
      render: () => <OrgAccessControlWidget token={token} orgId={orgId} />,
    },
    [WIDGET_IDS.ORG_RBAC_EDITOR]: {
      title: "RBAC Editor",
      render: () => <OrgRBACEditorWidget token={token} orgId={orgId} />,
    },
    [WIDGET_IDS.ORG_SUPPORT_DASHBOARD]: {
      title: "Support Dashboard",
      render: () => <OrgSupportDashboardWidget token={token} orgId={orgId} />,
    },
    [WIDGET_IDS.BILLING_STATUS]: {
      title: "Billing",
      render: () => <BillingStatusWidget isOwner />,
    },
  };

  return (
    <div data-testid="tier-5-home">
      <h2 className="mb-4 text-lg font-semibold text-foreground">Organization Administration</h2>
      <HomeLayout layout={layout} registry={registry} />
    </div>
  );
}
