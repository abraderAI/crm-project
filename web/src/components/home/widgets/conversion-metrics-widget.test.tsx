import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { ConversionMetricsWidget } from "./conversion-metrics-widget";

const mockFetchConversionMetrics = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchConversionMetrics: (...args: unknown[]) => mockFetchConversionMetrics(...args),
}));

describe("ConversionMetricsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchConversionMetrics.mockReturnValue(new Promise(() => {}));
    render(<ConversionMetricsWidget token="tok" />);
    expect(screen.getByTestId("conversion-metrics-loading")).toBeInTheDocument();
  });

  it("renders funnel metrics after loading", async () => {
    mockFetchConversionMetrics.mockResolvedValue({
      anonymous_sessions: 300,
      registrations: 100,
      conversions: 10,
    });

    render(<ConversionMetricsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("conversion-metrics-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("metric-anonymous")).toHaveTextContent("300");
    expect(screen.getByTestId("metric-registrations")).toHaveTextContent("100");
    expect(screen.getByTestId("metric-conversions")).toHaveTextContent("10");
  });

  it("calculates conversion rates correctly", async () => {
    mockFetchConversionMetrics.mockResolvedValue({
      anonymous_sessions: 1000,
      registrations: 200,
      conversions: 50,
    });

    render(<ConversionMetricsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("rate-registration")).toHaveTextContent("20.0%");
    });

    expect(screen.getByTestId("rate-conversion")).toHaveTextContent("25.0%");
  });

  it("handles zero denominators in rates", async () => {
    mockFetchConversionMetrics.mockResolvedValue({
      anonymous_sessions: 0,
      registrations: 0,
      conversions: 0,
    });

    render(<ConversionMetricsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("rate-registration")).toHaveTextContent("0.0%");
    });

    expect(screen.getByTestId("rate-conversion")).toHaveTextContent("0.0%");
  });

  it("shows error state on failure", async () => {
    mockFetchConversionMetrics.mockRejectedValue(new Error("fail"));

    render(<ConversionMetricsWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("conversion-metrics-error")).toBeInTheDocument();
    });
  });

  it("passes token to API function", async () => {
    mockFetchConversionMetrics.mockResolvedValue({
      anonymous_sessions: 0,
      registrations: 0,
      conversions: 0,
    });

    render(<ConversionMetricsWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchConversionMetrics).toHaveBeenCalledWith("my-token");
    });
  });
});
