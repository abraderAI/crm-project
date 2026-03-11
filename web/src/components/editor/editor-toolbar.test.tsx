import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Bold, Italic } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { EditorToolbar, buildDefaultActions, type ToolbarAction } from "./editor-toolbar";

const mockActions: ToolbarAction[] = [
  { key: "bold", icon: Bold, label: "Bold", isActive: false, onClick: vi.fn() },
  { key: "italic", icon: Italic, label: "Italic", isActive: true, onClick: vi.fn() },
];

describe("EditorToolbar", () => {
  it("renders toolbar container", () => {
    render(<EditorToolbar actions={mockActions} />);
    expect(screen.getByTestId("editor-toolbar")).toBeInTheDocument();
  });

  it("renders all action buttons", () => {
    render(<EditorToolbar actions={mockActions} />);
    expect(screen.getByTestId("toolbar-bold")).toBeInTheDocument();
    expect(screen.getByTestId("toolbar-italic")).toBeInTheDocument();
  });

  it("calls onClick when action button clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    const actions = [{ key: "test", icon: Bold, label: "Test", onClick }];
    render(<EditorToolbar actions={actions} />);

    await user.click(screen.getByTestId("toolbar-test"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("renders disabled action buttons", () => {
    const actions = [{ key: "undo", icon: Bold, label: "Undo", disabled: true, onClick: vi.fn() }];
    render(<EditorToolbar actions={actions} />);
    expect(screen.getByTestId("toolbar-undo")).toBeDisabled();
  });

  it("shows active state on active actions", () => {
    render(<EditorToolbar actions={mockActions} />);
    const italic = screen.getByTestId("toolbar-italic");
    expect(italic.className).toContain("bg-accent");
  });

  it("renders image button when onInsertImage provided", () => {
    render(<EditorToolbar actions={mockActions} onInsertImage={vi.fn()} />);
    expect(screen.getByTestId("toolbar-image")).toBeInTheDocument();
  });

  it("hides image button when onInsertImage not provided", () => {
    render(<EditorToolbar actions={mockActions} />);
    expect(screen.queryByTestId("toolbar-image")).not.toBeInTheDocument();
  });

  it("calls onInsertImage when image clicked", async () => {
    const user = userEvent.setup();
    const onInsertImage = vi.fn();
    render(<EditorToolbar actions={mockActions} onInsertImage={onInsertImage} />);

    await user.click(screen.getByTestId("toolbar-image"));
    expect(onInsertImage).toHaveBeenCalledOnce();
  });

  it("renders markdown toggle when onToggleMarkdown provided", () => {
    render(<EditorToolbar actions={mockActions} onToggleMarkdown={vi.fn()} />);
    expect(screen.getByTestId("toolbar-markdown")).toBeInTheDocument();
  });

  it("hides markdown toggle when onToggleMarkdown not provided", () => {
    render(<EditorToolbar actions={mockActions} />);
    expect(screen.queryByTestId("toolbar-markdown")).not.toBeInTheDocument();
  });

  it("calls onToggleMarkdown when clicked", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<EditorToolbar actions={mockActions} onToggleMarkdown={onToggle} />);

    await user.click(screen.getByTestId("toolbar-markdown"));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it("shows correct label for markdown mode", () => {
    render(
      <EditorToolbar actions={mockActions} onToggleMarkdown={vi.fn()} isMarkdownMode={true} />,
    );
    expect(screen.getByLabelText("Switch to rich editor")).toBeInTheDocument();
  });

  it("shows correct label for rich mode", () => {
    render(
      <EditorToolbar actions={mockActions} onToggleMarkdown={vi.fn()} isMarkdownMode={false} />,
    );
    expect(screen.getByLabelText("Switch to raw markdown")).toBeInTheDocument();
  });

  it("renders separator when image or markdown toggle present", () => {
    render(<EditorToolbar actions={mockActions} onInsertImage={vi.fn()} />);
    expect(screen.getByTestId("toolbar-separator")).toBeInTheDocument();
  });

  it("hides separator when neither image nor markdown toggle present", () => {
    render(<EditorToolbar actions={mockActions} />);
    expect(screen.queryByTestId("toolbar-separator")).not.toBeInTheDocument();
  });

  it("renders actions with correct aria-labels", () => {
    render(<EditorToolbar actions={mockActions} />);
    expect(screen.getByLabelText("Bold")).toBeInTheDocument();
    expect(screen.getByLabelText("Italic")).toBeInTheDocument();
  });
});

describe("buildDefaultActions", () => {
  it("returns 9 default actions", () => {
    const run = vi.fn();
    const chain = {
      focus: () => ({
        toggleBold: () => ({ run }),
        toggleItalic: () => ({ run }),
        toggleCode: () => ({ run }),
        toggleHeading: () => ({ run }),
        toggleBulletList: () => ({ run }),
        toggleOrderedList: () => ({ run }),
        toggleCodeBlock: () => ({ run }),
      }),
      undo: () => ({ run }),
      redo: () => ({ run }),
    };
    const editor = {
      isActive: () => false,
      chain: () => chain,
      can: () => ({
        chain: () => ({
          focus: () => ({ undo: () => ({ run: () => true }), redo: () => ({ run: () => false }) }),
        }),
      }),
    };

    const actions = buildDefaultActions(editor);
    expect(actions).toHaveLength(9);
    expect(actions.map((a) => a.key)).toEqual([
      "bold",
      "italic",
      "code",
      "heading",
      "bullet-list",
      "ordered-list",
      "code-block",
      "undo",
      "redo",
    ]);
  });

  it("calls editor methods when actions are invoked", () => {
    const run = vi.fn();
    const chain = {
      focus: () => ({
        toggleBold: () => ({ run }),
        toggleItalic: () => ({ run }),
        toggleCode: () => ({ run }),
        toggleHeading: () => ({ run }),
        toggleBulletList: () => ({ run }),
        toggleOrderedList: () => ({ run }),
        toggleCodeBlock: () => ({ run }),
      }),
      undo: () => ({ run }),
      redo: () => ({ run }),
    };
    const editor = {
      isActive: () => false,
      chain: () => chain,
      can: () => ({
        chain: () => ({
          focus: () => ({ undo: () => ({ run: () => true }), redo: () => ({ run: () => true }) }),
        }),
      }),
    };

    const actions = buildDefaultActions(editor);
    const boldAction = actions.find((a) => a.key === "bold");
    boldAction?.onClick();
    expect(run).toHaveBeenCalled();
  });
});
