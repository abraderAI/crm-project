import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BillingOverviewWidget } from "./billing-overview-widget";

describe("BillingOverviewWidget", () => {
  it("renders without crashing", () => {
    const { container } = render(<BillingOverviewWidget token="tok" />);
    expect(container).toBeTruthy();
  });

  it("shows not-wired error after load", async () => {
    render(<BillingOverviewWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("billing-overview-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("billing-overview-error")).toHaveTextContent(
      "Failed to load billing overview",
    );
  });
});
