import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Tier5Home } from "./tier-5-home";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";

// Mock child widgets.
vi.mock("./widgets/org-access-control-widget", () => ({
  OrgAccessControlWidget: () => <div data-testid="mock-org-access-control">Access Controls</div>,
}));
vi.mock("./widgets/org-rbac-editor-widget", () => ({
  OrgRBACEditorWidget: () => <div data-testid="mock-org-rbac-editor">RBAC Editor</div>,
}));
vi.mock("./widgets/org-support-dashboard-widget", () => ({
  OrgSupportDashboardWidget: () => (
    <div data-testid="mock-org-support-dashboard">Support Dashboard</div>
  ),
}));
vi.mock("./widgets/billing-status-widget", () => ({
  BillingStatusWidget: () => <div data-testid="mock-billing-status">Billing Status</div>,
}));

const DEFAULT_LAYOUT: WidgetConfig[] = [
  { widget_id: WIDGET_IDS.ORG_ACCESS_CONTROL, visible: true },
  { widget_id: WIDGET_IDS.ORG_RBAC_EDITOR, visible: true },
  { widget_id: WIDGET_IDS.ORG_SUPPORT_DASHBOARD, visible: true },
  { widget_id: WIDGET_IDS.BILLING_STATUS, visible: true },
];

describe("Tier5Home", () => {
  it("renders the tier 5 home container", () => {
    render(<Tier5Home layout={DEFAULT_LAYOUT} token="token" orgId="org-1" />);
    expect(screen.getByTestId("tier-5-home")).toBeInTheDocument();
  });

  it("displays 'Organization Administration' heading", () => {
    render(<Tier5Home layout={DEFAULT_LAYOUT} token="token" orgId="org-1" />);
    expect(screen.getByText("Organization Administration")).toBeInTheDocument();
  });

  it("renders all four Tier 5 widgets", () => {
    render(<Tier5Home layout={DEFAULT_LAYOUT} token="token" orgId="org-1" />);
    expect(screen.getByTestId("mock-org-access-control")).toBeInTheDocument();
    expect(screen.getByTestId("mock-org-rbac-editor")).toBeInTheDocument();
    expect(screen.getByTestId("mock-org-support-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("mock-billing-status")).toBeInTheDocument();
  });

  it("renders the home layout grid", () => {
    render(<Tier5Home layout={DEFAULT_LAYOUT} token="token" orgId="org-1" />);
    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
  });

  it("respects visibility settings in layout", () => {
    const layout: WidgetConfig[] = [
      { widget_id: WIDGET_IDS.ORG_ACCESS_CONTROL, visible: true },
      { widget_id: WIDGET_IDS.ORG_RBAC_EDITOR, visible: false },
      { widget_id: WIDGET_IDS.ORG_SUPPORT_DASHBOARD, visible: true },
      { widget_id: WIDGET_IDS.BILLING_STATUS, visible: false },
    ];
    render(<Tier5Home layout={layout} token="token" orgId="org-1" />);
    expect(screen.getByTestId("mock-org-access-control")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-org-rbac-editor")).not.toBeInTheDocument();
    expect(screen.getByTestId("mock-org-support-dashboard")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-billing-status")).not.toBeInTheDocument();
  });
});
