import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

import type { AdminSalesMetrics } from "@/lib/reporting-types";

// Mock next/navigation.
const mockReplace = vi.fn();
const mockSearchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: mockReplace }),
  useSearchParams: () => mockSearchParams,
}));

// Mock Clerk auth.
const mockGetToken = vi.fn().mockResolvedValue("test-token");
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock recharts for jsdom.
vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="responsive-container">{children}</div>
  ),
  BarChart: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="recharts-bar-chart">{children}</div>
  ),
  AreaChart: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="recharts-area-chart">{children}</div>
  ),
  Bar: ({ children }: { children?: React.ReactNode }) => (
    <div data-testid="recharts-bar">{children}</div>
  ),
  Area: () => <div data-testid="recharts-area" />,
  Cell: (props: { fill?: string; "data-testid"?: string }) => (
    <span data-testid={props["data-testid"]} fill={props.fill} />
  ),
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  LabelList: () => null,
}));

// Mock reporting-api.
vi.mock("@/lib/reporting-api", () => ({
  getAdminSalesExportUrl: () => "http://localhost:8080/v1/admin/reports/sales/export",
}));

// Mock api-client.
vi.mock("@/lib/api-client", () => ({
  buildHeaders: () => ({ Authorization: "Bearer test-token" }),
  buildUrl: (path: string) => `http://localhost:8080/v1${path}`,
}));

import AdminSalesPage from "./page";

const MOCK_METRICS: AdminSalesMetrics = {
  pipeline_funnel: [
    { stage: "new_lead", count: 10 },
    { stage: "closed_won", count: 3 },
  ],
  lead_velocity: [
    { date: "2026-03-01", count: 5 },
    { date: "2026-03-02", count: 8 },
  ],
  win_rate: 0.35,
  loss_rate: 0.15,
  avg_deal_value: 25000,
  leads_by_assignee: [
    { user_id: "u1", name: "Alice", count: 12 },
    { user_id: "u2", name: "Bob", count: 8 },
  ],
  score_distribution: [
    { range: "0-20", count: 5 },
    { range: "20-40", count: 10 },
    { range: "40-60", count: 15 },
    { range: "60-80", count: 8 },
    { range: "80-100", count: 3 },
  ],
  stage_conversion_rates: [{ from_stage: "new_lead", to_stage: "contacted", rate: 0.75 }],
  avg_time_in_stage: [
    { stage: "new_lead", avg_hours: 24 },
    { stage: "contacted", avg_hours: 48 },
  ],
  org_breakdown: [
    {
      org_id: "o1",
      org_name: "Acme Corp",
      org_slug: "acme-corp",
      total_leads: 35,
      win_rate: 0.4,
      avg_deal_value: 30000,
      open_pipeline_count: 8,
    },
    {
      org_id: "o2",
      org_name: "Beta Inc",
      org_slug: "beta-inc",
      total_leads: 15,
      win_rate: 0.25,
      avg_deal_value: 20000,
      open_pipeline_count: 4,
    },
  ],
};

let fetchSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  vi.clearAllMocks();
  fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(MOCK_METRICS), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
});

afterEach(() => {
  fetchSpy.mockRestore();
});

describe("AdminSalesPage", () => {
  it("renders page title", () => {
    render(<AdminSalesPage />);
    expect(screen.getByTestId("admin-sales-title")).toHaveTextContent("Platform Sales Overview");
  });

  it("renders 3 KPI metric cards with mocked data", async () => {
    render(<AdminSalesPage />);

    await waitFor(() => {
      expect(screen.queryAllByTestId("chart-skeleton")).toHaveLength(0);
    });

    const cards = screen.getAllByTestId("metric-card");
    expect(cards).toHaveLength(3);
    // Use getAllByText since "Total Leads" also appears in the table header.
    expect(screen.getAllByText("Total Leads").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Platform Win Rate")).toBeInTheDocument();
    // Use getAllByText since "Avg Deal Value" also appears in the table header.
    expect(screen.getAllByText("Avg Deal Value").length).toBeGreaterThanOrEqual(1);
  });

  it("renders KPI values after data loads", async () => {
    render(<AdminSalesPage />);

    await waitFor(() => {
      expect(screen.queryAllByTestId("chart-skeleton")).toHaveLength(0);
    });

    // Total leads = 10 + 3 = 13
    expect(screen.getByText("13")).toBeInTheDocument();
    expect(screen.getByText("35.0%")).toBeInTheDocument();
    expect(screen.getByText("$25,000")).toBeInTheDocument();
  });

  it("renders OrgBreakdownTable with correct row count", async () => {
    render(<AdminSalesPage />);

    await waitFor(() => {
      const rows = screen.getAllByTestId("org-breakdown-row");
      expect(rows).toHaveLength(2);
    });
  });

  it("shows loading skeleton", () => {
    fetchSpy.mockReturnValue(new Promise(() => {}));
    render(<AdminSalesPage />);

    const skeletons = screen.queryAllByTestId("chart-skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("shows error alert on fetch failure", async () => {
    fetchSpy.mockRejectedValue(new Error("Network error"));
    render(<AdminSalesPage />);

    await waitFor(() => {
      expect(screen.getByTestId("admin-sales-error")).toBeInTheDocument();
    });
  });

  it("renders export button", () => {
    render(<AdminSalesPage />);
    expect(screen.getByTestId("export-button")).toBeInTheDocument();
  });

  it("renders date range picker without assignee filter", () => {
    render(<AdminSalesPage />);
    expect(screen.getByTestId("date-range-picker")).toBeInTheDocument();
    expect(screen.queryByTestId("assignee-filter")).not.toBeInTheDocument();
  });

  it("renders chart sections", async () => {
    render(<AdminSalesPage />);

    await waitFor(() => {
      expect(screen.getByTestId("chart-section-pipeline-funnel")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-lead-velocity")).toBeInTheDocument();
      expect(screen.getByTestId("chart-section-score-distribution")).toBeInTheDocument();
    });
  });

  it("renders By Organization heading", async () => {
    render(<AdminSalesPage />);

    await waitFor(() => {
      expect(screen.getByText("By Organization")).toBeInTheDocument();
    });
  });
});
