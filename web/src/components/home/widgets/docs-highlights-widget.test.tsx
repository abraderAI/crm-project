import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { DocsHighlightsWidget } from "./docs-highlights-widget";

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
    id: "t1",
    title: "Getting Started Guide",
    slug: "getting-started",
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "t2",
    title: "API Reference",
    slug: "api-reference",
    created_at: "2026-01-02T00:00:00Z",
  },
];

describe("DocsHighlightsWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchGlobalThreads.mockReturnValue(new Promise(() => {}));
    render(<DocsHighlightsWidget />);
    expect(screen.getByTestId("docs-highlights-loading")).toBeInTheDocument();
  });

  it("renders thread list on successful fetch", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<DocsHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("docs-highlights-list")).toBeInTheDocument();
    });

    expect(screen.getByText("Getting Started Guide")).toBeInTheDocument();
    expect(screen.getByText("API Reference")).toBeInTheDocument();
  });

  it("renders links pointing to /docs/{slug}", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<DocsHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("docs-highlights-list")).toBeInTheDocument();
    });

    const link = screen.getByText("Getting Started Guide").closest("a");
    expect(link).toHaveAttribute("href", "/docs/getting-started");
  });

  it("shows empty state when no threads", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<DocsHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("docs-highlights-empty")).toBeInTheDocument();
    });

    expect(screen.getByText("No documentation available yet.")).toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    mockFetchGlobalThreads.mockRejectedValue(new Error("Network error"));

    render(<DocsHighlightsWidget />);

    await waitFor(() => {
      expect(screen.getByTestId("docs-highlights-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load documentation.")).toBeInTheDocument();
  });

  it("fetches from global-docs with limit=5", async () => {
    mockFetchGlobalThreads.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<DocsHighlightsWidget />);

    await waitFor(() => {
      expect(mockFetchGlobalThreads).toHaveBeenCalledWith("global-docs", { limit: 5 });
    });
  });
});
