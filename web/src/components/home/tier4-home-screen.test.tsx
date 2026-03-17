import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Tier4HomeScreen } from "./tier4-home-screen";
import { getDefaultLayout } from "@/lib/default-layouts";

// Mock all widget components to avoid network calls in unit tests.
vi.mock("./widgets/lead-pipeline-widget", () => ({
  LeadPipelineWidget: () => <div data-testid="mock-lead-pipeline">Lead Pipeline</div>,
}));
vi.mock("./widgets/recent-leads-widget", () => ({
  RecentLeadsWidget: () => <div data-testid="mock-recent-leads">Recent Leads</div>,
}));
vi.mock("./widgets/conversion-metrics-widget", () => ({
  ConversionMetricsWidget: () => (
    <div data-testid="mock-conversion-metrics">Conversion Metrics</div>
  ),
}));
vi.mock("./widgets/ticket-queue-widget", () => ({
  TicketQueueWidget: () => <div data-testid="mock-ticket-queue">Ticket Queue</div>,
}));
vi.mock("./widgets/ticket-stats-widget", () => ({
  TicketStatsWidget: () => <div data-testid="mock-ticket-stats">Ticket Stats</div>,
}));
vi.mock("./widgets/billing-overview-widget", () => ({
  BillingOverviewWidget: () => <div data-testid="mock-billing-overview">Billing Overview</div>,
}));

describe("Tier4HomeScreen", () => {
  it("renders the home screen container", () => {
    render(
      <Tier4HomeScreen token="tok" department="sales" layout={getDefaultLayout(4, "sales")} />,
    );

    expect(screen.getByTestId("tier4-home-screen")).toBeInTheDocument();
    expect(screen.getByText("DEFT Employee Dashboard")).toBeInTheDocument();
  });

  it("shows sales department badge", () => {
    render(
      <Tier4HomeScreen token="tok" department="sales" layout={getDefaultLayout(4, "sales")} />,
    );

    expect(screen.getByTestId("tier4-department-badge")).toHaveTextContent("Sales");
  });

  it("shows support department badge", () => {
    render(
      <Tier4HomeScreen token="tok" department="support" layout={getDefaultLayout(4, "support")} />,
    );

    expect(screen.getByTestId("tier4-department-badge")).toHaveTextContent("Support");
  });

  it("shows finance department badge", () => {
    render(
      <Tier4HomeScreen token="tok" department="finance" layout={getDefaultLayout(4, "finance")} />,
    );

    expect(screen.getByTestId("tier4-department-badge")).toHaveTextContent("Finance");
  });

  it("renders sales widgets with sales layout", () => {
    render(
      <Tier4HomeScreen token="tok" department="sales" layout={getDefaultLayout(4, "sales")} />,
    );

    expect(screen.getByTestId("mock-lead-pipeline")).toBeInTheDocument();
    expect(screen.getByTestId("mock-recent-leads")).toBeInTheDocument();
    expect(screen.getByTestId("mock-conversion-metrics")).toBeInTheDocument();
  });

  it("renders support widgets with support layout", () => {
    render(
      <Tier4HomeScreen token="tok" department="support" layout={getDefaultLayout(4, "support")} />,
    );

    expect(screen.getByTestId("mock-ticket-queue")).toBeInTheDocument();
    expect(screen.getByTestId("mock-ticket-stats")).toBeInTheDocument();
  });

  it("renders finance widgets with finance layout", () => {
    render(
      <Tier4HomeScreen token="tok" department="finance" layout={getDefaultLayout(4, "finance")} />,
    );

    expect(screen.getByTestId("mock-billing-overview")).toBeInTheDocument();
  });

  it("renders HomeLayout with the grid", () => {
    render(
      <Tier4HomeScreen token="tok" department="sales" layout={getDefaultLayout(4, "sales")} />,
    );

    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
  });

  it("does not render widgets outside the layout", () => {
    render(
      <Tier4HomeScreen token="tok" department="finance" layout={getDefaultLayout(4, "finance")} />,
    );

    // Finance layout should not include sales widgets.
    expect(screen.queryByTestId("mock-lead-pipeline")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mock-ticket-queue")).not.toBeInTheDocument();
  });
});
