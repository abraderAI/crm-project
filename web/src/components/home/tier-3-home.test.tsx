import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Tier3Home } from "./tier-3-home";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";

// Mock child widgets to avoid side effects.
vi.mock("./widgets/org-overview-widget", () => ({
  OrgOverviewWidget: ({ isOwner }: { isOwner?: boolean }) => (
    <div data-testid="mock-org-overview">{isOwner ? "Owner view" : "Member view"}</div>
  ),
}));
vi.mock("./widgets/org-support-tickets-widget", () => ({
  OrgSupportTicketsWidget: () => (
    <div data-testid="mock-org-support-tickets">Org Support Tickets</div>
  ),
}));
vi.mock("./widgets/billing-status-widget", () => ({
  BillingStatusWidget: ({ isOwner }: { isOwner: boolean }) =>
    isOwner ? <div data-testid="mock-billing-status">Billing Status</div> : null,
}));
vi.mock("./widgets/org-support-dashboard-widget", () => ({
  OrgSupportDashboardWidget: () => (
    <div data-testid="mock-org-support-dashboard">Support Dashboard</div>
  ),
}));
vi.mock("./widgets/my-forum-activity-widget", () => ({
  MyForumActivityWidget: () => <div data-testid="mock-my-forum-activity">Forum Activity</div>,
}));

const MEMBER_LAYOUT: WidgetConfig[] = [
  { widget_id: WIDGET_IDS.ORG_OVERVIEW, visible: true },
  { widget_id: WIDGET_IDS.ORG_SUPPORT_TICKETS, visible: true },
  { widget_id: WIDGET_IDS.MY_FORUM_ACTIVITY, visible: true },
];

const OWNER_LAYOUT: WidgetConfig[] = [
  { widget_id: WIDGET_IDS.ORG_SUPPORT_DASHBOARD, visible: true },
  { widget_id: WIDGET_IDS.BILLING_STATUS, visible: true },
  { widget_id: WIDGET_IDS.ORG_OVERVIEW, visible: true },
];

describe("Tier3Home", () => {
  it("renders the tier 3 home container", () => {
    render(<Tier3Home layout={MEMBER_LAYOUT} token="token" orgId="org-1" subType={null} />);
    expect(screen.getByTestId("tier-3-home")).toBeInTheDocument();
  });

  it("displays 'Home' heading for members", () => {
    render(<Tier3Home layout={MEMBER_LAYOUT} token="token" orgId="org-1" subType={null} />);
    expect(screen.getByText("Home")).toBeInTheDocument();
  });

  it("displays 'Organization Dashboard' heading for owners", () => {
    render(<Tier3Home layout={OWNER_LAYOUT} token="token" orgId="org-1" subType="owner" />);
    expect(screen.getByText("Organization Dashboard")).toBeInTheDocument();
  });

  it("renders member widgets for member variant", () => {
    render(<Tier3Home layout={MEMBER_LAYOUT} token="token" orgId="org-1" subType={null} />);
    expect(screen.getByTestId("mock-org-overview")).toHaveTextContent("Member view");
    expect(screen.getByTestId("mock-org-support-tickets")).toBeInTheDocument();
    expect(screen.getByTestId("mock-my-forum-activity")).toBeInTheDocument();
  });

  it("renders owner widgets for owner variant", () => {
    render(<Tier3Home layout={OWNER_LAYOUT} token="token" orgId="org-1" subType="owner" />);
    expect(screen.getByTestId("mock-org-support-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("mock-billing-status")).toBeInTheDocument();
    expect(screen.getByTestId("mock-org-overview")).toHaveTextContent("Owner view");
  });

  it("renders home layout grid", () => {
    render(<Tier3Home layout={MEMBER_LAYOUT} token="token" orgId="org-1" subType={null} />);
    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
  });

  it("respects visibility settings in layout", () => {
    const layout: WidgetConfig[] = [
      { widget_id: WIDGET_IDS.ORG_OVERVIEW, visible: true },
      { widget_id: WIDGET_IDS.ORG_SUPPORT_TICKETS, visible: false },
      { widget_id: WIDGET_IDS.MY_FORUM_ACTIVITY, visible: true },
    ];
    render(<Tier3Home layout={layout} token="token" orgId="org-1" subType={null} />);
    expect(screen.getByTestId("mock-org-overview")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-org-support-tickets")).not.toBeInTheDocument();
    expect(screen.getByTestId("mock-my-forum-activity")).toBeInTheDocument();
  });
});
