import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Message } from "@/lib/api-types";
import { MessageTimeline } from "./message-timeline";

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

describe("MessageTimeline", () => {
  it("renders empty state", () => {
    render(<MessageTimeline messages={[]} />);
    expect(screen.getByTestId("message-timeline-empty")).toHaveTextContent("No messages yet.");
  });

  it("renders messages", () => {
    render(<MessageTimeline messages={[baseMsg]} />);
    expect(screen.getByTestId("message-timeline")).toBeInTheDocument();
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
  });

  it("shows message body", () => {
    render(<MessageTimeline messages={[baseMsg]} />);
    expect(screen.getByTestId("message-body-m-1")).toHaveTextContent("Hello world");
  });

  it("shows message type badge for comment", () => {
    render(<MessageTimeline messages={[baseMsg]} />);
    expect(screen.getByTestId("message-type-m-1")).toHaveTextContent("Comment");
  });

  it("shows message type badge for note", () => {
    const note: Message = { ...baseMsg, id: "m-2", type: "note" };
    render(<MessageTimeline messages={[note]} />);
    expect(screen.getByTestId("message-type-m-2")).toHaveTextContent("Note");
  });

  it("shows message type badge for email", () => {
    const email: Message = { ...baseMsg, id: "m-3", type: "email" };
    render(<MessageTimeline messages={[email]} />);
    expect(screen.getByTestId("message-type-m-3")).toHaveTextContent("Email");
  });

  it("shows message type badge for call_log", () => {
    const call: Message = { ...baseMsg, id: "m-4", type: "call_log" };
    render(<MessageTimeline messages={[call]} />);
    expect(screen.getByTestId("message-type-m-4")).toHaveTextContent("Call");
  });

  it("shows message type badge for system", () => {
    const sys: Message = { ...baseMsg, id: "m-5", type: "system" };
    render(<MessageTimeline messages={[sys]} />);
    expect(screen.getByTestId("message-type-m-5")).toHaveTextContent("System");
  });

  it("shows edit button for message owner", () => {
    render(<MessageTimeline messages={[baseMsg]} currentUserId="u-1" onEdit={vi.fn()} />);
    expect(screen.getByTestId("message-edit-m-1")).toBeInTheDocument();
  });

  it("hides edit button for non-owners", () => {
    render(<MessageTimeline messages={[baseMsg]} currentUserId="u-other" onEdit={vi.fn()} />);
    expect(screen.queryByTestId("message-edit-m-1")).not.toBeInTheDocument();
  });

  it("hides edit button when no onEdit", () => {
    render(<MessageTimeline messages={[baseMsg]} currentUserId="u-1" />);
    expect(screen.queryByTestId("message-edit-m-1")).not.toBeInTheDocument();
  });

  it("calls onEdit with message ID", async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    render(<MessageTimeline messages={[baseMsg]} currentUserId="u-1" onEdit={onEdit} />);

    await user.click(screen.getByTestId("message-edit-m-1"));
    expect(onEdit).toHaveBeenCalledWith("m-1");
  });

  it("renders multiple messages", () => {
    const msgs: Message[] = [baseMsg, { ...baseMsg, id: "m-2", body: "Second message" }];
    render(<MessageTimeline messages={msgs} />);
    expect(screen.getByTestId("message-item-m-1")).toBeInTheDocument();
    expect(screen.getByTestId("message-item-m-2")).toBeInTheDocument();
  });
});
