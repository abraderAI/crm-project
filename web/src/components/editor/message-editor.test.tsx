import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { MessageEditor } from "./message-editor";

describe("MessageEditor", () => {
  it("renders the editor", () => {
    render(<MessageEditor />);
    expect(screen.getByTestId("message-editor")).toBeInTheDocument();
    expect(screen.getByTestId("message-editor-textarea")).toBeInTheDocument();
    expect(screen.getByTestId("editor-toolbar")).toBeInTheDocument();
  });

  it("shows placeholder text", () => {
    render(<MessageEditor placeholder="Type here..." />);
    expect(screen.getByTestId("message-editor-textarea")).toHaveAttribute(
      "placeholder",
      "Type here...",
    );
  });

  it("shows initial content", () => {
    render(<MessageEditor initialContent="Hello world" />);
    expect(screen.getByTestId("message-editor-textarea")).toHaveValue("Hello world");
  });

  it("calls onChange when text changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MessageEditor onChange={onChange} />);

    await user.type(screen.getByTestId("message-editor-textarea"), "test");
    expect(onChange).toHaveBeenCalled();
  });

  it("calls onSubmit with trimmed content", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<MessageEditor onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("message-editor-textarea"), "  Hello  ");
    await user.click(screen.getByTestId("message-editor-submit"));
    expect(onSubmit).toHaveBeenCalledWith("Hello");
  });

  it("does not submit when content is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<MessageEditor onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("message-editor-submit"));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("disables submit when content is only whitespace", () => {
    render(<MessageEditor />);
    expect(screen.getByTestId("message-editor-submit")).toBeDisabled();
  });

  it("enables submit when content has text", async () => {
    const user = userEvent.setup();
    render(<MessageEditor />);

    await user.type(screen.getByTestId("message-editor-textarea"), "content");
    expect(screen.getByTestId("message-editor-submit")).not.toBeDisabled();
  });

  it("shows custom submit label", () => {
    render(<MessageEditor submitLabel="Reply" initialContent="x" />);
    expect(screen.getByTestId("message-editor-submit")).toHaveTextContent("Reply");
  });

  it("disables textarea when disabled", () => {
    render(<MessageEditor disabled={true} />);
    expect(screen.getByTestId("message-editor-textarea")).toBeDisabled();
  });

  it("disables submit when disabled", () => {
    render(<MessageEditor disabled={true} initialContent="test" />);
    expect(screen.getByTestId("message-editor-submit")).toBeDisabled();
  });

  it("renders toolbar with markdown toggle", () => {
    render(<MessageEditor />);
    expect(screen.getByTestId("toolbar-markdown")).toBeInTheDocument();
  });

  it("toggles markdown mode", async () => {
    const user = userEvent.setup();
    render(<MessageEditor />);

    // Initially not markdown mode
    expect(screen.getByLabelText("Switch to markdown")).toBeInTheDocument();

    await user.click(screen.getByTestId("toolbar-markdown"));
    expect(screen.getByLabelText("Switch to rich text")).toBeInTheDocument();
  });

  it("renders image upload button when onImageUpload provided", () => {
    render(<MessageEditor onImageUpload={vi.fn()} />);
    expect(screen.getByTestId("toolbar-image")).toBeInTheDocument();
  });

  it("inserts bold markdown on toolbar action", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MessageEditor onChange={onChange} />);

    await user.click(screen.getByTestId("toolbar-bold"));
    expect(screen.getByTestId("message-editor-textarea")).toHaveValue("**bold**");
  });

  it("inserts code block markdown on toolbar action", async () => {
    const user = userEvent.setup();
    render(<MessageEditor />);

    await user.click(screen.getByTestId("toolbar-codeBlock"));
    expect(screen.getByTestId("message-editor-textarea")).toHaveValue("```\n\n```");
  });
});
