"use client";

import { useCallback, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { ArrowUp, ArrowDown, Eye, EyeOff, GripVertical, RotateCcw, Save } from "lucide-react";
import type { WidgetConfig } from "@/lib/tier-types";
import type { WidgetRegistry } from "./home-layout";
import { ResetConfirmDialog } from "./reset-confirm-dialog";

export interface HomeLayoutEditorProps {
  /** Current layout to edit. */
  layout: WidgetConfig[];
  /** Widget registry for looking up titles. */
  registry: WidgetRegistry;
  /** Called when the user saves the edited layout. */
  onSave: (layout: WidgetConfig[]) => Promise<void>;
  /** Called when the user resets to default. */
  onReset: () => Promise<void>;
  /** Whether the layout has been customized (show reset button). */
  isCustomized: boolean;
}

/** Editor for reordering and toggling visibility of home screen widgets. */
export function HomeLayoutEditor({
  layout,
  registry,
  onSave,
  onReset,
  isCustomized,
}: HomeLayoutEditorProps): ReactNode {
  const [editableLayout, setEditableLayout] = useState<WidgetConfig[]>(() => [...layout]);
  const [isSaving, setIsSaving] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [showResetDialog, setShowResetDialog] = useState(false);
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);
  const listRef = useRef<HTMLUListElement>(null);

  const toggleVisibility = (index: number): void => {
    setEditableLayout((prev) => {
      const next = [...prev];
      const item = next[index];
      if (item) {
        next[index] = { ...item, visible: !item.visible };
      }
      return next;
    });
  };

  const moveUp = useCallback((index: number): void => {
    if (index <= 0) return;
    setEditableLayout((prev) => {
      const next = [...prev];
      const item = next[index];
      const above = next[index - 1];
      if (item && above) {
        next[index - 1] = item;
        next[index] = above;
      }
      return next;
    });
  }, []);

  const moveDown = useCallback((index: number): void => {
    setEditableLayout((prev) => {
      if (index >= prev.length - 1) return prev;
      const next = [...prev];
      const item = next[index];
      const below = next[index + 1];
      if (item && below) {
        next[index + 1] = item;
        next[index] = below;
      }
      return next;
    });
  }, []);

  const handleSave = async (): Promise<void> => {
    setIsSaving(true);
    try {
      await onSave(editableLayout);
    } finally {
      setIsSaving(false);
    }
  };

  const handleResetClick = (): void => {
    setShowResetDialog(true);
  };

  const handleResetConfirm = async (): Promise<void> => {
    setShowResetDialog(false);
    setIsResetting(true);
    try {
      await onReset();
    } finally {
      setIsResetting(false);
    }
  };

  const handleResetCancel = (): void => {
    setShowResetDialog(false);
  };

  // Drag-and-drop handlers.
  const handleDragStart = (index: number): void => {
    setDragIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number): void => {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;
    setEditableLayout((prev) => {
      const next = [...prev];
      const dragged = next[dragIndex];
      if (!dragged) return prev;
      next.splice(dragIndex, 1);
      next.splice(index, 0, dragged);
      return next;
    });
    setDragIndex(index);
  };

  const handleDragEnd = (): void => {
    setDragIndex(null);
  };

  // Keyboard navigation for list items.
  const handleKeyDown = (e: KeyboardEvent<HTMLLIElement>, index: number): void => {
    switch (e.key) {
      case "ArrowUp":
        e.preventDefault();
        if (e.altKey && index > 0) {
          moveUp(index);
          setFocusedIndex(index - 1);
        } else if (index > 0) {
          setFocusedIndex(index - 1);
        }
        break;
      case "ArrowDown":
        e.preventDefault();
        if (e.altKey && index < editableLayout.length - 1) {
          moveDown(index);
          setFocusedIndex(index + 1);
        } else if (index < editableLayout.length - 1) {
          setFocusedIndex(index + 1);
        }
        break;
      case " ":
      case "Enter":
        e.preventDefault();
        toggleVisibility(index);
        break;
    }
  };

  // Focus management: when focusedIndex changes, focus the item.
  const focusItem = useCallback(
    (el: HTMLLIElement | null, index: number) => {
      if (el && index === focusedIndex) {
        el.focus();
      }
    },
    [focusedIndex],
  );

  return (
    <div data-testid="home-layout-editor" className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">Customize Home Screen</h3>
        <div className="flex gap-2">
          {isCustomized && (
            <button
              data-testid="editor-reset"
              onClick={handleResetClick}
              disabled={isResetting}
              className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-accent/50 disabled:opacity-50"
            >
              <RotateCcw className="h-3 w-3" />
              {isResetting ? "Resetting…" : "Reset to defaults"}
            </button>
          )}
          <button
            data-testid="editor-save"
            onClick={handleSave}
            disabled={isSaving}
            className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Save className="h-3 w-3" />
            {isSaving ? "Saving…" : "Save"}
          </button>
        </div>
      </div>

      <ul
        ref={listRef}
        data-testid="editor-widget-list"
        role="listbox"
        aria-label="Widget order"
        className="space-y-2"
      >
        {editableLayout.map((item, index) => {
          const entry = registry[item.widget_id];
          const title = entry?.title ?? item.widget_id;
          return (
            <li
              key={item.widget_id}
              ref={(el) => focusItem(el, index)}
              data-testid={`editor-item-${item.widget_id}`}
              role="option"
              aria-selected={index === focusedIndex}
              aria-grabbed={dragIndex === index}
              aria-label={`${title}${item.visible ? "" : " (hidden)"}`}
              tabIndex={index === focusedIndex || (focusedIndex === -1 && index === 0) ? 0 : -1}
              draggable
              onDragStart={() => handleDragStart(index)}
              onDragOver={(e) => handleDragOver(e, index)}
              onDragEnd={handleDragEnd}
              onKeyDown={(e) => handleKeyDown(e, index)}
              onFocus={() => setFocusedIndex(index)}
              className={`flex items-center gap-2 rounded-md border p-2 outline-none focus:ring-2 focus:ring-primary/50 ${
                dragIndex === index ? "border-primary bg-primary/5" : "border-border"
              }`}
            >
              <span
                className="cursor-grab text-muted-foreground"
                aria-hidden="true"
                data-testid={`drag-handle-${item.widget_id}`}
              >
                <GripVertical className="h-4 w-4" />
              </span>
              <button
                data-testid={`toggle-${item.widget_id}`}
                onClick={() => toggleVisibility(index)}
                aria-label={item.visible ? `Hide ${title}` : `Show ${title}`}
                className="rounded p-1 text-muted-foreground hover:bg-accent/50"
              >
                {item.visible ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
              </button>
              <span
                className={`flex-1 text-sm ${item.visible ? "text-foreground" : "text-muted-foreground line-through"}`}
              >
                {title}
              </span>
              <button
                data-testid={`move-up-${item.widget_id}`}
                onClick={() => moveUp(index)}
                disabled={index === 0}
                aria-label={`Move ${title} up`}
                className="rounded p-1 text-muted-foreground hover:bg-accent/50 disabled:opacity-30"
              >
                <ArrowUp className="h-4 w-4" />
              </button>
              <button
                data-testid={`move-down-${item.widget_id}`}
                onClick={() => moveDown(index)}
                disabled={index === editableLayout.length - 1}
                aria-label={`Move ${title} down`}
                className="rounded p-1 text-muted-foreground hover:bg-accent/50 disabled:opacity-30"
              >
                <ArrowDown className="h-4 w-4" />
              </button>
            </li>
          );
        })}
      </ul>

      <ResetConfirmDialog
        open={showResetDialog}
        onConfirm={handleResetConfirm}
        onCancel={handleResetCancel}
      />
    </div>
  );
}
