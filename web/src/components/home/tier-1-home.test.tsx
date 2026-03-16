import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Tier1Home } from "./tier-1-home";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";

// Mock child widgets to avoid their side effects in this composition test.
vi.mock("./widgets/docs-highlights-widget", () => ({
  DocsHighlightsWidget: () => <div data-testid="mock-docs-highlights">Docs</div>,
}));
vi.mock("./widgets/forum-highlights-widget", () => ({
  ForumHighlightsWidget: () => <div data-testid="mock-forum-highlights">Forum</div>,
}));
vi.mock("./widgets/get-started-widget", () => ({
  GetStartedWidget: () => <div data-testid="mock-get-started">GetStarted</div>,
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode;
    href: string;
    className?: string;
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

const DEFAULT_LAYOUT: WidgetConfig[] = [
  { widget_id: WIDGET_IDS.DOCS_HIGHLIGHTS, visible: true },
  { widget_id: WIDGET_IDS.FORUM_HIGHLIGHTS, visible: true },
  { widget_id: WIDGET_IDS.GET_STARTED, visible: true },
];

describe("Tier1Home", () => {
  it("renders the tier 1 home container", () => {
    render(<Tier1Home layout={DEFAULT_LAYOUT} />);
    expect(screen.getByTestId("tier-1-home")).toBeInTheDocument();
  });

  it("displays welcome heading", () => {
    render(<Tier1Home layout={DEFAULT_LAYOUT} />);
    expect(screen.getByText("Welcome to DEFT Evolution")).toBeInTheDocument();
  });

  it("renders all three tier 1 widgets", () => {
    render(<Tier1Home layout={DEFAULT_LAYOUT} />);
    expect(screen.getByTestId("mock-docs-highlights")).toBeInTheDocument();
    expect(screen.getByTestId("mock-forum-highlights")).toBeInTheDocument();
    expect(screen.getByTestId("mock-get-started")).toBeInTheDocument();
  });

  it("renders the home layout grid", () => {
    render(<Tier1Home layout={DEFAULT_LAYOUT} />);
    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
  });

  it("respects visibility settings in layout", () => {
    const layout: WidgetConfig[] = [
      { widget_id: WIDGET_IDS.DOCS_HIGHLIGHTS, visible: true },
      { widget_id: WIDGET_IDS.FORUM_HIGHLIGHTS, visible: false },
      { widget_id: WIDGET_IDS.GET_STARTED, visible: true },
    ];
    render(<Tier1Home layout={layout} />);
    expect(screen.getByTestId("mock-docs-highlights")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-forum-highlights")).not.toBeInTheDocument();
    expect(screen.getByTestId("mock-get-started")).toBeInTheDocument();
  });
});
