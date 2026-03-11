"use client";

import { useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";
import TiptapImage from "@tiptap/extension-image";
import { common, createLowlight } from "lowlight";
import { EditorToolbar, buildDefaultActions } from "./editor-toolbar";

const lowlight = createLowlight(common);

export interface MessageEditorProps {
  /** Initial content (HTML or markdown string). */
  initialContent?: string;
  /** Called when the user submits the message. */
  onSubmit: (content: string) => void;
  /** Called when content changes. */
  onChange?: (content: string) => void;
  /** Placeholder text. */
  placeholder?: string;
  /** Whether the editor is disabled. */
  disabled?: boolean;
  /** Whether to show the submit button. */
  showSubmit?: boolean;
  /** Submit button label. */
  submitLabel?: string;
}

/** Tiptap-based message editor with GFM, code highlighting, image insert, and markdown toggle. */
export function MessageEditor({
  initialContent = "",
  onSubmit,
  onChange,
  placeholder = "Write a message...",
  disabled = false,
  showSubmit = true,
  submitLabel = "Send",
}: MessageEditorProps): React.ReactNode {
  const [isMarkdownMode, setIsMarkdownMode] = useState(false);
  const [markdownContent, setMarkdownContent] = useState(initialContent);

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false }),
      CodeBlockLowlight.configure({ lowlight }),
      TiptapImage.configure({ inline: false, allowBase64: true }),
    ],
    content: initialContent,
    editable: !disabled && !isMarkdownMode,
    onUpdate: ({ editor: e }) => {
      const html = e.getHTML();
      onChange?.(html);
    },
    immediatelyRender: false,
  });

  const handleSubmit = (): void => {
    if (isMarkdownMode) {
      onSubmit(markdownContent);
    } else if (editor) {
      onSubmit(editor.getHTML());
    }
  };

  const toggleMarkdown = (): void => {
    if (isMarkdownMode && editor) {
      // Switching back to rich — load markdown as HTML (simplified).
      editor.commands.setContent(markdownContent);
    } else if (editor) {
      // Switching to markdown — save current HTML.
      setMarkdownContent(editor.getHTML());
    }
    setIsMarkdownMode(!isMarkdownMode);
  };

  const handleInsertImage = (): void => {
    if (!editor) return;
    const url = window.prompt("Image URL:");
    if (url) {
      editor.chain().focus().setImage({ src: url }).run();
    }
  };

  const actions = editor ? buildDefaultActions(editor) : [];

  return (
    <div
      data-testid="message-editor"
      className="overflow-hidden rounded-lg border border-border bg-background"
    >
      {/* Toolbar */}
      <EditorToolbar
        actions={actions}
        isMarkdownMode={isMarkdownMode}
        onToggleMarkdown={toggleMarkdown}
        onInsertImage={handleInsertImage}
      />

      {/* Editor area */}
      {isMarkdownMode ? (
        <textarea
          value={markdownContent}
          onChange={(e) => {
            setMarkdownContent(e.target.value);
            onChange?.(e.target.value);
          }}
          placeholder={placeholder}
          disabled={disabled}
          data-testid="markdown-textarea"
          className="w-full min-h-[120px] resize-y bg-background p-3 font-mono text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
        />
      ) : (
        <div className="prose prose-sm max-w-none p-3" data-testid="rich-editor">
          <EditorContent editor={editor} />
        </div>
      )}

      {/* Submit */}
      {showSubmit && (
        <div className="flex justify-end border-t border-border px-3 py-2">
          <button
            onClick={handleSubmit}
            disabled={disabled}
            data-testid="editor-submit-btn"
            className="rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {submitLabel}
          </button>
        </div>
      )}
    </div>
  );
}
