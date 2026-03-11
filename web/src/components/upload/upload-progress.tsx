"use client";

import { X } from "lucide-react";

export interface UploadProgressItem {
  id: string;
  filename: string;
  progress: number;
  error?: string;
}

export interface UploadProgressProps {
  items: UploadProgressItem[];
  onCancel?: (id: string) => void;
}

/** Format file progress as a percentage string. */
export function formatProgress(progress: number): string {
  return `${Math.min(100, Math.max(0, Math.round(progress)))}%`;
}

/** Upload progress indicator with filename, percentage, and optional cancel. */
export function UploadProgress({ items, onCancel }: UploadProgressProps): React.ReactNode {
  if (items.length === 0) return null;

  return (
    <div className="space-y-2" data-testid="upload-progress">
      {items.map((item) => (
        <div
          key={item.id}
          className="flex items-center gap-3 rounded-md border border-border bg-background p-2"
          data-testid={`upload-item-${item.id}`}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center justify-between">
              <span
                className="truncate text-xs font-medium text-foreground"
                data-testid={`upload-filename-${item.id}`}
              >
                {item.filename}
              </span>
              <span
                className="ml-2 text-xs text-muted-foreground"
                data-testid={`upload-percent-${item.id}`}
              >
                {item.error ? "Failed" : formatProgress(item.progress)}
              </span>
            </div>
            {!item.error && (
              <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-primary transition-all"
                  style={{ width: formatProgress(item.progress) }}
                  data-testid={`upload-bar-${item.id}`}
                />
              </div>
            )}
            {item.error && (
              <p
                className="mt-0.5 text-xs text-destructive"
                data-testid={`upload-error-${item.id}`}
              >
                {item.error}
              </p>
            )}
          </div>
          {onCancel && item.progress < 100 && !item.error && (
            <button
              type="button"
              onClick={() => onCancel(item.id)}
              aria-label={`Cancel upload ${item.filename}`}
              data-testid={`upload-cancel-${item.id}`}
              className="rounded-md p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      ))}
    </div>
  );
}
