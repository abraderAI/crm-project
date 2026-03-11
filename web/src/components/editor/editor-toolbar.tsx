"use client";

import {
  Bold,
  Code,
  CodeSquare,
  Heading2,
  ImageIcon,
  Italic,
  List,
  ListOrdered,
  FileCode,
} from "lucide-react";
import { cn } from "@/lib/utils";

export interface ToolbarAction {
  id: string;
  label: string;
  icon: typeof Bold;
  isActive?: boolean;
  disabled?: boolean;
}

export interface EditorToolbarProps {
  /** Toolbar button actions. */
  actions: ToolbarAction[];
  /** Called when a toolbar button is clicked. */
  onAction: (actionId: string) => void;
  /** Whether markdown mode is active. */
  markdownMode?: boolean;
  /** Called to toggle markdown mode. */
  onToggleMarkdown?: () => void;
  /** Called to trigger image upload. */
  onImageUpload?: () => void;
}

/** Build default toolbar actions for the Tiptap editor. */
export function buildDefaultActions(activeStates: Record<string, boolean>): ToolbarAction[] {
  return [
    { id: "bold", label: "Bold", icon: Bold, isActive: activeStates["bold"] },
    { id: "italic", label: "Italic", icon: Italic, isActive: activeStates["italic"] },
    { id: "code", label: "Code", icon: Code, isActive: activeStates["code"] },
    { id: "heading", label: "Heading", icon: Heading2, isActive: activeStates["heading"] },
    { id: "bulletList", label: "Bullet list", icon: List, isActive: activeStates["bulletList"] },
    {
      id: "orderedList",
      label: "Ordered list",
      icon: ListOrdered,
      isActive: activeStates["orderedList"],
    },
    {
      id: "codeBlock",
      label: "Code block",
      icon: CodeSquare,
      isActive: activeStates["codeBlock"],
    },
  ];
}

/** Toolbar for the Tiptap message editor. */
export function EditorToolbar({
  actions,
  onAction,
  markdownMode = false,
  onToggleMarkdown,
  onImageUpload,
}: EditorToolbarProps): React.ReactNode {
  return (
    <div
      className="flex items-center gap-0.5 border-b border-border p-1"
      data-testid="editor-toolbar"
    >
      {actions.map((action) => {
        const Icon = action.icon;
        return (
          <button
            key={action.id}
            type="button"
            onClick={() => onAction(action.id)}
            disabled={action.disabled}
            title={action.label}
            aria-label={action.label}
            data-testid={`toolbar-${action.id}`}
            className={cn(
              "rounded-md p-1.5 text-sm transition-colors",
              action.isActive
                ? "bg-accent text-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-foreground",
              action.disabled && "opacity-50",
            )}
          >
            <Icon className="h-4 w-4" />
          </button>
        );
      })}

      {/* Separator */}
      <div className="mx-1 h-5 w-px bg-border" />

      {/* Image upload */}
      {onImageUpload && (
        <button
          type="button"
          onClick={onImageUpload}
          title="Insert image"
          aria-label="Insert image"
          data-testid="toolbar-image"
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <ImageIcon className="h-4 w-4" />
        </button>
      )}

      {/* Markdown toggle */}
      {onToggleMarkdown && (
        <button
          type="button"
          onClick={onToggleMarkdown}
          title={markdownMode ? "Switch to rich text" : "Switch to markdown"}
          aria-label={markdownMode ? "Switch to rich text" : "Switch to markdown"}
          data-testid="toolbar-markdown"
          className={cn(
            "rounded-md p-1.5 text-sm transition-colors",
            markdownMode
              ? "bg-accent text-foreground"
              : "text-muted-foreground hover:bg-accent hover:text-foreground",
          )}
        >
          <FileCode className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
