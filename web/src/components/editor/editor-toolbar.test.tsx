import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Bold, Italic } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { EditorToolbar, buildDefaultActions, type ToolbarAction } from "./editor-toolbar";

const mockActions: ToolbarAction[] = [
  { id: "bold", label: "Bold", icon: Bold, isActive: false },
  { id: "italic", label: "Italic", icon: Italic, isActive: true },
];

describe("buildDefaultActions", () => {
  it("returns 7 default toolbar actions", () => {
    const actions = buildDefaultActions({});
    expect(actions).toHaveLength(7);
  });

  it("sets isActive based on active states", () => {
    const actions = buildDefaultActions({ bold: true, italic: false });
    expect(actions.find((a) => a.id === "bold")?.isActive).toBe(true);
    expect(actions.find((a) => a.id === "italic")?.isActive).toBe(false);
  });

  it("returns undefined for missing active states", () => {
    const actions = buildDefaultActions({});
    expect(actions.find((a) => a.id === "bold")?.isActive).toBeUndefined();
  });
});

describe("EditorToolbar", () => {
  it("renders toolbar container", () => {
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} />);
    expect(screen.getByTestId("editor-toolbar")).toBeInTheDocument();
  });

  it("renders all action buttons", () => {
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} />);
    expect(screen.getByTestId("toolbar-bold")).toBeInTheDocument();
    expect(screen.getByTestId("toolbar-italic")).toBeInTheDocument();
  });

  it("calls onAction with action id on click", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    render(<EditorToolbar actions={mockActions} onAction={onAction} />);

    await user.click(screen.getByTestId("toolbar-bold"));
    expect(onAction).toHaveBeenCalledWith("bold");
  });

  it("shows aria-label for each button", () => {
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} />);
    expect(screen.getByLabelText("Bold")).toBeInTheDocument();
    expect(screen.getByLabelText("Italic")).toBeInTheDocument();
  });

  it("disables disabled actions", () => {
    const actions: ToolbarAction[] = [{ id: "bold", label: "Bold", icon: Bold, disabled: true }];
    render(<EditorToolbar actions={actions} onAction={vi.fn()} />);
    expect(screen.getByTestId("toolbar-bold")).toBeDisabled();
  });

  it("renders image button when onImageUpload provided", () => {
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} onImageUpload={vi.fn()} />);
    expect(screen.getByTestId("toolbar-image")).toBeInTheDocument();
  });

  it("does not render image button when onImageUpload not provided", () => {
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} />);
    expect(screen.queryByTestId("toolbar-image")).not.toBeInTheDocument();
  });

  it("calls onImageUpload when image button clicked", async () => {
    const user = userEvent.setup();
    const onImage = vi.fn();
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} onImageUpload={onImage} />);
    await user.click(screen.getByTestId("toolbar-image"));
    expect(onImage).toHaveBeenCalledOnce();
  });

  it("renders markdown toggle when onToggleMarkdown provided", () => {
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} onToggleMarkdown={vi.fn()} />);
    expect(screen.getByTestId("toolbar-markdown")).toBeInTheDocument();
  });

  it("shows correct label when in markdown mode", () => {
    render(
      <EditorToolbar
        actions={mockActions}
        onAction={vi.fn()}
        onToggleMarkdown={vi.fn()}
        markdownMode={true}
      />,
    );
    expect(screen.getByLabelText("Switch to rich text")).toBeInTheDocument();
  });

  it("shows correct label when not in markdown mode", () => {
    render(
      <EditorToolbar
        actions={mockActions}
        onAction={vi.fn()}
        onToggleMarkdown={vi.fn()}
        markdownMode={false}
      />,
    );
    expect(screen.getByLabelText("Switch to markdown")).toBeInTheDocument();
  });

  it("calls onToggleMarkdown when markdown button clicked", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<EditorToolbar actions={mockActions} onAction={vi.fn()} onToggleMarkdown={onToggle} />);
    await user.click(screen.getByTestId("toolbar-markdown"));
    expect(onToggle).toHaveBeenCalledOnce();
  });
});
