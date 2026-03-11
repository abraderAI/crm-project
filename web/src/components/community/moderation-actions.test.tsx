import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ModerationActions } from "./moderation-actions";
import type { BoardOption } from "@/lib/api-types";

const defaultProps = {
  threadId: "t1",
  isPinned: false,
  isLocked: false,
  isHidden: false,
  onTogglePin: vi.fn(),
  onToggleLock: vi.fn(),
  onToggleHide: vi.fn(),
  onMove: vi.fn(),
  onMerge: vi.fn(),
};

const boards: BoardOption[] = [
  { id: "b1", name: "General", slug: "general" },
  { id: "b2", name: "Support", slug: "support" },
];

describe("ModerationActions", () => {
  it("renders the heading", () => {
    render(<ModerationActions {...defaultProps} />);
    expect(screen.getByText("Moderation")).toBeInTheDocument();
  });

  it("renders all toggle buttons", () => {
    render(<ModerationActions {...defaultProps} />);
    expect(screen.getByTestId("action-pin")).toBeInTheDocument();
    expect(screen.getByTestId("action-lock")).toBeInTheDocument();
    expect(screen.getByTestId("action-hide")).toBeInTheDocument();
    expect(screen.getByTestId("action-move-toggle")).toBeInTheDocument();
    expect(screen.getByTestId("action-merge-toggle")).toBeInTheDocument();
  });

  it("shows Pin label when not pinned", () => {
    render(<ModerationActions {...defaultProps} isPinned={false} />);
    expect(screen.getByTestId("action-pin")).toHaveTextContent("Pin");
  });

  it("shows Unpin label when pinned", () => {
    render(<ModerationActions {...defaultProps} isPinned={true} />);
    expect(screen.getByTestId("action-pin")).toHaveTextContent("Unpin");
  });

  it("shows Lock/Unlock label", () => {
    const { rerender } = render(<ModerationActions {...defaultProps} isLocked={false} />);
    expect(screen.getByTestId("action-lock")).toHaveTextContent("Lock");

    rerender(<ModerationActions {...defaultProps} isLocked={true} />);
    expect(screen.getByTestId("action-lock")).toHaveTextContent("Unlock");
  });

  it("shows Hide/Unhide label", () => {
    const { rerender } = render(<ModerationActions {...defaultProps} isHidden={false} />);
    expect(screen.getByTestId("action-hide")).toHaveTextContent("Hide");

    rerender(<ModerationActions {...defaultProps} isHidden={true} />);
    expect(screen.getByTestId("action-hide")).toHaveTextContent("Unhide");
  });

  it("calls onTogglePin", async () => {
    const user = userEvent.setup();
    const onTogglePin = vi.fn();
    render(<ModerationActions {...defaultProps} onTogglePin={onTogglePin} />);

    await user.click(screen.getByTestId("action-pin"));
    expect(onTogglePin).toHaveBeenCalledWith("t1");
  });

  it("calls onToggleLock", async () => {
    const user = userEvent.setup();
    const onToggleLock = vi.fn();
    render(<ModerationActions {...defaultProps} onToggleLock={onToggleLock} />);

    await user.click(screen.getByTestId("action-lock"));
    expect(onToggleLock).toHaveBeenCalledWith("t1");
  });

  it("calls onToggleHide", async () => {
    const user = userEvent.setup();
    const onToggleHide = vi.fn();
    render(<ModerationActions {...defaultProps} onToggleHide={onToggleHide} />);

    await user.click(screen.getByTestId("action-hide"));
    expect(onToggleHide).toHaveBeenCalledWith("t1");
  });

  it("shows move panel when Move clicked", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} boards={boards} />);

    expect(screen.queryByTestId("move-panel")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("action-move-toggle"));
    expect(screen.getByTestId("move-panel")).toBeInTheDocument();
  });

  it("shows no-boards message when boards empty", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} boards={[]} />);

    await user.click(screen.getByTestId("action-move-toggle"));
    expect(screen.getByTestId("move-no-boards")).toBeInTheDocument();
  });

  it("performs move action", async () => {
    const user = userEvent.setup();
    const onMove = vi.fn();
    render(<ModerationActions {...defaultProps} onMove={onMove} boards={boards} />);

    await user.click(screen.getByTestId("action-move-toggle"));
    await user.selectOptions(screen.getByTestId("move-board-select"), "b2");
    await user.click(screen.getByTestId("move-confirm-btn"));
    expect(onMove).toHaveBeenCalledWith("t1", "b2");
  });

  it("hides move panel after move", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} onMove={vi.fn()} boards={boards} />);

    await user.click(screen.getByTestId("action-move-toggle"));
    await user.selectOptions(screen.getByTestId("move-board-select"), "b1");
    await user.click(screen.getByTestId("move-confirm-btn"));
    expect(screen.queryByTestId("move-panel")).not.toBeInTheDocument();
  });

  it("shows merge panel when Merge clicked", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} />);

    expect(screen.queryByTestId("merge-panel")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("action-merge-toggle"));
    expect(screen.getByTestId("merge-panel")).toBeInTheDocument();
  });

  it("performs merge action", async () => {
    const user = userEvent.setup();
    const onMerge = vi.fn();
    render(<ModerationActions {...defaultProps} onMerge={onMerge} />);

    await user.click(screen.getByTestId("action-merge-toggle"));
    await user.type(screen.getByTestId("merge-thread-input"), "target-thread-id");
    await user.click(screen.getByTestId("merge-confirm-btn"));
    expect(onMerge).toHaveBeenCalledWith("t1", "target-thread-id");
  });

  it("hides merge panel after merge", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} onMerge={vi.fn()} />);

    await user.click(screen.getByTestId("action-merge-toggle"));
    await user.type(screen.getByTestId("merge-thread-input"), "target");
    await user.click(screen.getByTestId("merge-confirm-btn"));
    expect(screen.queryByTestId("merge-panel")).not.toBeInTheDocument();
  });

  it("closes move panel when merge is opened", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} boards={boards} />);

    await user.click(screen.getByTestId("action-move-toggle"));
    expect(screen.getByTestId("move-panel")).toBeInTheDocument();

    await user.click(screen.getByTestId("action-merge-toggle"));
    expect(screen.queryByTestId("move-panel")).not.toBeInTheDocument();
    expect(screen.getByTestId("merge-panel")).toBeInTheDocument();
  });

  it("disables confirm move when no board selected", async () => {
    const user = userEvent.setup();
    render(<ModerationActions {...defaultProps} boards={boards} />);

    await user.click(screen.getByTestId("action-move-toggle"));
    expect(screen.getByTestId("move-confirm-btn")).toBeDisabled();
  });

  it("applies active styling for pinned state", () => {
    render(<ModerationActions {...defaultProps} isPinned={true} />);
    expect(screen.getByTestId("action-pin")).toHaveClass("border-primary");
  });

  it("applies active styling for locked state", () => {
    render(<ModerationActions {...defaultProps} isLocked={true} />);
    expect(screen.getByTestId("action-lock")).toHaveClass("border-primary");
  });
});
