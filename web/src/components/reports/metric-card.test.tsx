import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MetricCard } from "./metric-card";

describe("MetricCard", () => {
  it("renders value and label", () => {
    render(<MetricCard label="Open Tickets" value={42} />);
    expect(screen.getByText("Open Tickets")).toBeInTheDocument();
    expect(screen.getByTestId("metric-card-value")).toHaveTextContent("42");
  });

  it("renders string value", () => {
    render(<MetricCard label="Win Rate" value="73%" />);
    expect(screen.getByTestId("metric-card-value")).toHaveTextContent("73%");
  });

  it("renders subLabel when provided", () => {
    render(<MetricCard label="Revenue" value="$1,200" subLabel="last 30 days" />);
    expect(screen.getByTestId("metric-card-sub-label")).toHaveTextContent("last 30 days");
  });

  it("does not render subLabel when not provided", () => {
    render(<MetricCard label="Count" value={10} />);
    expect(screen.queryByTestId("metric-card-sub-label")).not.toBeInTheDocument();
  });

  it("renders as Link when href is provided", () => {
    render(<MetricCard label="Tickets" value={5} href="/reports/support" />);
    const link = screen.getByTestId("metric-card-link");
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "/reports/support");
  });

  it("does not render Link when href is not provided", () => {
    render(<MetricCard label="Tickets" value={5} />);
    expect(screen.queryByTestId("metric-card-link")).not.toBeInTheDocument();
  });

  it("renders skeleton when loading is true", () => {
    render(<MetricCard label="Loading" value={0} loading={true} />);
    expect(screen.getByTestId("metric-card-skeleton")).toBeInTheDocument();
    expect(screen.queryByTestId("metric-card-value")).not.toBeInTheDocument();
  });

  it("renders value when loading is false", () => {
    render(<MetricCard label="Loaded" value={99} loading={false} />);
    expect(screen.queryByTestId("metric-card-skeleton")).not.toBeInTheDocument();
    expect(screen.getByTestId("metric-card-value")).toHaveTextContent("99");
  });

  it("renders the card container", () => {
    render(<MetricCard label="Test" value={1} />);
    expect(screen.getByTestId("metric-card")).toBeInTheDocument();
  });

  it("renders the inner content area", () => {
    render(<MetricCard label="Test" value={1} />);
    expect(screen.getByTestId("metric-card-inner")).toBeInTheDocument();
  });

  it("applies hover classes when href is provided", () => {
    render(<MetricCard label="Hover" value={1} href="/test" />);
    const card = screen.getByTestId("metric-card");
    expect(card.className).toContain("hover:border-foreground/20");
  });

  it("does not apply hover classes when no href", () => {
    render(<MetricCard label="No Hover" value={1} />);
    const card = screen.getByTestId("metric-card");
    expect(card.className).not.toContain("hover:border-foreground/20");
  });
});
