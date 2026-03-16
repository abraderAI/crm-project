import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { ForumHighlightsWidget } from "./forum-highlights-widget";

const mockFetchGlobalThreads = vi.fn();
vi.mock("@/lib/global-api", () => ({
  fetchGlobalThreads: (...args: unknown[]) => mockFetchGlobalThreads(...args),
  GLOBAL_SPACES: {
    DOCS: "global-docs",
    FORUM: "global-forum",
    SUPPORT: "global-support",
    LEADS: "global-leads",
  },
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

const MOCK_THREADS = [
  {
    id: "f1",
    title: "How to use widgets?",
    slug: "how-to-use-widgets",
    vote_score: 5,
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "f2",
    title: "Feature request: dark mode",
    slug: "feature-request-dark-mode",
    vote_score: 0,
    created_at: "2026-01-02T00:00:00Z",
  },
];

describe("ForumHighlightsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchGlobalThreads.mockReturnValue(new Promise(() => {}));
    render(<ForumHighlightsWidget />);
    expect(screen.getByTestId("forum-highlights-loading")).toBeInTheDocument();
  });

  it("renders thread list on successful fetch", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("forum-highlights-list")).toBeInTheDocument();
    });

    expect(screen.getByText("How to use widgets?")).toBeInTheDocument();
    expect(screen.getByText("Feature request: dark mode")).toBeInTheDocument();
  });

  it("renders links pointing to /forum/{slug}", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("forum-highlights-list")).toBeInTheDocument();
    });

    const link = screen.getByText("How to use widgets?").closest("a");
    expect(link).toHaveAttribute("href", "/forum/how-to-use-widgets");
  });

  it("shows vote score for threads with positive votes", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("forum-highlights-list")).toBeInTheDocument();
    });

    expect(screen.getByText("▲ 5")).toBeInTheDocument();
  });

  it("does not show vote score for zero-vote threads", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: [{ id: "f2", title: "Zero votes", slug: "zero", vote_score: 0 }],
      page_info: { has_more: false },
    });

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("forum-highlights-list")).toBeInTheDocument();
    });

    expect(screen.queryByText(/▲/)).not.toBeInTheDocument();
  });

  it("shows empty state when no threads", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("forum-highlights-empty")).toBeInTheDocument();
    });
  });

  it("shows error state on fetch failure", async () => {
    mockFetchGlobalThreads.mockRejectedValue(new Error("Network error"));

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("forum-highlights-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load forum threads.")).toBeInTheDocument();
  });

  it("fetches from global-forum with limit=5", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<ForumHighlightsWidget />);

    await waitFor(() => {
      expect(mockFetchGlobalThreads).toHaveBeenCalledWith("global-forum", { limit: 5 });
    });
  });
});
