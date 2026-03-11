import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BillingDashboard } from "./billing-dashboard";
import type { BillingInfo } from "@/lib/api-types";

const billing: BillingInfo = {
  org_id: "org1",
  tier: "pro",
  payment_status: "active",
  invoices: [
    {
      id: "inv1",
      amount: 9900,
      currency: "USD",
      status: "paid",
      issued_at: "2026-01-01T00:00:00Z",
      due_at: "2026-01-31T00:00:00Z",
      paid_at: "2026-01-05T00:00:00Z",
    },
    {
      id: "inv2",
      amount: 9900,
      currency: "USD",
      status: "open",
      issued_at: "2026-02-01T00:00:00Z",
      due_at: "2026-02-28T00:00:00Z",
    },
  ],
};

const billingEmpty: BillingInfo = {
  org_id: "org1",
  tier: "free",
  payment_status: "trialing",
  invoices: [],
};

describe("BillingDashboard", () => {
  it("renders the heading and icon", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByText("Billing")).toBeInTheDocument();
    expect(screen.getByTestId("billing-icon")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<BillingDashboard billing={null} loading={true} />);
    expect(screen.getByTestId("billing-loading")).toBeInTheDocument();
  });

  it("shows empty state when billing is null", () => {
    render(<BillingDashboard billing={null} />);
    expect(screen.getByTestId("billing-empty")).toHaveTextContent(
      "No billing information available.",
    );
  });

  it("displays billing tier", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("billing-tier-card")).toHaveTextContent("pro");
  });

  it("displays payment status badge", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("billing-payment-status")).toHaveTextContent("active");
    expect(screen.getByTestId("billing-payment-status")).toHaveClass("bg-green-100");
  });

  it("applies red color for past_due status", () => {
    const pastDue = { ...billing, payment_status: "past_due" };
    render(<BillingDashboard billing={pastDue} />);
    expect(screen.getByTestId("billing-payment-status")).toHaveClass("bg-red-100");
  });

  it("applies yellow color for trialing status", () => {
    render(<BillingDashboard billing={billingEmpty} />);
    expect(screen.getByTestId("billing-payment-status")).toHaveClass("bg-yellow-100");
  });

  it("applies default color for unknown status", () => {
    const unknown = { ...billing, payment_status: "unknown" };
    render(<BillingDashboard billing={unknown} />);
    expect(screen.getByTestId("billing-payment-status")).toHaveClass("bg-muted");
  });

  it("renders invoice list", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("invoice-list")).toBeInTheDocument();
    expect(screen.getByTestId("invoice-item-inv1")).toBeInTheDocument();
    expect(screen.getByTestId("invoice-item-inv2")).toBeInTheDocument();
  });

  it("displays invoice status badge", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("invoice-status-inv1")).toHaveTextContent("paid");
    expect(screen.getByTestId("invoice-status-inv1")).toHaveClass("bg-green-100");
  });

  it("displays invoice amount formatted", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("invoice-amount-inv1")).toHaveTextContent("$99.00");
  });

  it("displays invoice date", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("invoice-date-inv1")).toBeInTheDocument();
  });

  it("displays invoice due date", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("invoice-due-inv1")).toBeInTheDocument();
  });

  it("shows no invoices message when empty", () => {
    render(<BillingDashboard billing={billingEmpty} />);
    expect(screen.getByTestId("invoices-empty")).toHaveTextContent("No invoices yet.");
  });

  it("displays invoice count in heading", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByText("Invoices (2)")).toBeInTheDocument();
  });

  it("applies yellow color for open invoice status", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("invoice-status-inv2")).toHaveClass("bg-yellow-100");
  });

  it("renders the billing summary section", () => {
    render(<BillingDashboard billing={billing} />);
    expect(screen.getByTestId("billing-summary")).toBeInTheDocument();
  });
});
