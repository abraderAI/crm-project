import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Thread } from "@/lib/api-types";
import { ThreadList } from "./thread-list";

const mockThreads: Thread[] = [
  {
    id: "t-1",
    board_id: "b-1",
    title: "Bug Report",
    body: "Found a bug",
    slug: "bug-report",
    metadata: "{}",
    author_id: "user-1",
    is_pinned: true,
    is_locked: false,
    is_hidden: false,
    vote_score: 5,
    status: "open",
    priority: "high",
    created_at: "2025-01-15T10:00:00Z",
    updated_at: "2025-01-15T10:00:00Z",
  },
  {
    id: "t-2",
    board_id: "b-1",
    title: "Feature Request",
    slug: "feature-request",
    metadata: "{}",
    author_id: "user-2",
    is_pinned: false,
    is_locked: true,
    is_hidden: false,
    vote_score: 12,
    created_at: "2025-01-14T10:00:00Z",
    updated_at: "2025-01-14T10:00:00Z",
  },
];

describe("ThreadList", () => {
  it("renders thread list container", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByTestId("thread-list")).toBeInTheDocument();
  });

  it("renders threads header", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByText("Threads")).toBeInTheDocument();
  });

  it("renders filters", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByTestId("thread-filters")).toBeInTheDocument();
  });

  it("renders all thread items", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByTestId("thread-item-t-1")).toBeInTheDocument();
    expect(screen.getByTestId("thread-item-t-2")).toBeInTheDocument();
  });

  it("renders thread titles", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByText("Bug Report")).toBeInTheDocument();
    expect(screen.getByText("Feature Request")).toBeInTheDocument();
  });

  it("shows pin icon for pinned threads", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByTestId("pin-t-1")).toBeInTheDocument();
    expect(screen.queryByTestId("pin-t-2")).not.toBeInTheDocument();
  });

  it("shows lock icon for locked threads", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.queryByTestId("lock-t-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("lock-t-2")).toBeInTheDocument();
  });

  it("renders status badge", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByTestId("status-t-1")).toHaveTextContent("open");
  });

  it("renders priority", () => {
    render(<ThreadList threads={mockThreads} />);
    expect(screen.getByTestId("priority-t-1")).toHaveTextContent("high");
  });

  it("shows empty state when no threads", () => {
    render(<ThreadList threads={[]} />);
    expect(screen.getByTestId("threads-empty")).toHaveTextContent("No threads found.");
  });

  it("shows loading state", () => {
    render(<ThreadList threads={[]} loading={true} />);
    expect(screen.getByTestId("threads-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("threads-empty")).not.toBeInTheDocument();
  });

  it("calls onSelect when thread clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<ThreadList threads={mockThreads} onSelect={onSelect} />);

    await user.click(screen.getByTestId("thread-item-t-1"));
    expect(onSelect).toHaveBeenCalledWith("t-1");
  });

  it("calls onSelect on Enter key", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<ThreadList threads={mockThreads} onSelect={onSelect} />);

    const item = screen.getByTestId("thread-item-t-1");
    item.focus();
    await user.keyboard("{Enter}");
    expect(onSelect).toHaveBeenCalledWith("t-1");
  });

  it("renders create button when onCreate provided", () => {
    render(<ThreadList threads={mockThreads} onCreate={vi.fn()} />);
    expect(screen.getByTestId("thread-create-btn")).toBeInTheDocument();
  });

  it("calls onCreate when clicked", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    render(<ThreadList threads={mockThreads} onCreate={onCreate} />);

    await user.click(screen.getByTestId("thread-create-btn"));
    expect(onCreate).toHaveBeenCalledOnce();
  });

  it("renders load more when hasMore", () => {
    render(<ThreadList threads={mockThreads} hasMore={true} onLoadMore={vi.fn()} />);
    expect(screen.getByTestId("threads-load-more")).toBeInTheDocument();
  });

  it("calls onLoadMore when clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(<ThreadList threads={mockThreads} hasMore={true} onLoadMore={onLoadMore} />);

    await user.click(screen.getByTestId("threads-load-more"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("hides load more when loading", () => {
    render(<ThreadList threads={mockThreads} hasMore={true} loading={true} />);
    expect(screen.queryByTestId("threads-load-more")).not.toBeInTheDocument();
  });

  it("uses controlled filter values when provided", () => {
    const onChange = vi.fn();
    render(
      <ThreadList
        threads={mockThreads}
        filterValues={{ sortBy: "vote_score", sortDir: "asc" }}
        onFilterChange={onChange}
      />,
    );
    expect(screen.getByTestId("sort-field")).toHaveValue("vote_score");
  });
});
