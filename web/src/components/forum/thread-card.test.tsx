import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ThreadCard } from "./thread-card";
import type { ThreadWithAuthor } from "@/lib/api-types";

// Mock ForumVoteButton to avoid Clerk dependency in tests.
vi.mock("./forum-vote-button", () => ({
  ForumVoteButton: ({ initialScore }: { initialScore: number }) => (
    <div data-testid="forum-vote-btn">{initialScore}</div>
  ),
}));

const baseThread: ThreadWithAuthor = {
  id: "t1",
  board_id: "b1",
  title: "Test Thread",
  body: "This is the body text of a test thread.",
  slug: "test-thread",
  metadata: "{}",
  author_id: "user-1",
  is_pinned: false,
  is_locked: false,
  is_hidden: false,
  vote_score: 5,
  created_at: "2026-03-20T10:00:00Z",
  updated_at: "2026-03-20T10:00:00Z",
};

describe("ThreadCard", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders thread title and vote score", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T12:00:00Z"));
    render(<ThreadCard thread={baseThread} />);
    expect(screen.getByText("Test Thread")).toBeDefined();
    expect(screen.getByText("5")).toBeDefined();
  });

  it("renders body preview text", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T12:00:00Z"));
    render(<ThreadCard thread={baseThread} />);
    expect(screen.getByText("This is the body text of a test thread.")).toBeDefined();
  });

  it("shows pin icon for pinned threads", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T12:00:00Z"));
    render(<ThreadCard thread={{ ...baseThread, is_pinned: true }} />);
    expect(screen.getByTestId("pin-icon")).toBeDefined();
  });

  it("links to the thread detail page", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T12:00:00Z"));
    render(<ThreadCard thread={baseThread} />);
    const link = screen.getByTestId("forum-thread-card-t1");
    expect(link.getAttribute("href")).toBe("/forum/test-thread");
  });
});
