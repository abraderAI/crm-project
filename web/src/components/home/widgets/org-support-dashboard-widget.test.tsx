import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OrgSupportDashboardWidget } from "./org-support-dashboard-widget";

const mockFetchOrgSupportStats = vi.fn();
vi.mock("@/lib/org-api", () => ({
  fetchOrgSupportStats: (...args: unknown[]) => mockFetchOrgSupportStats(...args),
}));

const MOCK_STATS = {
  open: 5,
  pending: 3,
  resolved: 12,
  total: 20,
};

describe("OrgSupportDashboardWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchOrgSupportStats.mockReturnValue(new Promise(() => {}));
    render(<OrgSupportDashboardWidget token="token" orgId="org-1" />);
    expect(screen.getByTestId("org-support-dashboard-loading")).toBeInTheDocument();
  });

  it("renders stats on successful fetch", async () => {
    mockFetchOrgSupportStats.mockResolvedValue(MOCK_STATS);
    render(<OrgSupportDashboardWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-dashboard-widget")).toBeInTheDocument();
    });

    expect(screen.getByText("20 total tickets")).toBeInTheDocument();
  });

  it("displays correct status counts", async () => {
    mockFetchOrgSupportStats.mockResolvedValue(MOCK_STATS);
    render(<OrgSupportDashboardWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-dashboard-widget")).toBeInTheDocument();
    });

    expect(screen.getByTestId("org-dashboard-open")).toHaveTextContent("5");
    expect(screen.getByTestId("org-dashboard-pending")).toHaveTextContent("3");
    expect(screen.getByTestId("org-dashboard-resolved")).toHaveTextContent("12");
  });

  it("shows error state on fetch failure", async () => {
    mockFetchOrgSupportStats.mockRejectedValue(new Error("Network error"));
    render(<OrgSupportDashboardWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-support-dashboard-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load support statistics.")).toBeInTheDocument();
  });

  it("passes token and orgId to API", async () => {
    mockFetchOrgSupportStats.mockResolvedValue(MOCK_STATS);
    render(<OrgSupportDashboardWidget token="auth-token" orgId="my-org" />);

    await waitFor(() => {
      expect(mockFetchOrgSupportStats).toHaveBeenCalledWith("auth-token", "my-org");
    });
  });
});
