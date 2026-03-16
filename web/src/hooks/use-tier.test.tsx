import { render, screen, waitFor, act } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { TierProvider, useTier } from "./use-tier";

// Mock tier-api.
const mockFetchTierInfo = vi.fn();
vi.mock("@/lib/tier-api", () => ({
  fetchTierInfo: (...args: unknown[]) => mockFetchTierInfo(...args),
}));

/** Test component that displays tier info. */
function TierDisplay(): React.ReactNode {
  const { tier, subType, deftDepartment, orgId, isLoading } = useTier();
  return (
    <div>
      <span data-testid="tier">{tier}</span>
      <span data-testid="sub-type">{subType ?? "none"}</span>
      <span data-testid="department">{deftDepartment ?? "none"}</span>
      <span data-testid="org-id">{orgId ?? "none"}</span>
      <span data-testid="loading">{isLoading ? "true" : "false"}</span>
    </div>
  );
}

function renderWithProvider(token: string | null): ReturnType<typeof render> {
  return render(
    <TierProvider token={token}>
      <TierDisplay />
    </TierProvider>,
  );
}

describe("useTier", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("throws when used outside TierProvider", () => {
    // Suppress React error boundary console output.
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<TierDisplay />)).toThrow("useTier must be used within a TierProvider");
    spy.mockRestore();
  });

  it("returns tier 1 for anonymous user (null token)", async () => {
    mockFetchTierInfo.mockResolvedValue({ tier: 1, sub_type: null });
    renderWithProvider(null);

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("1");
    expect(screen.getByTestId("sub-type")).toHaveTextContent("none");
  });

  it("returns tier 2 for registered user", async () => {
    mockFetchTierInfo.mockResolvedValue({ tier: 2, sub_type: null });
    renderWithProvider("test-token");

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("2");
  });

  it("returns tier 3 with owner sub-type", async () => {
    mockFetchTierInfo.mockResolvedValue({
      tier: 3,
      sub_type: "owner",
      org_id: "org-1",
    });
    renderWithProvider("test-token");

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("3");
    expect(screen.getByTestId("sub-type")).toHaveTextContent("owner");
    expect(screen.getByTestId("org-id")).toHaveTextContent("org-1");
  });

  it("returns tier 4 with DEFT department", async () => {
    mockFetchTierInfo.mockResolvedValue({
      tier: 4,
      sub_type: null,
      deft_department: "sales",
    });
    renderWithProvider("deft-token");

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("4");
    expect(screen.getByTestId("department")).toHaveTextContent("sales");
  });

  it("returns tier 5 for customer org admin", async () => {
    mockFetchTierInfo.mockResolvedValue({
      tier: 5,
      sub_type: null,
      org_id: "cust-org-1",
    });
    renderWithProvider("admin-token");

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("5");
    expect(screen.getByTestId("org-id")).toHaveTextContent("cust-org-1");
  });

  it("returns tier 6 for platform admin", async () => {
    mockFetchTierInfo.mockResolvedValue({ tier: 6, sub_type: null });
    renderWithProvider("platform-admin-token");

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("6");
  });

  it("defaults to tier 1 on fetch error", async () => {
    mockFetchTierInfo.mockRejectedValue(new Error("Network error"));
    renderWithProvider("bad-token");

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    expect(screen.getByTestId("tier")).toHaveTextContent("1");
    expect(screen.getByTestId("sub-type")).toHaveTextContent("none");
  });

  it("shows loading state initially", () => {
    mockFetchTierInfo.mockReturnValue(new Promise(() => {})); // Never resolves.
    renderWithProvider("token");
    expect(screen.getByTestId("loading")).toHaveTextContent("true");
  });

  it("supports refresh to re-fetch tier", async () => {
    mockFetchTierInfo.mockResolvedValue({ tier: 2, sub_type: null });

    function RefreshTest(): React.ReactNode {
      const { tier, isLoading, refresh } = useTier();
      return (
        <div>
          <span data-testid="tier">{tier}</span>
          <span data-testid="loading">{isLoading ? "true" : "false"}</span>
          <button data-testid="refresh" onClick={refresh}>
            Refresh
          </button>
        </div>
      );
    }

    render(
      <TierProvider token="token">
        <RefreshTest />
      </TierProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
    });

    // Change mock to return tier 3 and refresh.
    mockFetchTierInfo.mockResolvedValue({ tier: 3, sub_type: "owner" });
    await act(async () => {
      screen.getByTestId("refresh").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("tier")).toHaveTextContent("3");
    });
  });
});
