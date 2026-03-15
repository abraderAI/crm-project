import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

import type { AdminSupportMetrics } from "@/lib/reporting-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock next/navigation.
const mockReplace = vi.fn();
const mockSearchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: mockReplace }),
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
  getAdminSupportExportUrl: () => "http://localhost:8080/v1/admin/reports/support/export",
}));

// Mock global fetch.
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

import AdminSupportPage from "./page";

const MOCK_METRICS: AdminSupportMetrics = {
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
  org_breakdown: [
    {
      org_id: "o1",
      org_name: "Acme Corp",
      org_slug: "acme-corp",
      open_count: 10,
      overdue_count: 2,
      avg_resolution_hours: 10.0,
      avg_first_response_hours: 1.5,
      total_in_range: 30,
    },
    {
      org_id: "o2",
      org_name: "Beta Inc",
      org_slug: "beta-inc",
      open_count: 4,
      overdue_count: 2,
      avg_resolution_hours: 15.0,
      avg_first_response_hours: 3.1,
      total_in_range: 12,
    },
  ],
};

describe("AdminSupportPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockBuildUrl.mockReturnValue("http://localhost:8080/v1/admin/reports/support");
    mockBuildHeaders.mockReturnValue({ Authorization: "Bearer test-token" });
    mockFetch.mockResolvedValue(new Response());
    mockParseResponse.mockResolvedValue(MOCK_METRICS);
  });

  it("renders page title", async () => {
    render(<AdminSupportPage />);
    expect(screen.getByTestId("admin-support-title")).toHaveTextContent(
      "Platform Support Overview",
    );
  });

  it("renders KPI cards with mocked data", async () => {
    render(<AdminSupportPage />);

    await waitFor(() => {
      const cards = screen.getAllByTestId("metric-card");
      expect(cards).toHaveLength(3);
    });

    // Check KPI values
    await waitFor(() => {
      expect(screen.getByText("12.5 hrs")).toBeInTheDocument();
    });
  });

  it("renders OrgBreakdownTable with correct row count", async () => {
    render(<AdminSupportPage />);

    await waitFor(() => {
      const rows = screen.getAllByTestId("org-breakdown-row");
      expect(rows).toHaveLength(2);
    });
  });

  it("shows loading skeleton", () => {
    mockParseResponse.mockReturnValue(new Promise(() => {}));
    render(<AdminSupportPage />);

    const skeletons = screen.getAllByTestId("chart-skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("shows error alert on fetch failure", async () => {
    mockParseResponse.mockRejectedValue(new Error("Network error"));
    render(<AdminSupportPage />);

    await waitFor(() => {
      expect(screen.getByTestId("admin-support-error")).toBeInTheDocument();
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });
  });

  it("renders export button", () => {
    render(<AdminSupportPage />);
    expect(screen.getByTestId("export-button")).toBeInTheDocument();
  });

  it("renders date range picker without assignee filter", () => {
    render(<AdminSupportPage />);
    expect(screen.getByTestId("date-range-picker")).toBeInTheDocument();
    expect(screen.queryByTestId("assignee-filter")).not.toBeInTheDocument();
  });

  it("renders chart sections", async () => {
    render(<AdminSupportPage />);

    await waitFor(() => {
      expect(screen.getByTestId("chart-section-status")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-volume")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-priority")).toBeInTheDocument();
    });
  });

  it("renders By Organization heading", async () => {
    render(<AdminSupportPage />);

    await waitFor(() => {
      expect(screen.getByText("By Organization")).toBeInTheDocument();
    });
  });
});
