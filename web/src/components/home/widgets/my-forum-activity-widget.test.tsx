import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { MyForumActivityWidget } from "./my-forum-activity-widget";

const mockFetchUserForumActivity = vi.fn();
vi.mock("@/lib/global-api", () => ({
  fetchUserForumActivity: (...args: unknown[]) => mockFetchUserForumActivity(...args),
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
  { id: "t1", title: "My first post", slug: "my-first-post" },
  { id: "t2", title: "Reply to a question", slug: "reply-to-question" },
];

describe("MyForumActivityWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton initially", () => {
    mockFetchUserForumActivity.mockReturnValue(new Promise(() => {}));
    render(<MyForumActivityWidget token="token" />);
    expect(screen.getByTestId("my-forum-activity-loading")).toBeInTheDocument();
  });

  it("renders thread list on successful fetch", async () => {
    mockFetchUserForumActivity.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<MyForumActivityWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-forum-activity-list")).toBeInTheDocument();
    });

    expect(screen.getByText("My first post")).toBeInTheDocument();
    expect(screen.getByText("Reply to a question")).toBeInTheDocument();
  });

  it("renders links to forum thread detail", async () => {
    mockFetchUserForumActivity.mockResolvedValue({
      data: MOCK_THREADS,
      page_info: { has_more: false },
    });

    render(<MyForumActivityWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-forum-activity-list")).toBeInTheDocument();
    });

    const link = screen.getByText("My first post").closest("a");
    expect(link).toHaveAttribute("href", "/forum/my-first-post");
  });

  it("shows empty state with join link when no activity", async () => {
    mockFetchUserForumActivity.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<MyForumActivityWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-forum-activity-empty")).toBeInTheDocument();
    });

    const joinLink = screen.getByText("Join the conversation");
    expect(joinLink.closest("a")).toHaveAttribute("href", "/forum");
  });

  it("shows error state on fetch failure", async () => {
    mockFetchUserForumActivity.mockRejectedValue(new Error("Network error"));

    render(<MyForumActivityWidget token="token" />);

    await waitFor(() => {
      expect(screen.getByTestId("my-forum-activity-error")).toBeInTheDocument();
    });

    expect(screen.getByText("Failed to load your forum activity.")).toBeInTheDocument();
  });

  it("passes token and limit to API", async () => {
    mockFetchUserForumActivity.mockResolvedValue({
      data: [],
      page_info: { has_more: false },
    });

    render(<MyForumActivityWidget token="my-auth-token" />);

    await waitFor(() => {
      expect(mockFetchUserForumActivity).toHaveBeenCalledWith("my-auth-token", { limit: 5 });
    });
  });
});
