import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Thread, Message } from "@/lib/api-types";
import { ThreadDetail } from "./thread-detail";

const mockThread: Thread = {
  id: "t-1",
  board_id: "b-1",
  title: "Bug Report",
  body: "Found a critical bug in auth module",
  slug: "bug-report",
  metadata: '{"source":"email","severity":"critical"}',
  author_id: "user-1",
  is_pinned: true,
  is_locked: false,
  is_hidden: false,
  vote_score: 5,
  status: "open",
  priority: "high",
  stage: "triage",
  assigned_to: "user-3",
  created_at: "2025-01-15T10:00:00Z",
  updated_at: "2025-01-15T10:00:00Z",
};

const mockMessages: Message[] = [
  {
    id: "msg-1",
    thread_id: "t-1",
    body: "First message",
    author_id: "user-1",
    metadata: "{}",
    type: "comment",
    created_at: "2025-01-15T10:00:00Z",
    updated_at: "2025-01-15T10:00:00Z",
  },
];

describe("ThreadDetail", () => {
  it("renders detail container", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("thread-detail")).toBeInTheDocument();
  });

  it("renders thread title", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByText("Bug Report")).toBeInTheDocument();
  });

  it("renders thread body", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("thread-body")).toHaveTextContent("Found a critical bug");
  });

  it("does not render body when not present", () => {
    const threadNoBody = { ...mockThread, body: undefined };
    render(<ThreadDetail thread={threadNoBody} messages={mockMessages} />);
    expect(screen.queryByTestId("thread-body")).not.toBeInTheDocument();
  });

  it("renders pin icon when pinned", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("thread-pin")).toBeInTheDocument();
  });

  it("renders lock icon when locked", () => {
    const locked = { ...mockThread, is_locked: true };
    render(<ThreadDetail thread={locked} messages={mockMessages} />);
    expect(screen.getByTestId("thread-lock")).toBeInTheDocument();
  });

  it("does not render lock when not locked", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.queryByTestId("thread-lock")).not.toBeInTheDocument();
  });

  it("renders sidebar with status", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("sidebar-status")).toHaveTextContent("open");
  });

  it("renders sidebar with priority", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("sidebar-priority")).toHaveTextContent("high");
  });

  it("renders sidebar with stage", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("sidebar-stage")).toHaveTextContent("triage");
  });

  it("renders sidebar with assigned_to", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("sidebar-assigned")).toHaveTextContent("user-3");
  });

  it("renders sidebar with vote score", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("sidebar-votes")).toHaveTextContent("5");
  });

  it("renders sidebar with author", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("sidebar-author")).toHaveTextContent("user-1");
  });

  it("renders custom metadata in sidebar", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("meta-source")).toHaveTextContent("email");
    expect(screen.getByTestId("meta-severity")).toHaveTextContent("critical");
  });

  it("renders message timeline", () => {
    render(<ThreadDetail thread={mockThread} messages={mockMessages} />);
    expect(screen.getByTestId("message-timeline")).toBeInTheDocument();
  });

  it("renders no messages when empty", () => {
    render(<ThreadDetail thread={mockThread} messages={[]} />);
    expect(screen.getByTestId("no-messages")).toBeInTheDocument();
  });

  it("passes currentUserId to message timeline", () => {
    render(
      <ThreadDetail
        thread={mockThread}
        messages={mockMessages}
        currentUserId="user-1"
        onEditMessage={vi.fn()}
      />,
    );
    expect(screen.getByTestId("message-edit-msg-1")).toBeInTheDocument();
  });

  it("handles invalid metadata JSON gracefully", () => {
    const badMeta = { ...mockThread, metadata: "invalid" };
    render(<ThreadDetail thread={badMeta} messages={mockMessages} />);
    expect(screen.getByTestId("thread-sidebar")).toBeInTheDocument();
  });

  it("handles array metadata gracefully", () => {
    const arrMeta = { ...mockThread, metadata: "[1,2]" };
    render(<ThreadDetail thread={arrMeta} messages={mockMessages} />);
    expect(screen.getByTestId("thread-sidebar")).toBeInTheDocument();
  });

  it("hides optional sidebar fields when not present", () => {
    const minimal = {
      ...mockThread,
      status: undefined,
      priority: undefined,
      stage: undefined,
      assigned_to: undefined,
      metadata: "{}",
    };
    render(<ThreadDetail thread={minimal} messages={mockMessages} />);
    expect(screen.queryByTestId("sidebar-status")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-priority")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-stage")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-assigned")).not.toBeInTheDocument();
  });
});
