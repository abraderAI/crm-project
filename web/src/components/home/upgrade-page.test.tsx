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
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

// Mock upgrade API.
const mockUpgradeToCustomer = vi.fn();
vi.mock("@/lib/upgrade-api", () => ({
  upgradeToCustomer: (...args: unknown[]) => mockUpgradeToCustomer(...args),
}));

// Mock useTier hook.
const mockRefresh = vi.fn();
vi.mock("@/hooks/use-tier", () => ({
  useTier: () => ({ tier: 2, isLoading: false, refresh: mockRefresh }),
}));

import { UpgradePage } from "./upgrade-page";

describe("UpgradePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockUpgradeToCustomer.mockResolvedValue({ tier: 3, message: "Upgraded" });
  });

  it("renders two plan comparison cards", () => {
    render(<UpgradePage />);
    expect(screen.getByTestId("plan-card-developer")).toBeInTheDocument();
    expect(screen.getByTestId("plan-card-customer")).toBeInTheDocument();
  });

  it("displays Developer plan as current", () => {
    render(<UpgradePage />);
    const devCard = screen.getByTestId("plan-card-developer");
    expect(devCard).toHaveTextContent("Current Plan");
  });

  it("displays Customer plan as target", () => {
    render(<UpgradePage />);
    const custCard = screen.getByTestId("plan-card-customer");
    expect(custCard).toHaveTextContent("Customer");
  });

  it("renders feature checklists for both tiers", () => {
    render(<UpgradePage />);
    expect(screen.getByTestId("developer-features")).toBeInTheDocument();
    expect(screen.getByTestId("customer-features")).toBeInTheDocument();
  });

  it("shows Activate Trial button", () => {
    render(<UpgradePage />);
    expect(screen.getByTestId("upgrade-button")).toBeInTheDocument();
    expect(screen.getByTestId("upgrade-button")).toHaveTextContent("Activate Trial");
  });

  it("calls upgrade API on button click", async () => {
    const user = userEvent.setup();
    render(<UpgradePage />);

    await user.click(screen.getByTestId("upgrade-button"));

    await waitFor(() => {
      expect(mockUpgradeToCustomer).toHaveBeenCalledWith("test-token");
    });
  });

  it("refreshes tier cache on successful upgrade", async () => {
    const user = userEvent.setup();
    render(<UpgradePage />);

    await user.click(screen.getByTestId("upgrade-button"));

    await waitFor(() => {
      expect(mockRefresh).toHaveBeenCalled();
    });
  });

  it("redirects to home on successful upgrade", async () => {
    const user = userEvent.setup();
    render(<UpgradePage />);

    await user.click(screen.getByTestId("upgrade-button"));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/");
    });
  });

  it("shows error message on upgrade failure", async () => {
    mockUpgradeToCustomer.mockRejectedValue(new Error("Upgrade failed"));
    const user = userEvent.setup();
    render(<UpgradePage />);

    await user.click(screen.getByTestId("upgrade-button"));

    await waitFor(() => {
      expect(screen.getByTestId("upgrade-error")).toBeInTheDocument();
    });
  });

  it("disables button while upgrade is in progress", async () => {
    // Make the upgrade take some time.
    mockUpgradeToCustomer.mockReturnValue(new Promise(() => {}));
    const user = userEvent.setup();
    render(<UpgradePage />);

    await user.click(screen.getByTestId("upgrade-button"));

    expect(screen.getByTestId("upgrade-button")).toBeDisabled();
  });

  it("does not call upgrade when token is null", async () => {
    mockGetToken.mockResolvedValue(null);
    const user = userEvent.setup();
    render(<UpgradePage />);

    await user.click(screen.getByTestId("upgrade-button"));

    await waitFor(() => {
      expect(mockUpgradeToCustomer).not.toHaveBeenCalled();
    });
  });
});
