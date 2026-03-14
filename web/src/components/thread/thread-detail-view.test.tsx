import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { Thread, Message } from "@/lib/api-types";

// Mock Clerk auth.
const mockGetToken = vi.fn();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken, userId: "user-1" }),
}));

// Mock next/navigation.
const mockRefresh = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: mockRefresh }),
}));

// Mock entity-api.
const mockToggleVote = vi.fn();
const mockCreateMessage = vi.fn();
const mockFetchThreadRevisions = vi.fn();
const mockFetchThreadUploads = vi.fn();
const mockUploadFile = vi.fn();
vi.mock("@/lib/entity-api", () => ({
  toggleVote: (...args: unknown[]) => mockToggleVote(...args),
  createMessage: (...args: unknown[]) => mockCreateMessage(...args),
  fetchThreadRevisions: (...args: unknown[]) => mockFetchThreadRevisions(...args),
  fetchThreadUploads: (...args: unknown[]) => mockFetchThreadUploads(...args),
  uploadFile: (...args: unknown[]) => mockUploadFile(...args),
}));

import { ThreadDetailView } from "./thread-detail-view";

const thread: Thread = {
  id: "t1",
  board_id: "b1",
  title: "Test Thread",
  body: "Thread body",
  slug: "test-thread",
  metadata: "{}",
  author_id: "user-1",
  is_pinned: false,
  is_locked: false,
  is_hidden: false,
  vote_score: 5,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const messages: Message[] = [
  {
    id: "m1",
    thread_id: "t1",
    body: "First message",
    author_id: "user-2",
    metadata: "{}",
    type: "comment",
    created_at: "2026-01-01T01:00:00Z",
    updated_at: "2026-01-01T01:00:00Z",
  },
];

const defaultProps = {
  thread,
  messages,
  orgSlug: "acme",
  spaceSlug: "community",
  boardSlug: "general",
  threadSlug: "test-thread",
};

describe("ThreadDetailView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    mockFetchThreadUploads.mockResolvedValue({ data: [], page_info: { has_more: false } });
    mockFetchThreadRevisions.mockResolvedValue({ data: [], page_info: { has_more: false } });
    mockToggleVote.mockResolvedValue({ id: "v1", thread_id: "t1", user_id: "user-1", weight: 1 });
  });

  it("renders VoteButton with correct initial state when hasVoted is true", () => {
    render(<ThreadDetailView {...defaultProps} hasVoted />);

    const voteButton = screen.getByTestId("vote-button");
    expect(voteButton).toBeInTheDocument();
    expect(screen.getByTestId("vote-score")).toHaveTextContent("5");
  });

  it("renders VoteButton with hasVoted false by default", () => {
    render(<ThreadDetailView {...defaultProps} />);

    const voteButton = screen.getByTestId("vote-button");
    expect(voteButton).toBeInTheDocument();
  });

  it("calls toggleVote via entity-api when VoteButton is clicked", async () => {
    const user = userEvent.setup();
    render(<ThreadDetailView {...defaultProps} hasVoted={false} />);

    await user.click(screen.getByTestId("vote-button"));

    await waitFor(() => {
      expect(mockToggleVote).toHaveBeenCalledWith(
        "test-token",
        "acme",
        "community",
        "general",
        "test-thread",
      );
    });
  });

  it("renders ThreadDetail with thread data", () => {
    render(<ThreadDetailView {...defaultProps} />);

    expect(screen.getByTestId("thread-detail")).toBeInTheDocument();
    expect(screen.getByTestId("thread-title")).toHaveTextContent("Test Thread");
  });

  it("renders revision toggle button", () => {
    render(<ThreadDetailView {...defaultProps} />);

    expect(screen.getByTestId("revision-toggle")).toBeInTheDocument();
  });
});
