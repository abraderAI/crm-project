import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OrgOverviewWidget } from "./org-overview-widget";

const mockFetchOrgOverview = vi.fn();
vi.mock("@/lib/org-api", () => ({
  fetchOrgOverview: (...args: unknown[]) => mockFetchOrgOverview(...args),
}));

const MOCK_OVERVIEW = {
  name: "Acme Corp",
  slug: "acme",
  member_count: 12,
  plan_status: "active",
  billing_tier: "enterprise",
};

describe("OrgOverviewWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchOrgOverview.mockReturnValue(new Promise(() => {}));
    render(<OrgOverviewWidget token="token" orgId="org-1" />);
    expect(screen.getByTestId("org-overview-loading")).toBeInTheDocument();
  });

  it("renders org data on successful fetch", async () => {
    mockFetchOrgOverview.mockResolvedValue(MOCK_OVERVIEW);
    render(<OrgOverviewWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-overview-widget")).toBeInTheDocument();
    });

    expect(screen.getByTestId("org-overview-name")).toHaveTextContent("Acme Corp");
    expect(screen.getByTestId("org-overview-member-count")).toHaveTextContent("12 members");
    expect(screen.getByTestId("org-overview-plan")).toHaveTextContent("enterprise");
  });

  it("shows billing status for org owners", async () => {
    mockFetchOrgOverview.mockResolvedValue(MOCK_OVERVIEW);
    render(<OrgOverviewWidget token="token" orgId="org-1" isOwner />);

    await waitFor(() => {
      expect(screen.getByTestId("org-overview-billing-status")).toBeInTheDocument();
    });

    expect(screen.getByTestId("org-overview-billing-status")).toHaveTextContent("active");
  });

  it("hides billing status for non-owners", async () => {
    mockFetchOrgOverview.mockResolvedValue(MOCK_OVERVIEW);
    render(<OrgOverviewWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-overview-widget")).toBeInTheDocument();
    });

    expect(screen.queryByTestId("org-overview-billing-status")).not.toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    mockFetchOrgOverview.mockRejectedValue(new Error("Network error"));
    render(<OrgOverviewWidget token="token" orgId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("org-overview-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load organization data.")).toBeInTheDocument();
  });

  it("passes token and orgId to API", async () => {
    mockFetchOrgOverview.mockResolvedValue(MOCK_OVERVIEW);
    render(<OrgOverviewWidget token="auth-token" orgId="my-org" />);

    await waitFor(() => {
      expect(mockFetchOrgOverview).toHaveBeenCalledWith("auth-token", "my-org");
    });
  });
});
