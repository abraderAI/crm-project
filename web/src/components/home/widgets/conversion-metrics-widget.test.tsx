import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ConversionMetricsWidget } from "./conversion-metrics-widget";

describe("ConversionMetricsWidget", () => {
  it("renders and shows not-wired error", async () => {
    render(<ConversionMetricsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("conversion-metrics-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("conversion-metrics-error")).toHaveTextContent("Failed to load conversion metrics");
  });
});
