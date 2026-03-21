import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock next/navigation.
const mockPush = vi.fn();
const mockSearchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
  useSearchParams: () => mockSearchParams,
}));

/* eslint-disable @typescript-eslint/no-require-imports */
// Mock recharts with simple stub components for jsdom.
vi.mock("recharts", () => {
  const React = require("react");
  const stub = (name: string) =>
    function StubComponent({ children }: { children?: React.ReactNode }): React.ReactNode {
      return React.createElement("div", { "data-testid": `recharts-${name}` }, children);
    };
  return {
    ResponsiveContainer: stub("responsive-container"),
    PieChart: stub("pie-chart"),
    Pie: stub("pie"),
    Cell: stub("cell"),
    Legend: stub("legend"),
    Tooltip: stub("tooltip"),
    AreaChart: stub("area-chart"),
    Area: stub("area"),
    BarChart: stub("bar-chart"),
    Bar: stub("bar"),
    XAxis: stub("x-axis"),
    YAxis: stub("y-axis"),
    CartesianGrid: stub("grid"),
  };
});

// Mock api-client functions.
const mockBuildUrl = vi.fn();
const mockBuildHeaders = vi.fn();
const mockParseResponse = vi.fn();
vi.mock("@/lib/api-client", () => ({
  buildUrl: (...args: unknown[]) => mockBuildUrl(...args),
  buildHeaders: (...args: unknown[]) => mockBuildHeaders(...args),
  parseResponse: (...args: unknown[]) => mockParseResponse(...args),
}));

// Mock reporting-api export URL builder.
vi.mock("@/lib/reporting-api", () => ({
  getSupportExportUrl: () => "http://localhost:8080/v1/orgs/default/reports/support/export",
}));

// Mock global fetch.
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import SupportDashboardPage from "./page";
import type { SupportMetrics } from "@/lib/reporting-types";

const MOCK_METRICS: SupportMetrics = {
  status_breakdown: { open: 14, in_progress: 7, resolved: 5, closed: 3 },
  volume_over_time: [
    { date: "2026-03-01", count: 5 },
    { date: "2026-03-02", count: 8 },
  ],
  avg_resolution_hours: 12.5,
  tickets_by_assignee: [
    { user_id: "u1", name: "Alice", count: 10 },
    { user_id: "u2", name: "Bob", count: 5 },
  ],
  tickets_by_priority: { urgent: 3, high: 7, medium: 12, low: 5, none: 2 },
  avg_first_response_hours: 2.3,
  overdue_count: 4,
};

describe("SupportDashboardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockBuildUrl.mockReturnValue("http://localhost:8080/v1/test");
    mockBuildHeaders.mockReturnValue({ Authorization: "Bearer test-token" });
    mockFetch.mockResolvedValue(new Response());
    // Return metrics for the first call, and members for subsequent calls.
    mockParseResponse.mockImplementation(() => Promise.resolve(MOCK_METRICS));
  });

  it("renders all 4 chart sections with mocked metrics", async () => {
    render(<SupportDashboardPage />);

    await waitFor(() => {
      expect(screen.getByTestId("chart-section-status")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-volume")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-assignee")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-priority")).toBeInTheDocument();
    });
  });

  it("shows skeleton while loading", () => {
    // Make fetch hang indefinitely.
    mockParseResponse.mockReturnValue(new Promise(() => {}));
    render(<SupportDashboardPage />);

    const skeletons = screen.getAllByTestId("chart-skeleton");
    expect(skeletons.length).toBe(4);
  });

  it("shows error alert on fetch failure", async () => {
    mockParseResponse.mockRejectedValue(new Error("Network error"));

    render(<SupportDashboardPage />);

    await waitFor(() => {
      expect(screen.getByTestId("error-alert")).toBeInTheDocument();
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });
  });

  it("shows fallback error for non-Error failures", async () => {
    mockParseResponse.mockRejectedValue("boom");
    render(<SupportDashboardPage />);
    await waitFor(() => {
      expect(screen.getByTestId("error-alert")).toHaveTextContent("Failed to load metrics");
    });
  });

  it("re-fetches when date range changes", async () => {
    const user = userEvent.setup();
    render(<SupportDashboardPage />);

    // Wait for initial fetch.
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    // Open date picker and change from date.
    await user.click(screen.getByTestId("date-range-trigger"));
    const fromInput = screen.getByTestId("date-range-from");
    await user.clear(fromInput);
    await user.type(fromInput, "2026-01-01");

    // Fetch should be called again.
    await waitFor(() => {
      expect(mockFetch.mock.calls.length).toBeGreaterThan(1);
    });
  });

  it("renders page title", async () => {
    render(<SupportDashboardPage />);
    expect(screen.getByTestId("support-title")).toHaveTextContent("Support Tickets");
  });

  it("renders 3 KPI metric cards", async () => {
    render(<SupportDashboardPage />);

    await waitFor(() => {
      const cards = screen.getAllByTestId("metric-card");
      expect(cards.length).toBe(3);
    });
  });

  it("renders filter bar with date picker and assignee filter", () => {
    render(<SupportDashboardPage />);
    expect(screen.getByTestId("filter-bar")).toBeInTheDocument();
    expect(screen.getByTestId("date-range-picker")).toBeInTheDocument();
    expect(screen.getByTestId("assignee-filter")).toBeInTheDocument();
  });

  it("renders export button", () => {
    render(<SupportDashboardPage />);
    expect(screen.getByTestId("export-button")).toBeInTheDocument();
  });

  it("displays KPI values when data loads", async () => {
    render(<SupportDashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("12.5 hrs")).toBeInTheDocument();
      expect(screen.getByText("2.3 hrs")).toBeInTheDocument();
    });
  });

  it("displays overdue count with link", async () => {
    render(<SupportDashboardPage />);

    await waitFor(() => {
      const link = screen.getByTestId("metric-card-link");
      expect(link).toHaveAttribute("href", "/crm?status=open&overdue=true");
    });
  });

  it("handles null metrics payload with KPI fallback values", async () => {
    mockParseResponse.mockResolvedValue(null);
    render(<SupportDashboardPage />);
    await waitFor(() => {
      expect(screen.queryAllByTestId("chart-skeleton")).toHaveLength(0);
    });
    expect(screen.getByText("Avg Resolution Time")).toBeInTheDocument();
    expect(screen.getByText("Avg First Response")).toBeInTheDocument();
  });
});
