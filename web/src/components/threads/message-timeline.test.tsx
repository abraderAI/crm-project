import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Message } from "@/lib/api-types";
import { MessageTimeline } from "./message-timeline";

const mockMessages: Message[] = [
  {
    id: "msg-1",
    thread_id: "t-1",
    body: "Hello world",
    author_id: "user-1",
    metadata: "{}",
    type: "comment",
    created_at: "2025-01-15T10:00:00Z",
    updated_at: "2025-01-15T10:00:00Z",
  },
  {
    id: "msg-2",
    thread_id: "t-1",
    body: "Internal note",
    author_id: "user-2",
    metadata: "{}",
    type: "note",
    created_at: "2025-01-15T11:00:00Z",
    updated_at: "2025-01-15T11:00:00Z",
  },
  {
    id: "msg-3",
    thread_id: "t-1",
    body: "System event",
    author_id: "system",
    metadata: "{}",
    type: "system",
    created_at: "2025-01-15T12:00:00Z",
    updated_at: "2025-01-15T12:00:00Z",
  },
];

describe("MessageTimeline", () => {
  it("renders no messages state when empty", () => {
    render(<MessageTimeline messages={[]} />);
    expect(screen.getByTestId("no-messages")).toHaveTextContent("No messages yet.");
  });

  it("does not render timeline container when empty", () => {
    render(<MessageTimeline messages={[]} />);
    expect(screen.queryByTestId("message-timeline")).not.toBeInTheDocument();
  });

  it("renders all messages", () => {
    render(<MessageTimeline messages={mockMessages} />);
    expect(screen.getByTestId("message-timeline")).toBeInTheDocument();
    expect(screen.getByTestId("message-msg-1")).toBeInTheDocument();
    expect(screen.getByTestId("message-msg-2")).toBeInTheDocument();
    expect(screen.getByTestId("message-msg-3")).toBeInTheDocument();
  });

  it("renders message body", () => {
    render(<MessageTimeline messages={mockMessages} />);
    expect(screen.getByTestId("message-body-msg-1")).toHaveTextContent("Hello world");
  });

  it("renders type badges", () => {
    render(<MessageTimeline messages={mockMessages} />);
    expect(screen.getByTestId("message-type-msg-1")).toHaveTextContent("Comment");
    expect(screen.getByTestId("message-type-msg-2")).toHaveTextContent("Note");
    expect(screen.getByTestId("message-type-msg-3")).toHaveTextContent("System");
  });

  it("renders message icons", () => {
    render(<MessageTimeline messages={mockMessages} />);
    expect(screen.getByTestId("message-icon-msg-1")).toBeInTheDocument();
    expect(screen.getByTestId("message-icon-msg-2")).toBeInTheDocument();
  });

  it("renders author IDs", () => {
    render(<MessageTimeline messages={mockMessages} />);
    expect(screen.getByTestId("message-author-msg-1")).toHaveTextContent("user-1");
  });

  it("renders timestamps", () => {
    render(<MessageTimeline messages={mockMessages} />);
    const time = screen.getByTestId("message-time-msg-1");
    expect(time).toHaveAttribute("datetime", "2025-01-15T10:00:00Z");
    expect(time.textContent).toBeTruthy();
  });

  it("shows edit button for author's own messages", () => {
    render(<MessageTimeline messages={mockMessages} currentUserId="user-1" onEdit={vi.fn()} />);
    expect(screen.getByTestId("message-edit-msg-1")).toBeInTheDocument();
    expect(screen.queryByTestId("message-edit-msg-2")).not.toBeInTheDocument();
  });

  it("hides edit button when no currentUserId", () => {
    render(<MessageTimeline messages={mockMessages} onEdit={vi.fn()} />);
    expect(screen.queryByTestId("message-edit-msg-1")).not.toBeInTheDocument();
  });

  it("hides edit button when no onEdit", () => {
    render(<MessageTimeline messages={mockMessages} currentUserId="user-1" />);
    expect(screen.queryByTestId("message-edit-msg-1")).not.toBeInTheDocument();
  });

  it("calls onEdit with message ID when edit clicked", async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    render(<MessageTimeline messages={mockMessages} currentUserId="user-1" onEdit={onEdit} />);

    await user.click(screen.getByTestId("message-edit-msg-1"));
    expect(onEdit).toHaveBeenCalledWith("msg-1");
  });

  it("renders email and call_log type messages", () => {
    const msgs: Message[] = [
      {
        id: "e1",
        thread_id: "t-1",
        body: "Email content",
        author_id: "u1",
        metadata: "{}",
        type: "email",
        created_at: "2025-01-15T10:00:00Z",
        updated_at: "2025-01-15T10:00:00Z",
      },
      {
        id: "c1",
        thread_id: "t-1",
        body: "Call notes",
        author_id: "u1",
        metadata: "{}",
        type: "call_log",
        created_at: "2025-01-15T11:00:00Z",
        updated_at: "2025-01-15T11:00:00Z",
      },
    ];
    render(<MessageTimeline messages={msgs} />);
    expect(screen.getByTestId("message-type-e1")).toHaveTextContent("Email");
    expect(screen.getByTestId("message-type-c1")).toHaveTextContent("Call Log");
  });
});
