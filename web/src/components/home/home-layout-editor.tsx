"use client";

import { useState, type ReactNode } from "react";
import { ArrowUp, ArrowDown, Eye, EyeOff, RotateCcw, Save } from "lucide-react";
import type { WidgetConfig } from "@/lib/tier-types";
import type { WidgetRegistry } from "./home-layout";

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

  const moveUp = (index: number): void => {
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
  };

  const moveDown = (index: number): void => {
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
  };

  const handleSave = async (): Promise<void> => {
    setIsSaving(true);
    try {
      await onSave(editableLayout);
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = async (): Promise<void> => {
    setIsResetting(true);
    try {
      await onReset();
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div data-testid="home-layout-editor" className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">Customize Home Screen</h3>
        <div className="flex gap-2">
          {isCustomized && (
            <button
              data-testid="editor-reset"
              onClick={handleReset}
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

      <ul data-testid="editor-widget-list" className="space-y-2">
        {editableLayout.map((item, index) => {
          const entry = registry[item.widget_id];
          const title = entry?.title ?? item.widget_id;
          return (
            <li
              key={item.widget_id}
              data-testid={`editor-item-${item.widget_id}`}
              className="flex items-center gap-2 rounded-md border border-border p-2"
            >
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
    </div>
  );
}
