"use client";

import { useState } from "react";
import { Pin, Lock, EyeOff, ArrowRightLeft, GitMerge } from "lucide-react";
import { cn } from "@/lib/utils";
import type { BoardOption } from "@/lib/api-types";

export interface ModerationActionsProps {
  /** Current thread ID. */
  threadId: string;
  /** Whether the thread is currently pinned. */
  isPinned: boolean;
  /** Whether the thread is currently locked. */
  isLocked: boolean;
  /** Whether the thread is currently hidden. */
  isHidden: boolean;
  /** Called when pin is toggled. */
  onTogglePin: (threadId: string) => void;
  /** Called when lock is toggled. */
  onToggleLock: (threadId: string) => void;
  /** Called when hide is toggled. */
  onToggleHide: (threadId: string) => void;
  /** Called when the thread is moved to a different board. */
  onMove: (threadId: string, targetBoardId: string) => void;
  /** Called when the thread is merged into another. */
  onMerge: (threadId: string, targetThreadId: string) => void;
  /** Available boards for the move action. */
  boards?: BoardOption[];
  /** Whether any action is in progress. */
  loading?: boolean;
}

/** Moderation action bar for pin, lock, hide, move, and merge operations. */
export function ModerationActions({
  threadId,
  isPinned,
  isLocked,
  isHidden,
  onTogglePin,
  onToggleLock,
  onToggleHide,
  onMove,
  onMerge,
  boards = [],
  loading = false,
}: ModerationActionsProps): React.ReactNode {
  const [showMove, setShowMove] = useState(false);
  const [showMerge, setShowMerge] = useState(false);
  const [moveTarget, setMoveTarget] = useState("");
  const [mergeTarget, setMergeTarget] = useState("");

  const handleMove = (): void => {
    if (!moveTarget) return;
    onMove(threadId, moveTarget);
    setShowMove(false);
    setMoveTarget("");
  };

  const handleMerge = (): void => {
    if (!mergeTarget.trim()) return;
    onMerge(threadId, mergeTarget.trim());
    setShowMerge(false);
    setMergeTarget("");
  };

  return (
    <div data-testid="moderation-actions" className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold text-foreground">Moderation</h3>

      {/* Toggle buttons */}
      <div className="flex flex-wrap gap-2" data-testid="toggle-actions">
        <button
          onClick={() => onTogglePin(threadId)}
          disabled={loading}
          data-testid="action-pin"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors",
            isPinned
              ? "border-primary bg-primary/10 text-primary"
              : "border-border text-muted-foreground hover:bg-accent",
            loading && "opacity-50",
          )}
        >
          <Pin className="h-3.5 w-3.5" />
          {isPinned ? "Unpin" : "Pin"}
        </button>
        <button
          onClick={() => onToggleLock(threadId)}
          disabled={loading}
          data-testid="action-lock"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors",
            isLocked
              ? "border-primary bg-primary/10 text-primary"
              : "border-border text-muted-foreground hover:bg-accent",
            loading && "opacity-50",
          )}
        >
          <Lock className="h-3.5 w-3.5" />
          {isLocked ? "Unlock" : "Lock"}
        </button>
        <button
          onClick={() => onToggleHide(threadId)}
          disabled={loading}
          data-testid="action-hide"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors",
            isHidden
              ? "border-destructive bg-destructive/10 text-destructive"
              : "border-border text-muted-foreground hover:bg-accent",
            loading && "opacity-50",
          )}
        >
          <EyeOff className="h-3.5 w-3.5" />
          {isHidden ? "Unhide" : "Hide"}
        </button>
        <button
          onClick={() => {
            setShowMove(!showMove);
            setShowMerge(false);
          }}
          disabled={loading}
          data-testid="action-move-toggle"
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent"
        >
          <ArrowRightLeft className="h-3.5 w-3.5" />
          Move
        </button>
        <button
          onClick={() => {
            setShowMerge(!showMerge);
            setShowMove(false);
          }}
          disabled={loading}
          data-testid="action-merge-toggle"
          className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent"
        >
          <GitMerge className="h-3.5 w-3.5" />
          Merge
        </button>
      </div>

      {/* Move panel */}
      {showMove && (
        <div className="rounded-lg border border-border p-3" data-testid="move-panel">
          <p className="mb-2 text-xs font-medium text-foreground">Move to board:</p>
          {boards.length === 0 ? (
            <p className="text-xs text-muted-foreground" data-testid="move-no-boards">
              No other boards available.
            </p>
          ) : (
            <>
              <select
                value={moveTarget}
                onChange={(e) => setMoveTarget(e.target.value)}
                data-testid="move-board-select"
                className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
              >
                <option value="">Select a board...</option>
                {boards.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
              <button
                onClick={handleMove}
                disabled={!moveTarget || loading}
                data-testid="move-confirm-btn"
                className="mt-2 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                Confirm Move
              </button>
            </>
          )}
        </div>
      )}

      {/* Merge panel */}
      {showMerge && (
        <div className="rounded-lg border border-border p-3" data-testid="merge-panel">
          <p className="mb-2 text-xs font-medium text-foreground">Merge into thread ID:</p>
          <input
            type="text"
            value={mergeTarget}
            onChange={(e) => setMergeTarget(e.target.value)}
            placeholder="Target thread ID..."
            data-testid="merge-thread-input"
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          />
          <button
            onClick={handleMerge}
            disabled={!mergeTarget.trim() || loading}
            data-testid="merge-confirm-btn"
            className="mt-2 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            Confirm Merge
          </button>
        </div>
      )}
    </div>
  );
}
