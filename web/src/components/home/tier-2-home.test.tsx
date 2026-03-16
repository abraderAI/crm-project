import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Tier2Home } from "./tier-2-home";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";

// Mock child widgets to avoid their side effects.
vi.mock("./widgets/my-profile-widget", () => ({
  MyProfileWidget: ({ profile }: { profile: unknown }) => (
    <div data-testid="mock-my-profile">{profile ? "Profile loaded" : "No profile"}</div>
  ),
}));
vi.mock("./widgets/my-forum-activity-widget", () => ({
  MyForumActivityWidget: () => <div data-testid="mock-my-forum-activity">Forum Activity</div>,
}));
vi.mock("./widgets/my-support-tickets-widget", () => ({
  MySupportTicketsWidget: () => <div data-testid="mock-my-support-tickets">Support Tickets</div>,
}));
vi.mock("./widgets/upgrade-cta-widget", () => ({
  UpgradeCTAWidget: () => <div data-testid="mock-upgrade-cta">Upgrade CTA</div>,
}));

const DEFAULT_LAYOUT: WidgetConfig[] = [
  { widget_id: WIDGET_IDS.MY_PROFILE, visible: true },
  { widget_id: WIDGET_IDS.MY_FORUM_ACTIVITY, visible: true },
  { widget_id: WIDGET_IDS.MY_SUPPORT_TICKETS, visible: true },
  { widget_id: WIDGET_IDS.UPGRADE_CTA, visible: true },
];

const MOCK_PROFILE = {
  displayName: "Test User",
  email: "test@example.com",
};

describe("Tier2Home", () => {
  it("renders the tier 2 home container", () => {
    render(<Tier2Home layout={DEFAULT_LAYOUT} token="token" profile={MOCK_PROFILE} />);
    expect(screen.getByTestId("tier-2-home")).toBeInTheDocument();
  });

  it("displays home heading", () => {
    render(<Tier2Home layout={DEFAULT_LAYOUT} token="token" profile={MOCK_PROFILE} />);
    expect(screen.getByText("Home")).toBeInTheDocument();
  });

  it("renders all four tier 2 widgets", () => {
    render(<Tier2Home layout={DEFAULT_LAYOUT} token="token" profile={MOCK_PROFILE} />);
    expect(screen.getByTestId("mock-my-profile")).toBeInTheDocument();
    expect(screen.getByTestId("mock-my-forum-activity")).toBeInTheDocument();
    expect(screen.getByTestId("mock-my-support-tickets")).toBeInTheDocument();
    expect(screen.getByTestId("mock-upgrade-cta")).toBeInTheDocument();
  });

  it("renders the home layout grid", () => {
    render(<Tier2Home layout={DEFAULT_LAYOUT} token="token" profile={MOCK_PROFILE} />);
    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
  });

  it("respects visibility settings in layout", () => {
    const layout: WidgetConfig[] = [
      { widget_id: WIDGET_IDS.MY_PROFILE, visible: true },
      { widget_id: WIDGET_IDS.MY_FORUM_ACTIVITY, visible: false },
      { widget_id: WIDGET_IDS.MY_SUPPORT_TICKETS, visible: true },
      { widget_id: WIDGET_IDS.UPGRADE_CTA, visible: false },
    ];
    render(<Tier2Home layout={layout} token="token" profile={MOCK_PROFILE} />);
    expect(screen.getByTestId("mock-my-profile")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-my-forum-activity")).not.toBeInTheDocument();
    expect(screen.getByTestId("mock-my-support-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-upgrade-cta")).not.toBeInTheDocument();
  });

  it("passes profile=null when not available", () => {
    render(<Tier2Home layout={DEFAULT_LAYOUT} token="token" profile={null} />);
    expect(screen.getByTestId("mock-my-profile")).toHaveTextContent("No profile");
  });
});
