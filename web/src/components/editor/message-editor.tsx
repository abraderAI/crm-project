"use client";

import { useRef, useState } from "react";
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
  /** Minimum height for the editable area (e.g. "50vh"). Defaults to 320px. */
  editorMinHeight?: string;
  /** Optional image options surfaced in the image insertion panel. */
  imageOptions?: { label: string; url: string }[];
}

/** Tiptap-based message editor with rich formatting, image insertion, and markdown mode. */
export function MessageEditor({
  initialContent = "",
  onSubmit,
  onChange,
  placeholder = "Write a message...",
  disabled = false,
  showSubmit = true,
  submitLabel = "Send",
  editorMinHeight = "320px",
  imageOptions = [],
}: MessageEditorProps): React.ReactNode {
  const [isMarkdownMode, setIsMarkdownMode] = useState(false);
  const [markdownContent, setMarkdownContent] = useState(initialContent);
  const [isImagePanelOpen, setIsImagePanelOpen] = useState(false);
  const [imageUrl, setImageUrl] = useState("");
  const [imageError, setImageError] = useState("");
  const imageFileInputRef = useRef<HTMLInputElement>(null);

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
      editor.commands.setContent(markdownContent);
    } else if (editor) {
      setMarkdownContent(editor.getHTML());
    }
    setIsMarkdownMode(!isMarkdownMode);
  };

  const handleInsertImage = (): void => {
    setImageError("");
    setIsImagePanelOpen((prev) => !prev);
  };

  const insertImageByUrl = (): void => {
    if (!editor) return;
    if (!imageUrl.trim()) {
      setImageError("Provide an image URL.");
      return;
    }
    editor.chain().focus().setImage({ src: imageUrl.trim() }).run();
    setImageError("");
    setImageUrl("");
    setIsImagePanelOpen(false);
  };

  const insertImageByFile = (file: File): void => {
    if (!editor) return;
    if (!file.type.startsWith("image/")) {
      setImageError("Please choose an image file.");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      const src = reader.result;
      if (typeof src === "string") {
        editor.chain().focus().setImage({ src }).run();
        setImageError("");
        setIsImagePanelOpen(false);
      }
    };
    reader.onerror = () => {
      setImageError("Unable to read selected image.");
    };
    reader.readAsDataURL(file);
  };

  const actions = editor ? buildDefaultActions(editor) : [];

  return (
    <div
      data-testid="message-editor"
      className="overflow-hidden rounded-xl border border-border/80 bg-background/95 shadow-sm transition-shadow hover:shadow-md"
    >
      <EditorToolbar
        actions={actions}
        isMarkdownMode={isMarkdownMode}
        onToggleMarkdown={toggleMarkdown}
        onInsertImage={handleInsertImage}
      />
      {isImagePanelOpen && (
        <div
          data-testid="image-insert-panel"
          className="flex flex-col gap-2 border-b border-border bg-muted/40 px-3 py-3"
        >
          <div className="flex flex-wrap items-center gap-2">
            <input
              type="url"
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
              placeholder="https://example.com/image.png"
              data-testid="image-url-input"
              className="h-8 min-w-52 flex-1 rounded-md border border-border bg-background px-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none"
            />
            <button
              type="button"
              data-testid="insert-image-url-btn"
              onClick={insertImageByUrl}
              className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90"
            >
              Insert URL
            </button>
            <label
              htmlFor="image-file-input"
              className="cursor-pointer rounded-md border border-border bg-background px-3 py-1.5 text-xs text-foreground hover:bg-accent"
            >
              Choose image
            </label>
            <input
              id="image-file-input"
              ref={imageFileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) {
                  insertImageByFile(file);
                }
                if (imageFileInputRef.current) {
                  imageFileInputRef.current.value = "";
                }
              }}
            />
          </div>
          {imageOptions.length > 0 && (
            <div className="flex flex-wrap gap-1.5" data-testid="image-options-list">
              {imageOptions.map((opt) => (
                <button
                  key={`${opt.label}-${opt.url}`}
                  type="button"
                  data-testid={`image-option-${opt.label}`}
                  onClick={() => {
                    if (!editor) return;
                    editor.chain().focus().setImage({ src: opt.url }).run();
                    setImageError("");
                    setIsImagePanelOpen(false);
                  }}
                  className="rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground hover:bg-accent"
                >
                  {opt.label}
                </button>
              ))}
            </div>
          )}
          {imageError && (
            <p data-testid="image-insert-error" className="text-xs text-red-600">
              {imageError}
            </p>
          )}
        </div>
      )}

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
          style={{ minHeight: editorMinHeight }}
          className="w-full resize-y bg-background px-4 py-3 font-mono text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
        />
      ) : (
        <div
          className="max-w-none bg-background px-4 py-3"
          style={{ minHeight: editorMinHeight }}
          data-testid="rich-editor"
        >
          <EditorContent
            editor={editor}
            className="[&_.ProseMirror]:min-h-[inherit] [&_.ProseMirror]:outline-none [&_.ProseMirror]:text-sm [&_.ProseMirror]:leading-6 [&_.ProseMirror]:text-foreground [&_.ProseMirror_p]:my-2 [&_.ProseMirror_ul]:my-2 [&_.ProseMirror_ul]:ml-5 [&_.ProseMirror_ul]:list-disc [&_.ProseMirror_ol]:my-2 [&_.ProseMirror_ol]:ml-5 [&_.ProseMirror_ol]:list-decimal [&_.ProseMirror_li]:my-1 [&_.ProseMirror_img]:my-2 [&_.ProseMirror_img]:max-w-full [&_.ProseMirror_img]:rounded-md"
          />
        </div>
      )}

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
