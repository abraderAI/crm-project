import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { TierHomeScreen } from "./tier-home-screen";
import { WIDGET_IDS } from "@/lib/default-layouts";

// Mock hooks.
const mockUseTier = vi.fn();
const mockUseHomeLayout = vi.fn();

vi.mock("@/hooks/use-tier", () => ({
  useTier: () => mockUseTier(),
}));

vi.mock("@/hooks/use-home-layout", () => ({
  useHomeLayout: (...args: unknown[]) => mockUseHomeLayout(...args),
}));

// Mock tier home components.
vi.mock("./tier-1-home", () => ({
  Tier1Home: ({ layout }: { layout: unknown[] }) => (
    <div data-testid="mock-tier-1-home">Tier 1 ({layout.length} widgets)</div>
  ),
}));

vi.mock("./tier-2-home", () => ({
  Tier2Home: ({ layout }: { layout: unknown[] }) => (
    <div data-testid="mock-tier-2-home">Tier 2 ({layout.length} widgets)</div>
  ),
}));

vi.mock("./tier-3-home", () => ({
  Tier3Home: ({ layout, subType }: { layout: unknown[]; subType: string | null }) => (
    <div data-testid="mock-tier-3-home">
      Tier 3 ({layout.length} widgets, sub: {subType ?? "member"})
    </div>
  ),
}));

vi.mock("./tier-5-home", () => ({
  Tier5Home: ({ layout }: { layout: unknown[] }) => (
    <div data-testid="mock-tier-5-home">Tier 5 ({layout.length} widgets)</div>
  ),
}));

describe("TierHomeScreen", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state while tier is resolving", () => {
    mockUseTier.mockReturnValue({
      tier: 1,
      isLoading: true,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [],
      isLoading: true,
    });

    render(<TierHomeScreen token={null} />);
    expect(screen.getByTestId("tier-home-loading")).toBeInTheDocument();
  });

  it("shows loading state while layout is loading", () => {
    mockUseTier.mockReturnValue({
      tier: 2,
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [],
      isLoading: true,
    });

    render(<TierHomeScreen token="token" />);
    expect(screen.getByTestId("tier-home-loading")).toBeInTheDocument();
  });

  it("renders Tier 1 home for anonymous users", () => {
    mockUseTier.mockReturnValue({
      tier: 1,
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [
        { widget_id: WIDGET_IDS.DOCS_HIGHLIGHTS, visible: true },
        { widget_id: WIDGET_IDS.FORUM_HIGHLIGHTS, visible: true },
        { widget_id: WIDGET_IDS.GET_STARTED, visible: true },
      ],
      isLoading: false,
    });

    render(<TierHomeScreen token={null} />);
    expect(screen.getByTestId("tier-home-screen")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-1-home")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-1-home")).toHaveTextContent("3 widgets");
  });

  it("renders Tier 2 home for registered users", () => {
    mockUseTier.mockReturnValue({
      tier: 2,
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [
        { widget_id: WIDGET_IDS.MY_PROFILE, visible: true },
        { widget_id: WIDGET_IDS.MY_FORUM_ACTIVITY, visible: true },
        { widget_id: WIDGET_IDS.MY_SUPPORT_TICKETS, visible: true },
        { widget_id: WIDGET_IDS.UPGRADE_CTA, visible: true },
      ],
      isLoading: false,
    });

    const profile = { displayName: "User", email: "u@test.com" };
    render(<TierHomeScreen token="token" profile={profile} />);
    expect(screen.getByTestId("tier-home-screen")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-2-home")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-2-home")).toHaveTextContent("4 widgets");
  });

  it("renders Tier 3 home for paying customers", () => {
    mockUseTier.mockReturnValue({
      tier: 3,
      subType: null,
      orgId: "org-1",
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [
        { widget_id: WIDGET_IDS.ORG_OVERVIEW, visible: true },
        { widget_id: WIDGET_IDS.ORG_SUPPORT_TICKETS, visible: true },
      ],
      isLoading: false,
    });

    render(<TierHomeScreen token="token" />);
    expect(screen.getByTestId("tier-home-screen")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-3-home")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-3-home")).toHaveTextContent("2 widgets");
    expect(screen.getByTestId("mock-tier-3-home")).toHaveTextContent("sub: member");
  });

  it("renders Tier 3 owner variant", () => {
    mockUseTier.mockReturnValue({
      tier: 3,
      subType: "owner",
      orgId: "org-1",
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [
        { widget_id: WIDGET_IDS.ORG_SUPPORT_DASHBOARD, visible: true },
        { widget_id: WIDGET_IDS.BILLING_STATUS, visible: true },
      ],
      isLoading: false,
    });

    render(<TierHomeScreen token="token" />);
    expect(screen.getByTestId("mock-tier-3-home")).toHaveTextContent("sub: owner");
  });

  it("renders Tier 5 home for org admins", () => {
    mockUseTier.mockReturnValue({
      tier: 5,
      subType: null,
      orgId: "org-1",
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [
        { widget_id: WIDGET_IDS.ORG_ACCESS_CONTROL, visible: true },
        { widget_id: WIDGET_IDS.ORG_RBAC_EDITOR, visible: true },
        { widget_id: WIDGET_IDS.ORG_SUPPORT_DASHBOARD, visible: true },
        { widget_id: WIDGET_IDS.BILLING_STATUS, visible: true },
      ],
      isLoading: false,
    });

    render(<TierHomeScreen token="token" />);
    expect(screen.getByTestId("tier-home-screen")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-5-home")).toBeInTheDocument();
    expect(screen.getByTestId("mock-tier-5-home")).toHaveTextContent("4 widgets");
  });

  it("renders placeholder for tier 4", () => {
    mockUseTier.mockReturnValue({
      tier: 4,
      subType: null,
      orgId: null,
      isLoading: false,
      deftDepartment: "sales",
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [],
      isLoading: false,
    });

    render(<TierHomeScreen token="token" />);
    expect(screen.getByTestId("tier-home-placeholder")).toBeInTheDocument();
    expect(screen.getByText(/Tier 4 home screen/)).toBeInTheDocument();
  });

  it("renders placeholder for tier 6", () => {
    mockUseTier.mockReturnValue({
      tier: 6,
      subType: null,
      orgId: null,
      isLoading: false,
      deftDepartment: null,
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [],
      isLoading: false,
    });

    render(<TierHomeScreen token="token" />);
    expect(screen.getByTestId("tier-home-placeholder")).toBeInTheDocument();
    expect(screen.getByText(/Tier 6 home screen/)).toBeInTheDocument();
  });

  it("passes tier and department to useHomeLayout", () => {
    mockUseTier.mockReturnValue({
      tier: 4,
      isLoading: false,
      deftDepartment: "sales",
    });
    mockUseHomeLayout.mockReturnValue({
      layout: [],
      isLoading: false,
    });

    render(<TierHomeScreen token="token" />);

    expect(mockUseHomeLayout).toHaveBeenCalledWith(4, "token", "sales");
  });
});
