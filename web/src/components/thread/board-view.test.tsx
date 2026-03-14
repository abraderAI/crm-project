import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { BoardView } from "./board-view";
import type { Thread } from "@/lib/api-types";

function makeThread(overrides: Partial<Thread> & { id: string; title: string }): Thread {
  return {
    board_id: "b1",
    slug: overrides.id,
    body: "",
    metadata: "{}",
    author_id: "u1",
    is_pinned: false,
    is_locked: false,
    is_hidden: false,
    vote_score: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

const threads: Thread[] = [
  makeThread({
    id: "t1",
    title: "Bug report",
    status: "open",
    priority: "high",
    vote_score: 10,
    created_at: "2026-01-03T00:00:00Z",
    updated_at: "2026-01-03T00:00:00Z",
  }),
  makeThread({
    id: "t2",
    title: "Feature request",
    status: "closed",
    priority: "low",
    vote_score: 25,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-05T00:00:00Z",
  }),
  makeThread({
    id: "t3",
    title: "Question about API",
    status: "open",
    priority: "medium",
    vote_score: 5,
    created_at: "2026-01-02T00:00:00Z",
    updated_at: "2026-01-02T00:00:00Z",
  }),
];

describe("BoardView", () => {
  it("renders thread filters and thread list", () => {
    render(<BoardView threads={threads} basePath="/threads" />);
    expect(screen.getByTestId("board-view")).toBeInTheDocument();
    expect(screen.getByTestId("thread-filters")).toBeInTheDocument();
    expect(screen.getByTestId("thread-list")).toBeInTheDocument();
  });

  it("shows all threads by default sorted newest first", () => {
    render(<BoardView threads={threads} basePath="/threads" />);
    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items).toHaveLength(3);
    // Newest first: t1 (Jan 3), t3 (Jan 2), t2 (Jan 1)
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t1");
    expect(items[1]).toHaveAttribute("data-testid", "thread-item-t3");
    expect(items[2]).toHaveAttribute("data-testid", "thread-item-t2");
  });

  it("filters by status", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.selectOptions(screen.getByTestId("thread-status-filter"), "closed");

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t2");
  });

  it("filters by priority", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.selectOptions(screen.getByTestId("thread-priority-filter"), "high");

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t1");
  });

  it("filters by search query", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.type(screen.getByTestId("thread-search-input"), "API");

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t3");
  });

  it("sorts by most votes", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.selectOptions(screen.getByTestId("thread-sort-select"), "most_votes");

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items).toHaveLength(3);
    // Most votes: t2 (25), t1 (10), t3 (5)
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t2");
    expect(items[1]).toHaveAttribute("data-testid", "thread-item-t1");
    expect(items[2]).toHaveAttribute("data-testid", "thread-item-t3");
  });

  it("sorts by oldest", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.selectOptions(screen.getByTestId("thread-sort-select"), "oldest");

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t2");
    expect(items[2]).toHaveAttribute("data-testid", "thread-item-t1");
  });

  it("sorts by recently updated", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.selectOptions(screen.getByTestId("thread-sort-select"), "recently_updated");

    const items = screen.getAllByTestId(/^thread-item-/);
    // Most recently updated: t2 (Jan 5), t1 (Jan 3), t3 (Jan 2)
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t2");
    expect(items[1]).toHaveAttribute("data-testid", "thread-item-t1");
    expect(items[2]).toHaveAttribute("data-testid", "thread-item-t3");
  });

  it("shows empty state when filters match nothing", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.type(screen.getByTestId("thread-search-input"), "nonexistent");

    expect(screen.getByTestId("thread-list-empty")).toBeInTheDocument();
  });

  it("renders VoteSort control", () => {
    render(<BoardView threads={threads} basePath="/threads" />);
    expect(screen.getByTestId("vote-sort")).toBeInTheDocument();
  });

  it("sorts by most votes when VoteSort Top voted is clicked", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.click(screen.getByTestId("sort-option-votes"));

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items).toHaveLength(3);
    // Most votes: t2 (25), t1 (10), t3 (5)
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t2");
    expect(items[1]).toHaveAttribute("data-testid", "thread-item-t1");
    expect(items[2]).toHaveAttribute("data-testid", "thread-item-t3");
  });

  it("sorts by oldest when VoteSort Oldest is clicked", async () => {
    const user = userEvent.setup();
    render(<BoardView threads={threads} basePath="/threads" />);

    await user.click(screen.getByTestId("sort-option-oldest"));

    const items = screen.getAllByTestId(/^thread-item-/);
    expect(items[0]).toHaveAttribute("data-testid", "thread-item-t2");
    expect(items[2]).toHaveAttribute("data-testid", "thread-item-t1");
  });
});
