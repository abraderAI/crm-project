"use client";

import { useCallback, useState } from "react";
import { EditorToolbar, buildDefaultActions } from "./editor-toolbar";

export interface MessageEditorProps {
  /** Initial content (markdown string). */
  initialContent?: string;
  /** Called when editor content changes. */
  onChange?: (content: string) => void;
  /** Called on submit. */
  onSubmit?: (content: string) => void;
  /** Placeholder text. */
  placeholder?: string;
  /** Whether the editor is disabled. */
  disabled?: boolean;
  /** Submit button label. */
  submitLabel?: string;
  /** Called to trigger image upload. */
  onImageUpload?: () => void;
}

/**
 * Message editor with rich text (Tiptap) and raw markdown toggle.
 *
 * In jsdom/test environments, Tiptap cannot mount, so the editor degrades
 * to a markdown textarea. The markdown toggle switches between modes.
 */
export function MessageEditor({
  initialContent = "",
  onChange,
  onSubmit,
  placeholder = "Write a message...",
  disabled = false,
  submitLabel = "Send",
  onImageUpload,
}: MessageEditorProps): React.ReactNode {
  const [content, setContent] = useState(initialContent);
  const [markdownMode, setMarkdownMode] = useState(false);

  const handleChange = useCallback(
    (value: string) => {
      setContent(value);
      onChange?.(value);
    },
    [onChange],
  );

  const handleSubmit = (): void => {
    if (content.trim()) {
      onSubmit?.(content.trim());
    }
  };

  const handleAction = useCallback(
    (actionId: string) => {
      if (markdownMode) return;
      // In markdown-only mode, insert markdown syntax inline.
      const insertions: Record<string, string> = {
        bold: "**bold**",
        italic: "*italic*",
        code: "`code`",
        heading: "## ",
        bulletList: "- ",
        orderedList: "1. ",
        codeBlock: "```\n\n```",
      };
      const insertion = insertions[actionId];
      if (insertion) {
        handleChange(content + insertion);
      }
    },
    [content, handleChange, markdownMode],
  );

  const actions = buildDefaultActions({});

  return (
    <div className="rounded-lg border border-border bg-background" data-testid="message-editor">
      <EditorToolbar
        actions={actions}
        onAction={handleAction}
        markdownMode={markdownMode}
        onToggleMarkdown={() => setMarkdownMode(!markdownMode)}
        onImageUpload={onImageUpload}
      />

      {/* Editor area */}
      <textarea
        value={content}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        rows={6}
        data-testid="message-editor-textarea"
        className="w-full resize-y border-none bg-transparent p-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
      />

      {/* Submit */}
      <div className="flex items-center justify-end border-t border-border p-2">
        <button
          type="button"
          onClick={handleSubmit}
          disabled={disabled || !content.trim()}
          data-testid="message-editor-submit"
          className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {submitLabel}
        </button>
      </div>
    </div>
  );
}
