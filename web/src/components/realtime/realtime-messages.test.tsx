import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RealtimeMessages } from "./realtime-messages";
import type { Message } from "@/lib/api-types";

const baseMsg: Message = {
  id: "m-1",
  thread_id: "t-1",
  body: "Hello world",
  author_id: "u-1",
  metadata: "{}",
  type: "comment",
  created_at: "2025-01-15T10:00:00Z",
  updated_at: "2025-01-15T10:00:00Z",
};

const secondMsg: Message = {
  ...baseMsg,
  id: "m-2",
  body: "Second message",
  created_at: "2025-01-15T11:00:00Z",
  updated_at: "2025-01-15T11:00:00Z",
};

describe("RealtimeMessages", () => {
  it("renders the container", () => {
    render(<RealtimeMessages threadId="t-1" initialMessages={[]} />);
    expect(screen.getByTestId("realtime-messages")).toBeInTheDocument();
  });

  it("renders initial messages", () => {
    render(<RealtimeMessages threadId="t-1" initialMessages={[baseMsg]} />);
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
  });

  it("renders empty state when no messages", () => {
    render(<RealtimeMessages threadId="t-1" initialMessages={[]} />);
    expect(screen.getByTestId("message-timeline-empty")).toBeInTheDocument();
  });

  it("renders multiple messages", () => {
    render(<RealtimeMessages threadId="t-1" initialMessages={[baseMsg, secondMsg]} />);
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
    expect(screen.getByTestId("message-item-m-2")).toBeInTheDocument();
  });

  it("calls wsSubscribe on mount with thread channel", () => {
    const wsSubscribe = vi.fn();
    render(<RealtimeMessages threadId="t-1" initialMessages={[]} wsSubscribe={wsSubscribe} />);
    expect(wsSubscribe).toHaveBeenCalledWith("thread:t-1");
  });

  it("calls wsUnsubscribe on unmount", () => {
    const wsUnsubscribe = vi.fn();
    const { unmount } = render(
      <RealtimeMessages threadId="t-1" initialMessages={[]} wsUnsubscribe={wsUnsubscribe} />,
    );
    unmount();
    expect(wsUnsubscribe).toHaveBeenCalledWith("thread:t-1");
  });

  it("appends external new messages without duplicates", () => {
    const newMsg: Message = {
      ...baseMsg,
      id: "m-new",
      body: "New real-time message",
    };

    const { rerender } = render(<RealtimeMessages threadId="t-1" initialMessages={[baseMsg]} />);

    rerender(
      <RealtimeMessages
        threadId="t-1"
        initialMessages={[baseMsg]}
        externalNewMessages={[newMsg]}
      />,
    );

    expect(screen.getByTestId("message-item-m-new")).toBeInTheDocument();
    // Original still there.
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
  });

  it("does not duplicate messages from externalNewMessages", () => {
    const { rerender } = render(<RealtimeMessages threadId="t-1" initialMessages={[baseMsg]} />);

    // Re-send the same message.
    rerender(
      <RealtimeMessages
        threadId="t-1"
        initialMessages={[baseMsg]}
        externalNewMessages={[baseMsg]}
      />,
    );

    // Should only have one instance.
    const items = screen.getAllByTestId("message-item-m-1");
    expect(items).toHaveLength(1);
  });

  it("displays typing indicator when external typing users provided", () => {
    render(
      <RealtimeMessages
        threadId="t-1"
        initialMessages={[]}
        externalTypingUsers={[{ userId: "u-2", userName: "Alice", expiresAt: Date.now() + 5000 }]}
      />,
    );
    expect(screen.getByTestId("typing-indicator")).toBeInTheDocument();
    expect(screen.getByTestId("typing-message")).toHaveTextContent("Alice is typing...");
  });

  it("does not show typing indicator when no one is typing", () => {
    render(<RealtimeMessages threadId="t-1" initialMessages={[]} externalTypingUsers={[]} />);
    expect(screen.queryByTestId("typing-indicator")).not.toBeInTheDocument();
  });

  it("passes onEditMessage to MessageTimeline", () => {
    const onEdit = vi.fn();
    render(
      <RealtimeMessages
        threadId="t-1"
        initialMessages={[baseMsg]}
        currentUserId="u-1"
        onEditMessage={onEdit}
      />,
    );
    // Edit button should exist for the message author.
    expect(screen.getByTestId("message-edit-m-1")).toBeInTheDocument();
  });
});
