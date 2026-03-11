import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Thread } from "@/lib/api-types";
import { ThreadList, formatDate } from "./thread-list";

const baseThread: Thread = {
  id: "t-1",
  board_id: "b-1",
  title: "First Thread",
  body: "Hello world",
  slug: "first-thread",
  metadata: "{}",
  author_id: "u-1",
  is_pinned: false,
  is_locked: false,
  is_hidden: false,
  vote_score: 5,
  created_at: "2025-01-15T10:00:00Z",
  updated_at: "2025-01-15T10:00:00Z",
  status: "open",
  priority: "high",
};

describe("formatDate", () => {
  it("formats a valid ISO date string", () => {
    expect(formatDate("2025-01-15T10:00:00Z")).toBe("Jan 15, 2025");
  });

  it("returns input for invalid date", () => {
    expect(formatDate("not-a-date")).toBe("not-a-date");
  });
});

describe("ThreadList", () => {
  it("renders empty state when no threads", () => {
    render(<ThreadList threads={[]} basePath="/threads" />);
    expect(screen.getByTestId("thread-list-empty")).toHaveTextContent("No threads yet.");
  });

  it("renders custom empty message", () => {
    render(<ThreadList threads={[]} basePath="/threads" emptyMessage="Start a discussion" />);
    expect(screen.getByTestId("thread-list-empty")).toHaveTextContent("Start a discussion");
  });

  it("renders thread items", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" />);
    expect(screen.getByTestId("thread-list")).toBeInTheDocument();
    expect(screen.getByTestId("thread-item-t-1")).toBeInTheDocument();
    expect(screen.getByText("First Thread")).toBeInTheDocument();
  });

  it("links to thread slug", () => {
    render(<ThreadList threads={[baseThread]} basePath="/boards/b1/threads" />);
    const link = screen.getByTestId("thread-item-t-1");
    expect(link).toHaveAttribute("href", "/boards/b1/threads/first-thread");
  });

  it("shows vote score", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" />);
    expect(screen.getByTestId("thread-votes-t-1")).toHaveTextContent("5");
  });

  it("shows status badge", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" />);
    expect(screen.getByTestId("thread-status-t-1")).toHaveTextContent("open");
  });

  it("shows priority badge", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" />);
    expect(screen.getByTestId("thread-priority-t-1")).toHaveTextContent("high");
  });

  it("shows pin icon for pinned threads", () => {
    const pinned = { ...baseThread, is_pinned: true };
    render(<ThreadList threads={[pinned]} basePath="/threads" />);
    expect(screen.getByTestId("thread-pin-t-1")).toBeInTheDocument();
  });

  it("does not show pin icon for non-pinned threads", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" />);
    expect(screen.queryByTestId("thread-pin-t-1")).not.toBeInTheDocument();
  });

  it("shows lock icon for locked threads", () => {
    const locked = { ...baseThread, is_locked: true };
    render(<ThreadList threads={[locked]} basePath="/threads" />);
    expect(screen.getByTestId("thread-lock-t-1")).toBeInTheDocument();
  });

  it("shows load more button when hasMore", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" hasMore={true} />);
    expect(screen.getByTestId("thread-load-more")).toHaveTextContent("Load more");
  });

  it("calls onLoadMore when clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(
      <ThreadList
        threads={[baseThread]}
        basePath="/threads"
        hasMore={true}
        onLoadMore={onLoadMore}
      />,
    );
    await user.click(screen.getByTestId("thread-load-more"));
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  it("disables load more when loading", () => {
    render(<ThreadList threads={[baseThread]} basePath="/threads" hasMore={true} loading={true} />);
    expect(screen.getByTestId("thread-load-more")).toBeDisabled();
    expect(screen.getByTestId("thread-load-more")).toHaveTextContent("Loading...");
  });

  it("hides status/priority when not set", () => {
    const noMeta = { ...baseThread, status: undefined, priority: undefined };
    render(<ThreadList threads={[noMeta]} basePath="/threads" />);
    expect(screen.queryByTestId("thread-status-t-1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("thread-priority-t-1")).not.toBeInTheDocument();
  });
});
