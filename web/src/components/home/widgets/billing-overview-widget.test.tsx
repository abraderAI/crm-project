import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BillingOverviewWidget } from "./billing-overview-widget";

describe("BillingOverviewWidget", () => {
  it("renders and shows not-wired error", async () => {
    render(<BillingOverviewWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("billing-overview-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("billing-overview-error")).toHaveTextContent("Failed to load billing overview");
  });
});
