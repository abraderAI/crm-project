import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { BillingOverviewWidget } from "./billing-overview-widget";

const mockFetchBillingOverview = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchBillingOverview: (...args: unknown[]) => mockFetchBillingOverview(...args),
}));

describe("BillingOverviewWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchBillingOverview.mockReturnValue(new Promise(() => {}));
    render(<BillingOverviewWidget token="tok" />);
    expect(screen.getByTestId("billing-overview-loading")).toBeInTheDocument();
  });

  it("renders billing data after loading", async () => {
    mockFetchBillingOverview.mockResolvedValue({
      paying_org_count: 25,
      mrr: 5000,
      recent_payments: 12,
    });

    render(<BillingOverviewWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("billing-overview-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("billing-orgs")).toHaveTextContent("25");
    expect(screen.getByTestId("billing-mrr")).toHaveTextContent("$5,000");
    expect(screen.getByTestId("billing-payments")).toHaveTextContent("12");
  });

  it("shows stub note when MRR is zero", async () => {
    mockFetchBillingOverview.mockResolvedValue({
      paying_org_count: 5,
      mrr: 0,
      recent_payments: 0,
    });

    render(<BillingOverviewWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("billing-stub-note")).toBeInTheDocument();
    });
  });

  it("hides stub note when MRR is non-zero", async () => {
    mockFetchBillingOverview.mockResolvedValue({
      paying_org_count: 5,
      mrr: 1000,
      recent_payments: 3,
    });

    render(<BillingOverviewWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("billing-overview-content")).toBeInTheDocument();
    });

    expect(screen.queryByTestId("billing-stub-note")).not.toBeInTheDocument();
  });

  it("shows error state on failure", async () => {
    mockFetchBillingOverview.mockRejectedValue(new Error("fail"));

    render(<BillingOverviewWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("billing-overview-error")).toBeInTheDocument();
    });
  });

  it("passes token to API", async () => {
    mockFetchBillingOverview.mockResolvedValue({
      paying_org_count: 0,
      mrr: 0,
      recent_payments: 0,
    });

    render(<BillingOverviewWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchBillingOverview).toHaveBeenCalledWith("my-token");
    });
  });
});
