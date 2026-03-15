import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import type { SalesMetrics } from "@/lib/reporting-types";

// Mock next/navigation.
const mockPush = vi.fn();
const mockReplace = vi.fn();
const mockSearchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
  useSearchParams: () => mockSearchParams,
}));

// Mock Clerk auth.
const mockGetToken = vi.fn().mockResolvedValue("test-token");
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken, orgId: "org-123" }),
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

// Mock reporting-api (getSalesExportUrl is used on the client).
vi.mock("@/lib/reporting-api", () => ({
  getSalesExportUrl: () => "http://localhost:8080/v1/orgs/org-123/reports/sales/export",
}));

// Mock api-client to avoid actual network.
vi.mock("@/lib/api-client", () => ({
  buildHeaders: () => ({ Authorization: "Bearer test-token" }),
  buildUrl: (path: string) => `http://localhost:8080/v1${path}`,
}));

import SalesPage from "./page";

const MOCK_METRICS: SalesMetrics = {
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

describe("SalesPage", () => {
  it("renders all 6 chart sections with mocked getSalesMetrics", async () => {
    render(<SalesPage />);

    await waitFor(() => {
      expect(screen.getByTestId("chart-section-pipeline-funnel")).toBeInTheDocument();
    });

    expect(screen.getByTestId("chart-section-lead-velocity")).toBeInTheDocument();
    expect(screen.getByTestId("chart-section-leads-by-assignee")).toBeInTheDocument();
    expect(screen.getByTestId("chart-section-score-distribution")).toBeInTheDocument();
    expect(screen.getByTestId("chart-section-stage-conversion")).toBeInTheDocument();
    expect(screen.getByTestId("chart-section-time-in-stage")).toBeInTheDocument();
  });

  it("shows 3 KPI metric cards", async () => {
    render(<SalesPage />);

    await waitFor(() => {
      expect(screen.queryAllByTestId("chart-skeleton")).toHaveLength(0);
    });

    const cards = screen.getAllByTestId("metric-card");
    expect(cards).toHaveLength(3);
    expect(screen.getByText("Win Rate")).toBeInTheDocument();
    expect(screen.getByText("Loss Rate")).toBeInTheDocument();
    expect(screen.getByText("Avg Deal Value")).toBeInTheDocument();
  });

  it("shows skeleton while loading", () => {
    // Make fetch hang forever.
    fetchSpy.mockReturnValue(new Promise(() => {}));
    render(<SalesPage />);

    const skeletons = screen.queryAllByTestId("chart-skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("shows error alert on fetch failure", async () => {
    fetchSpy.mockRejectedValue(new Error("Network error"));
    render(<SalesPage />);

    await waitFor(() => {
      expect(screen.getByTestId("sales-error")).toBeInTheDocument();
    });
  });

  it("renders the page title", async () => {
    render(<SalesPage />);

    expect(screen.getByTestId("sales-page-title")).toHaveTextContent("Sales Pipeline");
  });

  it("renders filter bar", async () => {
    render(<SalesPage />);

    expect(screen.getByTestId("sales-filter-bar")).toBeInTheDocument();
    expect(screen.getByTestId("date-range-picker")).toBeInTheDocument();
    expect(screen.getByTestId("assignee-filter")).toBeInTheDocument();
  });

  it("renders export button", async () => {
    render(<SalesPage />);

    expect(screen.getByTestId("export-button")).toBeInTheDocument();
  });

  it("renders KPI values after data loads", async () => {
    render(<SalesPage />);

    await waitFor(() => {
      expect(screen.queryAllByTestId("chart-skeleton")).toHaveLength(0);
    });

    // Check that KPI values are rendered.
    expect(screen.getByText("35.0%")).toBeInTheDocument();
    expect(screen.getByText("15.0%")).toBeInTheDocument();
    expect(screen.getByText("$25,000")).toBeInTheDocument();
  });
});
