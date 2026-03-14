import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Thread, Message } from "@/lib/api-types";
import { ThreadDetail } from "./thread-detail";

const baseThread: Thread = {
  id: "t-1",
  board_id: "b-1",
  title: "Test Thread",
  body: "Thread body content",
  slug: "test-thread",
  metadata: '{"company":"Acme"}',
  author_id: "u-1",
  is_pinned: false,
  is_locked: false,
  is_hidden: false,
  vote_score: 10,
  status: "open",
  priority: "high",
  created_at: "2025-01-15T10:00:00Z",
  updated_at: "2025-01-15T10:00:00Z",
};

const baseMsg: Message = {
  id: "m-1",
  thread_id: "t-1",
  body: "First reply",
  author_id: "u-2",
  metadata: "{}",
  type: "comment",
  created_at: "2025-01-15T11:00:00Z",
  updated_at: "2025-01-15T11:00:00Z",
};

describe("ThreadDetail", () => {
  it("renders thread detail container", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} />);
    expect(screen.getByTestId("thread-detail")).toBeInTheDocument();
  });

  it("renders thread title", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} />);
    expect(screen.getByTestId("thread-title")).toHaveTextContent("Test Thread");
  });

  it("renders thread body", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} />);
    expect(screen.getByTestId("thread-body")).toHaveTextContent("Thread body content");
  });

  it("does not render body when absent", () => {
    const noBody = { ...baseThread, body: undefined };
    render(<ThreadDetail thread={noBody} messages={[]} />);
    expect(screen.queryByTestId("thread-body")).not.toBeInTheDocument();
  });

  it("renders metadata sidebar", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} />);
    expect(screen.getByTestId("metadata-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-status")).toHaveTextContent("open");
    expect(screen.getByTestId("sidebar-priority")).toHaveTextContent("high");
  });

  it("shows message count", () => {
    render(<ThreadDetail thread={baseThread} messages={[baseMsg]} />);
    expect(screen.getByText("Messages (1)")).toBeInTheDocument();
  });

  it("renders message timeline", () => {
    render(<ThreadDetail thread={baseThread} messages={[baseMsg]} />);
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
  });

  it("shows new message button when onNewMessage provided and not locked", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} onNewMessage={vi.fn()} />);
    expect(screen.getByTestId("thread-new-message-btn")).toBeInTheDocument();
  });

  it("calls onNewMessage when clicked", async () => {
    const user = userEvent.setup();
    const onNew = vi.fn();
    render(<ThreadDetail thread={baseThread} messages={[]} onNewMessage={onNew} />);
    await user.click(screen.getByTestId("thread-new-message-btn"));
    expect(onNew).toHaveBeenCalledOnce();
  });

  it("hides new message button when thread is locked", () => {
    const locked = { ...baseThread, is_locked: true };
    render(<ThreadDetail thread={locked} messages={[]} onNewMessage={vi.fn()} />);
    expect(screen.queryByTestId("thread-new-message-btn")).not.toBeInTheDocument();
  });

  it("shows locked notice for locked threads", () => {
    const locked = { ...baseThread, is_locked: true };
    render(<ThreadDetail thread={locked} messages={[]} />);
    expect(screen.getByTestId("thread-locked-notice")).toBeInTheDocument();
  });

  it("does not show locked notice for unlocked threads", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} />);
    expect(screen.queryByTestId("thread-locked-notice")).not.toBeInTheDocument();
  });

  it("renders editor slot when provided", () => {
    render(
      <ThreadDetail
        thread={baseThread}
        messages={[]}
        editorSlot={<div data-testid="test-editor">Editor</div>}
      />,
    );
    expect(screen.getByTestId("thread-editor-slot")).toBeInTheDocument();
    expect(screen.getByTestId("test-editor")).toBeInTheDocument();
  });

  it("does not render editor slot when not provided", () => {
    render(<ThreadDetail thread={baseThread} messages={[]} />);
    expect(screen.queryByTestId("thread-editor-slot")).not.toBeInTheDocument();
  });

  it("renders messagesSlot when provided instead of default MessageTimeline", () => {
    render(
      <ThreadDetail
        thread={baseThread}
        messages={[baseMsg]}
        messagesSlot={<div data-testid="custom-messages">Realtime</div>}
      />,
    );
    expect(screen.getByTestId("custom-messages")).toBeInTheDocument();
    // Default timeline should NOT render
    expect(screen.queryByTestId("message-item-m-1")).not.toBeInTheDocument();
  });

  it("still shows message count header when messagesSlot is provided", () => {
    render(
      <ThreadDetail
        thread={baseThread}
        messages={[baseMsg]}
        messagesSlot={<div data-testid="custom-messages">Realtime</div>}
      />,
    );
    expect(screen.getByText("Messages (1)")).toBeInTheDocument();
  });
});
