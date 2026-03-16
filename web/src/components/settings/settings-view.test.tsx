import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock next/navigation.
vi.mock("next/navigation", () => ({
  usePathname: () => "/settings",
  useRouter: () => ({ push: vi.fn() }),
}));

// Mock @clerk/nextjs.
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: vi.fn().mockResolvedValue("mock-token") }),
  UserProfile: () => <div data-testid="clerk-user-profile">Clerk Profile</div>,
}));

// Mock useTier hook.
vi.mock("@/hooks/use-tier", () => ({
  useTier: () => ({
    tier: 3,
    subType: null,
    deftDepartment: null,
    orgId: null,
    isLoading: false,
    refresh: vi.fn(),
  }),
}));

// Mock ApiKeys to isolate SettingsView tests.
vi.mock("./api-keys", () => ({
  ApiKeys: ({ token }: { token: string }) => (
    <div data-testid="api-keys-component">API Keys (token: {token})</div>
  ),
}));

import { SettingsView } from "./settings-view";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("SettingsView", () => {
  it("renders the Account Settings heading", () => {
    render(<SettingsView />);
    expect(screen.getByText("Account Settings")).toBeInTheDocument();
  });

  it("renders the Profile section with Clerk UserProfile", () => {
    render(<SettingsView />);
    expect(screen.getByTestId("settings-profile-section")).toBeInTheDocument();
    expect(screen.getByText("Profile")).toBeInTheDocument();
    expect(screen.getByTestId("clerk-user-profile")).toBeInTheDocument();
  });

  it("renders the Notifications section with link to preferences", () => {
    render(<SettingsView />);
    expect(screen.getByTestId("settings-notifications-section")).toBeInTheDocument();
    expect(screen.getByText("Notifications")).toBeInTheDocument();
    const link = screen.getByTestId("notifications-preferences-link");
    expect(link).toHaveAttribute("href", "/notifications/preferences");
  });

  it("renders the API Keys section", async () => {
    render(<SettingsView />);
    expect(screen.getByTestId("settings-api-keys-section")).toBeInTheDocument();
    // After token resolves, ApiKeys should render.
    await vi.waitFor(() => {
      expect(screen.getByTestId("api-keys-component")).toBeInTheDocument();
    });
  });

  it("renders the Current Tier section with tier badge", () => {
    render(<SettingsView />);
    expect(screen.getByTestId("settings-tier-section")).toBeInTheDocument();
    expect(screen.getByText("Current Tier")).toBeInTheDocument();
    expect(screen.getByTestId("tier-badge")).toHaveTextContent("Tier 3");
    expect(screen.getByTestId("tier-label")).toHaveTextContent("Paying Customer");
  });

  it("renders the settings-page container", () => {
    render(<SettingsView />);
    expect(screen.getByTestId("settings-page")).toBeInTheDocument();
  });
});
