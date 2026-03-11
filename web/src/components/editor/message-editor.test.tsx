import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock Tiptap hooks — useEditor returns a mock editor, EditorContent renders a div.
const mockGetHTML = vi.fn().mockReturnValue("<p>test</p>");
const mockSetContent = vi.fn();
const mockRun = vi.fn();
const mockSetImage = vi.fn().mockReturnValue({ run: mockRun });
const mockEditor = {
  getHTML: mockGetHTML,
  commands: { setContent: mockSetContent },
  isActive: () => false,
  chain: () => ({
    focus: () => ({
      toggleBold: () => ({ run: mockRun }),
      toggleItalic: () => ({ run: mockRun }),
      toggleCode: () => ({ run: mockRun }),
      toggleHeading: () => ({ run: mockRun }),
      toggleBulletList: () => ({ run: mockRun }),
      toggleOrderedList: () => ({ run: mockRun }),
      toggleCodeBlock: () => ({ run: mockRun }),
      setImage: mockSetImage,
    }),
    undo: () => ({ run: mockRun }),
    redo: () => ({ run: mockRun }),
  }),
  can: () => ({
    chain: () => ({
      focus: () => ({
        undo: () => ({ run: () => true }),
        redo: () => ({ run: () => true }),
      }),
    }),
  }),
};

vi.mock("@tiptap/react", () => ({
  useEditor: () => mockEditor,
  EditorContent: () => <div data-testid="tiptap-content">Editor content</div>,
}));

vi.mock("@tiptap/starter-kit", () => ({
  default: { configure: () => ({}) },
}));

vi.mock("@tiptap/extension-code-block-lowlight", () => ({
  default: { configure: () => ({}) },
}));

vi.mock("@tiptap/extension-image", () => ({
  default: { configure: () => ({}) },
}));

vi.mock("lowlight", () => ({
  common: {},
  createLowlight: () => ({}),
}));

// Import after mocks — dynamic import required since vi.mock is hoisted.
import { MessageEditor } from "./message-editor";

describe("MessageEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders editor container", () => {
    render(<MessageEditor onSubmit={vi.fn()} />);
    expect(screen.getByTestId("message-editor")).toBeInTheDocument();
  });

  it("renders toolbar", () => {
    render(<MessageEditor onSubmit={vi.fn()} />);
    expect(screen.getByTestId("editor-toolbar")).toBeInTheDocument();
  });

  it("renders rich editor by default", () => {
    render(<MessageEditor onSubmit={vi.fn()} />);
    expect(screen.getByTestId("rich-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("markdown-textarea")).not.toBeInTheDocument();
  });

  it("switches to markdown mode on toggle", async () => {
    const user = userEvent.setup();
    render(<MessageEditor onSubmit={vi.fn()} />);

    await user.click(screen.getByTestId("toolbar-markdown"));
    expect(screen.getByTestId("markdown-textarea")).toBeInTheDocument();
    expect(screen.queryByTestId("rich-editor")).not.toBeInTheDocument();
  });

  it("switches back to rich mode", async () => {
    const user = userEvent.setup();
    render(<MessageEditor onSubmit={vi.fn()} />);

    await user.click(screen.getByTestId("toolbar-markdown")); // to markdown
    await user.click(screen.getByTestId("toolbar-markdown")); // back to rich
    expect(screen.getByTestId("rich-editor")).toBeInTheDocument();
    expect(mockSetContent).toHaveBeenCalled();
  });

  it("renders submit button by default", () => {
    render(<MessageEditor onSubmit={vi.fn()} />);
    expect(screen.getByTestId("editor-submit-btn")).toBeInTheDocument();
    expect(screen.getByTestId("editor-submit-btn")).toHaveTextContent("Send");
  });

  it("hides submit button when showSubmit is false", () => {
    render(<MessageEditor onSubmit={vi.fn()} showSubmit={false} />);
    expect(screen.queryByTestId("editor-submit-btn")).not.toBeInTheDocument();
  });

  it("uses custom submit label", () => {
    render(<MessageEditor onSubmit={vi.fn()} submitLabel="Save" />);
    expect(screen.getByTestId("editor-submit-btn")).toHaveTextContent("Save");
  });

  it("calls onSubmit with HTML in rich mode", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<MessageEditor onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("editor-submit-btn"));
    expect(onSubmit).toHaveBeenCalledWith("<p>test</p>");
  });

  it("calls onSubmit with markdown content in markdown mode", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<MessageEditor onSubmit={onSubmit} initialContent="# Hello" />);

    await user.click(screen.getByTestId("toolbar-markdown"));
    const textarea = screen.getByTestId("markdown-textarea");
    await user.clear(textarea);
    await user.type(textarea, "raw markdown");
    await user.click(screen.getByTestId("editor-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith("raw markdown");
  });

  it("disables submit button when disabled", () => {
    render(<MessageEditor onSubmit={vi.fn()} disabled={true} />);
    expect(screen.getByTestId("editor-submit-btn")).toBeDisabled();
  });

  it("renders markdown toggle button", () => {
    render(<MessageEditor onSubmit={vi.fn()} />);
    expect(screen.getByTestId("toolbar-markdown")).toBeInTheDocument();
  });

  it("renders image insert button", () => {
    render(<MessageEditor onSubmit={vi.fn()} />);
    expect(screen.getByTestId("toolbar-image")).toBeInTheDocument();
  });

  it("prompts for image URL on insert image", async () => {
    const user = userEvent.setup();
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValue("https://example.com/img.png");
    render(<MessageEditor onSubmit={vi.fn()} />);

    await user.click(screen.getByTestId("toolbar-image"));
    expect(promptSpy).toHaveBeenCalledWith("Image URL:");
    expect(mockSetImage).toHaveBeenCalledWith({ src: "https://example.com/img.png" });
    promptSpy.mockRestore();
  });

  it("does not insert image when prompt cancelled", async () => {
    const user = userEvent.setup();
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValue(null);
    render(<MessageEditor onSubmit={vi.fn()} />);

    await user.click(screen.getByTestId("toolbar-image"));
    expect(mockSetImage).not.toHaveBeenCalled();
    promptSpy.mockRestore();
  });
});
