import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Tier6HomeScreen } from "./tier6-home-screen";
import { getDefaultLayout } from "@/lib/default-layouts";

// Mock all widget components.
vi.mock("./widgets/system-health-widget", () => ({
  SystemHealthWidget: () => <div data-testid="mock-system-health">System Health</div>,
}));
vi.mock("./widgets/recent-audit-log-widget", () => ({
  RecentAuditLogWidget: () => <div data-testid="mock-audit-log">Audit Log</div>,
}));
vi.mock("./widgets/lead-pipeline-widget", () => ({
  LeadPipelineWidget: () => <div data-testid="mock-lead-pipeline">Lead Pipeline</div>,
}));
vi.mock("./widgets/recent-leads-widget", () => ({
  RecentLeadsWidget: () => <div data-testid="mock-recent-leads">Recent Leads</div>,
}));
vi.mock("./widgets/conversion-metrics-widget", () => ({
  ConversionMetricsWidget: () => <div data-testid="mock-conversion-metrics">Conversion</div>,
}));
vi.mock("./widgets/ticket-queue-widget", () => ({
  TicketQueueWidget: () => <div data-testid="mock-ticket-queue">Ticket Queue</div>,
}));
vi.mock("./widgets/ticket-stats-widget", () => ({
  TicketStatsWidget: () => <div data-testid="mock-ticket-stats">Ticket Stats</div>,
}));
vi.mock("./widgets/billing-overview-widget", () => ({
  BillingOverviewWidget: () => <div data-testid="mock-billing-overview">Billing</div>,
}));

describe("Tier6HomeScreen", () => {
  const defaultLayout = getDefaultLayout(6);

  it("renders the home screen container", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("tier6-home-screen")).toBeInTheDocument();
    expect(screen.getByText("Platform Admin Dashboard")).toBeInTheDocument();
  });

  it("shows admin badge", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("tier6-admin-badge")).toHaveTextContent("Admin");
  });

  it("renders quick links", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("tier6-quick-links")).toBeInTheDocument();
    expect(screen.getByTestId("link-user-management")).toHaveAttribute("href", "/admin/users");
    expect(screen.getByTestId("link-feature-flags")).toHaveAttribute(
      "href",
      "/admin/feature-flags",
    );
    expect(screen.getByTestId("link-audit-log")).toHaveAttribute("href", "/admin/audit-log");
  });

  it("renders system health widget", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("mock-system-health")).toBeInTheDocument();
  });

  it("renders audit log widget", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("mock-audit-log")).toBeInTheDocument();
  });

  it("renders all T4 widgets", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("mock-lead-pipeline")).toBeInTheDocument();
    expect(screen.getByTestId("mock-recent-leads")).toBeInTheDocument();
    expect(screen.getByTestId("mock-conversion-metrics")).toBeInTheDocument();
    expect(screen.getByTestId("mock-ticket-queue")).toBeInTheDocument();
    expect(screen.getByTestId("mock-ticket-stats")).toBeInTheDocument();
    expect(screen.getByTestId("mock-billing-overview")).toBeInTheDocument();
  });

  it("renders HomeLayout with the grid", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
  });

  it("renders all default layout widgets (no widget hidden)", () => {
    render(<Tier6HomeScreen token="tok" layout={defaultLayout} />);

    const grid = screen.getByTestId("home-layout");
    const widgets = grid.querySelectorAll("[data-widget-id]");

    // Tier 6 default: 8 widgets (system-health, audit-log + 6 T4).
    expect(widgets.length).toBe(8);
  });
});
