"use client";

import { useState } from "react";
import { Clock, Eye } from "lucide-react";
import { cn } from "@/lib/utils";

export interface Revision {
  id: string;
  version: number;
  editorId: string;
  previousContent: string;
  createdAt: string;
}

export interface RevisionHistoryProps {
  /** List of revisions, newest first. */
  revisions: Revision[];
  /** Called when user selects a revision to view. */
  onSelect?: (revision: Revision) => void;
}

/** Format an ISO timestamp for display. */
function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

/** Revision history panel showing a list of versions with content preview. */
export function RevisionHistory({ revisions, onSelect }: RevisionHistoryProps): React.ReactNode {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  if (revisions.length === 0) {
    return (
      <p className="py-4 text-center text-sm text-muted-foreground" data-testid="no-revisions">
        No revision history.
      </p>
    );
  }

  const selectedRevision = revisions.find((r) => r.id === selectedId);

  const handleSelect = (rev: Revision): void => {
    setSelectedId(rev.id === selectedId ? null : rev.id);
    onSelect?.(rev);
  };

  return (
    <div data-testid="revision-history" className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <Clock className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-semibold text-foreground">
          Revision History ({revisions.length})
        </h3>
      </div>

      {/* Revision list */}
      <div className="flex flex-col gap-1" data-testid="revision-list">
        {revisions.map((rev) => (
          <button
            key={rev.id}
            data-testid={`revision-${rev.id}`}
            onClick={() => handleSelect(rev)}
            className={cn(
              "flex items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-accent",
              selectedId === rev.id && "bg-accent",
            )}
          >
            <Eye className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <span className="font-medium text-foreground">v{rev.version}</span>
            <span className="text-xs text-muted-foreground">by {rev.editorId}</span>
            <span className="ml-auto text-xs text-muted-foreground">
              {formatDate(rev.createdAt)}
            </span>
          </button>
        ))}
      </div>

      {/* Content preview */}
      {selectedRevision && (
        <div
          data-testid="revision-content"
          className="rounded-lg border border-border bg-muted/30 p-3"
        >
          <h4 className="mb-2 text-xs font-semibold text-muted-foreground">
            v{selectedRevision.version} — Previous content
          </h4>
          <pre className="whitespace-pre-wrap text-sm text-foreground">
            {selectedRevision.previousContent}
          </pre>
        </div>
      )}
    </div>
  );
}
