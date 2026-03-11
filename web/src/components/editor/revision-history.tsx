"use client";

import { History } from "lucide-react";
import type { Revision } from "@/lib/api-types";
import { formatDate } from "@/components/thread/thread-list";

export interface RevisionHistoryProps {
  revisions: Revision[];
  /** Called when a revision is selected for viewing. */
  onViewRevision?: (revision: Revision) => void;
  /** Currently selected revision ID. */
  selectedId?: string;
}

/** List of entity revisions with version, editor, timestamp. */
export function RevisionHistory({
  revisions,
  onViewRevision,
  selectedId,
}: RevisionHistoryProps): React.ReactNode {
  if (revisions.length === 0) {
    return (
      <div
        className="py-4 text-center text-sm text-muted-foreground"
        data-testid="revision-history-empty"
      >
        No revision history.
      </div>
    );
  }

  return (
    <div data-testid="revision-history">
      <div className="mb-2 flex items-center gap-2">
        <History className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-semibold text-foreground">
          Revision history ({revisions.length})
        </h3>
      </div>
      <div className="divide-y divide-border rounded-lg border border-border">
        {revisions.map((rev) => (
          <button
            key={rev.id}
            type="button"
            onClick={() => onViewRevision?.(rev)}
            data-testid={`revision-item-${rev.id}`}
            className={`flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-accent/50 ${selectedId === rev.id ? "bg-accent" : ""}`}
          >
            <div>
              <span
                className="font-medium text-foreground"
                data-testid={`revision-version-${rev.id}`}
              >
                v{rev.version}
              </span>
              <span className="ml-2 text-xs text-muted-foreground">
                {formatDate(rev.created_at)}
              </span>
            </div>
            <span
              className="text-xs text-muted-foreground"
              data-testid={`revision-editor-${rev.id}`}
            >
              {rev.editor_id}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
