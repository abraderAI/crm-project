"use client";

import {
  Bold,
  Code,
  Heading2,
  Image,
  Italic,
  List,
  ListOrdered,
  FileCode,
  Undo,
  Redo,
} from "lucide-react";
import { cn } from "@/lib/utils";

export interface ToolbarAction {
  /** Unique key for the action. */
  key: string;
  /** Icon component. */
  icon: typeof Bold;
  /** Tooltip label. */
  label: string;
  /** Whether this action is currently active/toggled. */
  isActive?: boolean;
  /** Whether this action is disabled. */
  disabled?: boolean;
  /** Click handler. */
  onClick: () => void;
}

export interface EditorToolbarProps {
  /** Formatting actions from the editor. */
  actions: ToolbarAction[];
  /** Whether raw markdown mode is active. */
  isMarkdownMode?: boolean;
  /** Toggle raw markdown mode. */
  onToggleMarkdown?: () => void;
  /** Insert an image. */
  onInsertImage?: () => void;
}

/** Toolbar for the message editor with formatting buttons and markdown toggle. */
export function EditorToolbar({
  actions,
  isMarkdownMode = false,
  onToggleMarkdown,
  onInsertImage,
}: EditorToolbarProps): React.ReactNode {
  return (
    <div
      data-testid="editor-toolbar"
      className="flex flex-wrap items-center gap-0.5 border-b border-border px-2 py-1"
    >
      {actions.map((action) => (
        <button
          key={action.key}
          onClick={action.onClick}
          disabled={action.disabled}
          title={action.label}
          aria-label={action.label}
          data-testid={`toolbar-${action.key}`}
          className={cn(
            "rounded p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:opacity-40",
            action.isActive && "bg-accent text-foreground",
          )}
        >
          <action.icon className="h-4 w-4" />
        </button>
      ))}

      {/* Separator */}
      {(onInsertImage ?? onToggleMarkdown) && (
        <div className="mx-1 h-5 w-px bg-border" data-testid="toolbar-separator" />
      )}

      {/* Image insert */}
      {onInsertImage && (
        <button
          onClick={onInsertImage}
          title="Insert image"
          aria-label="Insert image"
          data-testid="toolbar-image"
          className="rounded p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Image className="h-4 w-4" />
        </button>
      )}

      {/* Markdown toggle */}
      {onToggleMarkdown && (
        <button
          onClick={onToggleMarkdown}
          title={isMarkdownMode ? "Switch to rich editor" : "Switch to raw markdown"}
          aria-label={isMarkdownMode ? "Switch to rich editor" : "Switch to raw markdown"}
          data-testid="toolbar-markdown"
          className={cn(
            "rounded p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
            isMarkdownMode && "bg-accent text-foreground",
          )}
        >
          <FileCode className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

/** Fluent command chain returned by editor.chain(). */
export interface ChainedCommandLike {
  focus: () => ChainedCommandLike;
  toggleBold: () => ChainedCommandLike;
  toggleItalic: () => ChainedCommandLike;
  toggleCode: () => ChainedCommandLike;
  toggleHeading: (attrs: { level: 1 | 2 | 3 | 4 | 5 | 6 }) => ChainedCommandLike;
  toggleBulletList: () => ChainedCommandLike;
  toggleOrderedList: () => ChainedCommandLike;
  toggleCodeBlock: () => ChainedCommandLike;
  undo: () => ChainedCommandLike;
  redo: () => ChainedCommandLike;
  run: () => boolean;
}

/** Minimal editor interface for building toolbar actions. */
export interface EditorForToolbar {
  isActive: (name: string) => boolean;
  chain: () => ChainedCommandLike;
  can: () => { chain: () => ChainedCommandLike };
}

/** Build default toolbar actions from a Tiptap-like editor API. */
export function buildDefaultActions(editor: EditorForToolbar): ToolbarAction[] {
  return [
    {
      key: "bold",
      icon: Bold,
      label: "Bold",
      isActive: editor.isActive("bold"),
      onClick: () => editor.chain().focus().toggleBold().run(),
    },
    {
      key: "italic",
      icon: Italic,
      label: "Italic",
      isActive: editor.isActive("italic"),
      onClick: () => editor.chain().focus().toggleItalic().run(),
    },
    {
      key: "code",
      icon: Code,
      label: "Inline code",
      isActive: editor.isActive("code"),
      onClick: () => editor.chain().focus().toggleCode().run(),
    },
    {
      key: "heading",
      icon: Heading2,
      label: "Heading",
      isActive: editor.isActive("heading"),
      onClick: () => editor.chain().focus().toggleHeading({ level: 2 }).run(),
    },
    {
      key: "bullet-list",
      icon: List,
      label: "Bullet list",
      isActive: editor.isActive("bulletList"),
      onClick: () => editor.chain().focus().toggleBulletList().run(),
    },
    {
      key: "ordered-list",
      icon: ListOrdered,
      label: "Ordered list",
      isActive: editor.isActive("orderedList"),
      onClick: () => editor.chain().focus().toggleOrderedList().run(),
    },
    {
      key: "code-block",
      icon: FileCode,
      label: "Code block",
      isActive: editor.isActive("codeBlock"),
      onClick: () => editor.chain().focus().toggleCodeBlock().run(),
    },
    {
      key: "undo",
      icon: Undo,
      label: "Undo",
      disabled: !editor.can().chain().focus().undo().run(),
      onClick: () => editor.chain().undo().run(),
    },
    {
      key: "redo",
      icon: Redo,
      label: "Redo",
      disabled: !editor.can().chain().focus().redo().run(),
      onClick: () => editor.chain().redo().run(),
    },
  ];
}
